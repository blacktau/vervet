package shell

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
	"vervet/internal/models"
)

// ExecuteWithOIDC runs a mongosh query using the OIDC auth-code flow.
// This uses the fallback approach: mongosh handles its own OIDC browser flow.
// Since the user already authenticated for the Go driver, the OIDC provider's
// session cookie is still active — mongosh's browser redirect completes
// instantly without user interaction.
func ExecuteWithOIDC(ctx context.Context, uri string, query string, cfg Config) (models.QueryResult, error) {
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

	args := []string{
		uri,
		"--quiet",
		"--norc",
		"--authenticationMechanism", "MONGODB-OIDC",
		"--oidcFlows", "auth-code",
		"--file", queryFile,
	}

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
			return models.QueryResult{}, fmt.Errorf("%s", errMsg)
		}
		return models.QueryResult{}, fmt.Errorf("mongosh exited with: %w", err)
	}

	return parseOutput(stdout.String()), nil
}
