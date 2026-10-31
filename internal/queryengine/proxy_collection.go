package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
)

// eagerMethods are methods that execute immediately and return real results.
var eagerMethods = []string{
	"insertOne", "insertMany",
	"updateOne", "updateMany",
	"deleteOne", "deleteMany",
	"replaceOne",
	"countDocuments", "aggregate", "distinct",
	"findOneAndDelete", "findOneAndReplace", "findOneAndUpdate",
	"estimatedDocumentCount", "bulkWrite", "drop",
	"createIndex", "createIndexes", "dropIndex", "dropIndexes", "listIndexes",
}

// arrayCursorMethods are the eager methods mongosh returns a cursor from. They
// execute immediately here and yield an array, so the cursor terminals scripts
// chain onto them (`db.c.aggregate([...]).toArray()`) are added to that array.
var arrayCursorMethods = map[string]bool{
	"aggregate":   true,
	"listIndexes": true,
}

// withCursorMethods adds toArray to an already-materialised array result.
func withCursorMethods(rt *goja.Runtime, method string, val goja.Value) goja.Value {
	if !arrayCursorMethods[method] {
		return val
	}
	obj, ok := val.(*goja.Object)
	if !ok || obj.ClassName() != "Array" {
		return val
	}
	_ = obj.Set("toArray", func() goja.Value { return obj })
	return obj
}

// newCollectionProxy creates a Goja object with methods for each supported
// MongoDB operation. Write methods and findOne execute eagerly via dispatch(). find
// returns a lazyCursor for deferred execution.
func newCollectionProxy(ec *execContext, collName string) goja.Value {
	obj := ec.rt.NewObject()

	// find — returns lazyCursor
	_ = obj.Set("find", func(call goja.FunctionCall) goja.Value {
		args := exportArgs(call)
		cursor := &lazyCursor{
			ec:         ec,
			collection: collName,
		}
		if len(args) > 0 {
			cursor.filter = args[0]
		}
		if len(args) > 1 {
			cursor.projection = args[1]
		}
		return cursor.toGojaObject()
	})

	// findOne — runs immediately and returns the document, or null when none
	// matches, as in mongosh, so scripts can read fields from it and pass it on.
	_ = obj.Set("findOne", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		op := CapturedOp{Collection: collName, Method: "findOne", Args: exportArgs(call)}
		result, err := dispatch(ec.ctx, ec.client, ec.dbName, op)
		if err != nil {
			panic(ec.rt.NewGoError(err))
		}
		if len(result.Documents) == 0 {
			return goja.Null()
		}
		val := toGojaValue(ec.rt, result)
		if obj, ok := val.(*goja.Object); ok {
			ec.rememberFindOne(obj, result)
		}
		return val
	})

	// Eager methods — execute immediately, return real results
	for _, method := range eagerMethods {
		m := method
		_ = obj.Set(m, func(call goja.FunctionCall) goja.Value {
			if ec.client == nil {
				panic(ec.rt.NewGoError(fmt.Errorf("no MongoDB client available")))
			}
			args := exportArgs(call)
			op := CapturedOp{
				Collection: collName,
				Method:     m,
				Args:       args,
			}
			result, err := dispatch(ec.ctx, ec.client, ec.dbName, op)
			if err != nil {
				panic(ec.rt.NewGoError(err))
			}
			return withCursorMethods(ec.rt, m, toGojaValue(ec.rt, result))
		})
	}

	setCollectionInfoMethods(obj, ec, collName)

	return obj
}
