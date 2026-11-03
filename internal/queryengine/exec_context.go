package queryengine

import (
	"context"
	"math"
	"reflect"
	"time"

	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// maxRememberedReads bounds what a script looping over reads keeps alive. A value
// that has fallen off the list is still shown, but as a plain script value.
const maxRememberedReads = 16

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
	// reads are the values reads last handed to the script (see readDisplay). They
	// are kept on the script's first context, whichever database was read.
	reads []readResult
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
	obj, ok := val.(*goja.Object)
	if !ok {
		return
	}
	root := ec.root()
	root.reads = append(root.reads, readResult{obj: obj, result: result})
	if len(root.reads) > maxRememberedReads {
		root.reads = root.reads[1:]
	}
}

// rememberedRead returns the read that handed val to the script, if any.
func (ec *execContext) rememberedRead(val goja.Value) *readResult {
	reads := ec.root().reads
	for i := range reads {
		if val == reads[i].obj {
			return &reads[i]
		}
	}
	return nil
}

// readDisplay returns the server's result for a value a read returned, given as
// raw, the script's final value exported, when the script has not changed it.
// Shown as the JS value instead, the documents would lose types JS cannot hold,
// such as a whole-number double reading back as an int32, and nothing would say
// where they are stored.
func (ec *execContext) readDisplay(read *readResult, raw any) (models.QueryResult, bool) {
	if !ec.readUnchanged(read.result, raw) {
		return models.QueryResult{}, false
	}
	return read.result, true
}

// readUnchanged compares a list one document at a time, so that a large result
// isn't held in memory as a second full copy.
func (ec *execContext) readUnchanged(result models.QueryResult, raw any) bool {
	docs, isList := raw.([]any)
	if !isList || result.Single {
		return sameExported(raw, exportValue(toGojaValue(ec.rt, result)))
	}
	if len(docs) != len(result.Documents) {
		return false
	}
	for i, doc := range docs {
		if !sameExported(doc, exportValue(toJSValue(ec.rt, result.Documents[i]))) {
			return false
		}
	}
	return true
}

// sameExported reports whether two values exported from a script are the same.
// BSON wrappers are compared by the value they hold: they also hold Go funcs,
// which never compare equal.
func sameExported(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		if held, isWrapper := x["__bsonValue"]; isWrapper {
			return reflect.DeepEqual(unwrapBSON(held), unwrapBSON(y["__bsonValue"]))
		}
		for k, v := range x {
			other, ok := y[k]
			if !ok || !sameExported(v, other) {
				return false
			}
		}
		return true
	case bson.D:
		y, ok := b.(bson.D)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if x[i].Key != y[i].Key || !sameExported(x[i].Value, y[i].Value) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !sameExported(x[i], y[i]) {
				return false
			}
		}
		return true
	case float64:
		y, ok := b.(float64)
		return ok && (x == y || (math.IsNaN(x) && math.IsNaN(y)))
	case time.Time:
		y, ok := b.(time.Time)
		return ok && x.Equal(y)
	}
	return reflect.DeepEqual(a, b)
}

// unwrapBSON gives the value a __bsonValue field holds.
func unwrapBSON(held any) any {
	if w, ok := held.(*bsonWrapper); ok {
		return w.Value
	}
	return held
}
