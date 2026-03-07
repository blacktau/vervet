package queryengine

import (
	"context"
	"vervet/internal/models"
)

// QueryEngine defines the interface for executing MongoDB queries.
// Implementations can use different execution strategies (e.g. mongosh subprocess, goja JS engine).
type QueryEngine interface {
	ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error)
}
