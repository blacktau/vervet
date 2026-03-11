package queryengine

import (
	"context"
	"vervet/internal/models"
	"vervet/internal/shell"
)

// ShellEngine implements QueryEngine by delegating to mongosh via the shell package.
type ShellEngine struct {
	cfg shell.Config
}

// NewShellEngine creates a ShellEngine with the given shell configuration.
func NewShellEngine(cfg shell.Config) *ShellEngine {
	return &ShellEngine{cfg: cfg}
}

// ExecuteQuery runs the query against MongoDB using mongosh.
func (e *ShellEngine) ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error) {
	return shell.Execute(ctx, uri, query, e.cfg)
}
