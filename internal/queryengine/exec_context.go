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
}

// rememberFindOne records the document findOne is about to return.
func (ec *execContext) rememberFindOne(obj *goja.Object, result models.QueryResult) {
	ec.lastFindOne = &findOneResult{obj: obj, result: result}
}

// isLastFindOne reports whether val is the document findOne last returned.
func (ec *execContext) isLastFindOne(val goja.Value) bool {
	return ec.lastFindOne != nil && val == ec.lastFindOne.obj
}

// findOneDisplay returns the server's result for the document findOne last
// returned, given as raw, the script's final value exported, when the script
// has not changed it. Call it only when isLastFindOne. Shown as the JS object
// instead, the document would lose types JS cannot hold: a whole-number double
// reads back as a plain integer and is shown as an int32.
func (ec *execContext) findOneDisplay(raw any) (models.QueryResult, bool) {
	last := ec.lastFindOne
	// Compared in display form: exported BSON wrappers hold Go funcs, which
	// never compare equal, while their display form is plain Extended JSON.
	handedOut := exportValue(toGojaValue(ec.rt, last.result))
	if !reflect.DeepEqual(displayValue(raw), displayValue(handedOut)) {
		return models.QueryResult{}, false
	}
	return last.result, true
}
