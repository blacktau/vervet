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
//
// A one-line preamble before the script frames its print/console calls, and
// the epilogue frames the result, so the two can be told apart in stdout (see
// output.go). The preamble keeps the script starting on line 2, which
// remapError counts on.
//
// It also returns where the capture prefix went, which remapError needs to
// correct positions on that line.
func wrapQuery(query, nonce string) (string, capturePos) {
	trimmed := strings.TrimSpace(query)
	body := prependToLastStatement(trimmed, capturePrefix)
	capture := capturePosition(trimmed, body)

	return fmt.Sprintf(`%s
%s
;(function () {
  const __result = globalThis.%s;
  if (__result === undefined) { return; }
  const __val = (__result !== null && typeof __result.toArray === 'function') ? __result.toArray() : __result;
  const __print = globalThis.__vervetPrint;
  let __out;
  try {
    __out = EJSON.stringify(__val, {relaxed: false});
  } catch(_e) {
    try {
      __out = JSON.stringify(__val);
    } catch(_e2) {
      __out = String(__val);
    }
  }
  __print(%q + "R");
  __print(__out);
})();
`, outputPreamble(nonce), body, resultGlobal, frameMarker(nonce)), capture
}

// capturePos is the user line (1-based) and column (0-based) where wrapQuery
// inserted the capture prefix. line is 0 when it inserted none.
type capturePos struct {
	line, col int
}

// capturePosition finds where prependToLastStatement put the prefix in body,
// the wrapped form of trimmed.
func capturePosition(trimmed, body string) capturePos {
	if body == trimmed {
		return capturePos{}
	}
	start := lastStatementStart(trimmed)
	before := trimmed[:start]
	return capturePos{
		line: strings.Count(before, "\n") + 1,
		col:  start - (strings.LastIndexByte(before, '\n') + 1),
	}
}

// resultGlobal holds the query's value between the user's script and the
// serialising epilogue. It is a globalThis property rather than a declaration
// so assigning it never collides with the script's own bindings.
const resultGlobal = "globalThis.__vervetResult"

// Execute runs a one-shot mongosh --eval command against the given URI.
// The user's query is wrapped in JavaScript that converts the result to JSON.
// If the output is valid JSON, it is returned as structured documents;
// otherwise the raw text is returned. What the script printed is returned in
// Output, on failure too.
func Execute(ctx context.Context, uri string, query string, cfg Config) (models.QueryResult, error) {
	return runMongosh(ctx, query, cfg, true, func(queryFile string) []string {
		return buildArgs(uri, queryFile)
	})
}

// runMongosh writes the wrapped query to a file, runs mongosh with the args
// built for that file, and splits its output into the result and what the
// script printed. remap rewrites error positions to the user's own lines.
func runMongosh(ctx context.Context, query string, cfg Config, remap bool, args func(queryFile string) []string) (models.QueryResult, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	nonce, err := newNonce()
	if err != nil {
		return models.QueryResult{}, err
	}

	wrapped, capture := wrapQuery(query, nonce)
	queryFile, cleanup, err := writeQueryFile(wrapped, cfg.ScriptDir)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, "mongosh", args(queryFile)...)
	cmd.Dir = cfg.ScriptDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		return buildResult(stdout.String(), stderr.String(), nonce), nil
	}

	if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
		return models.QueryResult{}, ErrShellNotFound
	}

	// What the script printed is kept on every failure, a timeout or cancel
	// included: progress lines show where a long script stalled.
	result, errMsg := failedRun(stdout.String(), stderr.String(), nonce)
	if ctx.Err() == context.DeadlineExceeded {
		return result, ErrQueryTimeout
	}
	if ctx.Err() == context.Canceled {
		return result, ctx.Err()
	}
	if errMsg == "" {
		return result, fmt.Errorf("mongosh exited with: %w", err)
	}
	if remap {
		errMsg = remapError(errMsg, query, capture)
	}
	return result, fmt.Errorf("%s", errMsg)
}

// parseResult parses the script's serialised final value: JSON documents when
// it is JSON, otherwise text.
func parseResult(output string) models.QueryResult {
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
