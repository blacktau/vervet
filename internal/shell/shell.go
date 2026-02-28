package shell

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
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

// Execute runs a one-shot mongosh --eval command against the given URI.
// It spawns a new mongosh process per call, captures stdout+stderr,
// and enforces a timeout via context.
func Execute(ctx context.Context, uri string, query string, cfg Config) (string, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	args := buildArgs(uri, query, cfg)
	cmd := exec.CommandContext(ctx, "mongosh", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", ErrQueryTimeout
		}
		if ctx.Err() == context.Canceled {
			return "", ctx.Err()
		}
		// Check if mongosh is not found
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return "", ErrShellNotFound
		}
		// mongosh returned a non-zero exit code — return the actual error output.
		// mongosh may write errors to stderr, stdout, or both.
		errMsg := stderr.String() + stdout.String()
		if len(errMsg) > 0 {
			return "", fmt.Errorf("%s", errMsg)
		}
		return "", fmt.Errorf("mongosh exited with: %w", err)
	}

	return stdout.String(), nil
}

func buildArgs(uri string, query string, cfg Config) []string {
	return []string{
		uri,
		"--quiet",
		"--norc",
		"--eval", query,
	}
}
