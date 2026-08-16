package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"vervet/internal/models"
)

var (
	ErrShellNotFound = fmt.Errorf("mongosh not found in PATH")
	ErrQueryTimeout  = fmt.Errorf("query timed out")
)

// Config holds settings for mongosh execution.
type Config struct {
	Timeout time.Duration
	// ScriptDir is the directory the query was saved in, empty for an unsaved
	// tab. The temp script handed to mongosh is written there and mongosh runs
	// with it as its working directory, so the script's __dirname, load() and
	// relative paths all point at the user's own directory rather than /tmp.
	ScriptDir string
}

// writeQueryFile writes the wrapped query to a temp file in dir (the system
// temp directory when dir is empty) and returns its path with a cleanup func.
func writeQueryFile(wrapped, dir string) (string, func(), error) {
	tmpFile, err := os.CreateTemp(dir, "vervet-query-*.js")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	remove := func() { _ = os.Remove(tmpFile.Name()) }

	if _, err := tmpFile.WriteString(wrapped); err != nil {
		tmpFile.Close()
		remove()
		return "", nil, fmt.Errorf("failed to write query file: %w", err)
	}
	tmpFile.Close()

	return tmpFile.Name(), remove, nil
}

// CheckMongosh returns true if mongosh is available in PATH.
func CheckMongosh() bool {
	_, err := exec.LookPath("mongosh")
	return err == nil
}

// wrapQuery appends JavaScript that converts the query's value to JSON. It
// handles cursors (via toArray()), plain objects, and falls back to string
// output for non-serializable results.
//
// The value is captured by assigning the last top-level statement to a global.
// The query itself stays at top level and is NOT wrapped in a function: mongosh
// evaluates scripts at top level, and wrapping changes their meaning — a script
// opening with `const db = db.getSiblingDB(name)` throws "Cannot access 'db'
// before initialization" inside a function body but runs fine at top level.
//
// The start of the last statement is found by scanning the source while
// tracking bracket depth, strings, and comments — so multi-line expressions are
// handled. When that statement is a declaration or other non-expression, no
// value is captured and only what the script printed is returned.
func wrapQuery(query string) string {
	body := prependToLastStatement(strings.TrimSpace(query), resultGlobal+" = ")

	return fmt.Sprintf(`%s
;(function () {
  const __result = globalThis.%s;
  if (__result === undefined) { return; }
  const __val = (__result !== null && typeof __result.toArray === 'function') ? __result.toArray() : __result;
  try {
    const __json = EJSON.stringify(__val, {relaxed: false});
    print(__json);
  } catch(_e) {
    try {
      const __json2 = JSON.stringify(__val);
      print(__json2);
    } catch(_e2) {
      print(String(__val));
    }
  }
})();
`, body, resultGlobal)
}

// resultGlobal holds the query's value between the user's script and the
// serialising epilogue. It is a globalThis property rather than a declaration
// so assigning it never collides with the script's own bindings.
const resultGlobal = "globalThis.__vervetResult"

// Execute runs a one-shot mongosh --eval command against the given URI.
// The user's query is wrapped in JavaScript that converts the result to JSON.
// If the output is valid JSON, it is returned as structured documents;
// otherwise the raw text is returned.
func Execute(ctx context.Context, uri string, query string, cfg Config) (models.QueryResult, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	queryFile, cleanup, err := writeQueryFile(wrapQuery(query), cfg.ScriptDir)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer cleanup()

	args := buildArgs(uri, queryFile)
	cmd := exec.CommandContext(ctx, "mongosh", args...)
	cmd.Dir = cfg.ScriptDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return models.QueryResult{}, ErrQueryTimeout
		}
		if ctx.Err() == context.Canceled {
			return models.QueryResult{}, ctx.Err()
		}
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return models.QueryResult{}, ErrShellNotFound
		}
		errMsg := stderr.String() + stdout.String()
		if len(errMsg) > 0 {
			return models.QueryResult{}, fmt.Errorf("%s", remapError(errMsg, query))
		}
		return models.QueryResult{}, fmt.Errorf("mongosh exited with: %w", err)
	}

	return withStderr(parseOutput(stdout.String()), stderr.String()), nil
}

// withStderr folds mongosh's stderr into the result. Scripts routinely send
// summaries and warnings there with console.error, and Vervet has a single
// output pane, so dropping it loses output the user asked for. Structured
// results are left alone rather than polluted with text.
func withStderr(result models.QueryResult, stderr string) models.QueryResult {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" || len(result.Documents) > 0 {
		return result
	}
	if result.RawOutput == "" {
		result.RawOutput = stderr
		return result
	}
	result.RawOutput = result.RawOutput + "\n" + stderr
	return result
}

// parseOutput attempts to parse mongosh output as JSON documents.
func parseOutput(output string) models.QueryResult {
	output = strings.TrimSpace(output)
	if output == "" {
		return models.QueryResult{}
	}

	// Try parsing as a single JSON array
	var arr []any
	if err := json.Unmarshal([]byte(output), &arr); err == nil {
		return models.QueryResult{Documents: arr}
	}

	// Try parsing as a single JSON object
	var single map[string]any
	if err := json.Unmarshal([]byte(output), &single); err == nil {
		return models.QueryResult{Documents: []any{single}}
	}

	// Try parsing as newline-delimited JSON objects
	lines := strings.Split(output, "\n")
	var docs []any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var doc any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return models.QueryResult{RawOutput: output}
		}
		docs = append(docs, doc)
	}

	if len(docs) > 0 {
		return models.QueryResult{Documents: docs}
	}

	return models.QueryResult{RawOutput: output}
}

func buildArgs(uri string, queryFile string) []string {
	return []string{
		uri,
		"--quiet",
		"--norc",
		"--file", queryFile,
	}
}
