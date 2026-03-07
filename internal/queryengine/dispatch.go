package queryengine

import (
	"context"
	"fmt"
	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
)

// dispatch executes a captured operation against MongoDB.
// This is a placeholder — full implementation comes in Task 6.
func dispatch(ctx context.Context, client *mongo.Client, dbName string, op CapturedOp) (models.QueryResult, error) {
	return models.QueryResult{}, fmt.Errorf("dispatch not yet implemented for method: %s", op.Method)
}
