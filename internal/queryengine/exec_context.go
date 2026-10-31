package queryengine

import (
	"context"
	"reflect"

	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// execContext holds the dependencies needed by proxy methods to execute
// MongoDB operations during Goja script execution.
type execContext struct {
	ctx      context.Context
	client   *mongo.Client
	dbName   string
	rt       *goja.Runtime
	pageSize int64
	// lastFindOne is the document findOne last handed to the script (see
	// findOneDisplay).
	lastFindOne *findOneResult
}

// findOneResult pairs the document findOne gave a script with the server's
// result it came from.
type findOneResult struct {
	obj    *goja.Object
	result models.QueryResult
	// snapshot is obj as exported when handed out, to tell if the script
	// has changed it since.
	snapshot any
}

// rememberFindOne records the document findOne is about to return.
func (ec *execContext) rememberFindOne(obj *goja.Object, result models.QueryResult) {
	ec.lastFindOne = &findOneResult{obj: obj, result: result, snapshot: exportValue(obj)}
}

// findOneDisplay returns the server's result for val when val is the document
// findOne last returned and the script has not changed it. Shown as the JS
// object instead, the document would lose types JS cannot hold: a
// whole-number double reads back as a plain integer and is shown as an int32.
func (ec *execContext) findOneDisplay(val goja.Value) (models.QueryResult, bool) {
	last := ec.lastFindOne
	if last == nil || val != last.obj {
		return models.QueryResult{}, false
	}
	if !reflect.DeepEqual(exportValue(val), last.snapshot) {
		return models.QueryResult{}, false
	}
	return last.result, true
}
