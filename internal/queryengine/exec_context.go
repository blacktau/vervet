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
	// parent is the context this one was made from by getSiblingDB, if any.
	parent *execContext
	// lastRead is the value a read last handed to the script (see readDisplay).
	// It is kept on the script's first context, whichever database was read.
	lastRead *readResult
}

// readResult pairs a value a read gave a script with the server's result it
// came from.
type readResult struct {
	obj    *goja.Object
	result models.QueryResult
}

// sibling makes the context for another database in the same script.
func (ec *execContext) sibling(dbName string) *execContext {
	return &execContext{
		ctx:      ec.ctx,
		client:   ec.client,
		dbName:   dbName,
		rt:       ec.rt,
		pageSize: ec.pageSize,
		parent:   ec.root(),
	}
}

func (ec *execContext) root() *execContext {
	if ec.parent != nil {
		return ec.parent
	}
	return ec
}

// rememberRead records the value a read is about to return.
func (ec *execContext) rememberRead(val goja.Value, result models.QueryResult) {
	if obj, ok := val.(*goja.Object); ok {
		ec.root().lastRead = &readResult{obj: obj, result: result}
	}
}

// isLastRead reports whether val is the value a read last returned.
func (ec *execContext) isLastRead(val goja.Value) bool {
	last := ec.root().lastRead
	return last != nil && val == last.obj
}

// readDisplay returns the server's result for the value a read last returned,
// given as raw, the script's final value exported, when the script has not
// changed it. Call it only when isLastRead. Shown as the JS value instead, the
// documents would lose types JS cannot hold, such as a whole-number double
// reading back as an int32, and nothing would say where they are stored.
func (ec *execContext) readDisplay(raw any) (models.QueryResult, bool) {
	last := ec.root().lastRead
	// Compared in display form: exported BSON wrappers hold Go funcs, which
	// never compare equal, while their display form is plain Extended JSON.
	handedOut := exportValue(toGojaValue(ec.rt, last.result))
	if !reflect.DeepEqual(displayValue(raw), displayValue(handedOut)) {
		return models.QueryResult{}, false
	}
	return last.result, true
}
