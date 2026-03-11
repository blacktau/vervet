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

// newCollectionProxy creates a Goja object with methods for each supported
// MongoDB operation. Write methods execute eagerly via dispatch(). find/findOne
// return a lazyCursor for deferred execution.
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

	// findOne — returns lazyCursor with isFindOne flag
	_ = obj.Set("findOne", func(call goja.FunctionCall) goja.Value {
		args := exportArgs(call)
		cursor := &lazyCursor{
			ec:         ec,
			collection: collName,
			isFindOne:  true,
		}
		if len(args) > 0 {
			cursor.filter = args[0]
		}
		return cursor.toGojaObject()
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
			return toGojaValue(ec.rt, result)
		})
	}

	return obj
}
