package queryengine

import (
	"context"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/mongo"
)

// execContext holds the dependencies needed by proxy methods to execute
// MongoDB operations during Goja script execution.
type execContext struct {
	ctx    context.Context
	client *mongo.Client
	dbName string
	rt     *goja.Runtime
}
