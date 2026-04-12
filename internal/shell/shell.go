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
}

// CheckMongosh returns true if mongosh is available in PATH.
func CheckMongosh() bool {
	_, err := exec.LookPath("mongosh")
	return err == nil
}

// wrapQuery wraps the user's query in JavaScript that converts the result to
// JSON. It handles cursors (via toArray()), plain objects, and falls back to
// string output for non-serializable results.
//
// The query is placed inside an IIFE with "return" prepended to the last
// top-level statement, so its value is captured and serialized. The start of
// the last statement is found by scanning the source while tracking bracket
// depth, strings, and comments — so multi-line expressions are handled.
func wrapQuery(query string) string {
	body := prependReturnToLastStatement(strings.TrimSpace(query))

	return fmt.Sprintf(`
const __result = (() => { %s })();
const __val = (typeof __result?.toArray === 'function') ? __result.toArray() : __result;
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
`, body)
}

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

	wrapped := wrapQuery(query)

	tmpFile, err := os.CreateTemp("", "vervet-query-*.js")
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.WriteString(wrapped); err != nil {
		tmpFile.Close()
		return models.QueryResult{}, fmt.Errorf("failed to write query file: %w", err)
	}
	tmpFile.Close()

	args := buildArgs(uri, tmpFile.Name())
	cmd := exec.CommandContext(ctx, "mongosh", args...)

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

	return parseOutput(stdout.String()), nil
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
