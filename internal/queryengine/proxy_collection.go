package queryengine

import (
	"strconv"

	"vervet/internal/models"

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

// findAndModifyMethods give a script the document, or null when none matched.
var findAndModifyMethods = map[string]bool{
	"findOneAndDelete":  true,
	"findOneAndReplace": true,
	"findOneAndUpdate":  true,
}

// withCursorMethods adds the cursor methods mongosh scripts use to a result that
// has already been read into an array. They are not enumerable, so the array still
// looks like an array. Reading through them consumes it as a cursor would, while
// indexing and the Array methods see every document.
func withCursorMethods(rt *goja.Runtime, method string, val goja.Value) goja.Value {
	if !arrayCursorMethods[method] {
		return val
	}
	obj, ok := val.(*goja.Object)
	if !ok || obj.ClassName() != "Array" {
		return val
	}
	length := int(obj.Get("length").ToInteger())
	index := 0
	closed := false
	next := func() goja.Value {
		if closed || index >= length {
			return goja.Null()
		}
		doc := obj.Get(strconv.Itoa(index))
		index++
		return doc
	}
	remaining := func() int {
		if closed {
			return 0
		}
		return length - index
	}
	define := func(name string, fn func(goja.FunctionCall) goja.Value) {
		_ = obj.DefineDataProperty(name, rt.ToValue(fn), goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_FALSE)
	}
	define("toArray", func(goja.FunctionCall) goja.Value {
		if index == 0 && !closed {
			index = length
			return obj
		}
		rest := make([]any, 0, remaining())
		for remaining() > 0 {
			rest = append(rest, next())
		}
		return rt.NewArray(rest...)
	})
	define("next", func(goja.FunctionCall) goja.Value { return next() })
	define("tryNext", func(goja.FunctionCall) goja.Value { return next() })
	define("hasNext", func(goja.FunctionCall) goja.Value { return rt.ToValue(remaining() > 0) })
	define("itcount", func(goja.FunctionCall) goja.Value {
		n := remaining()
		index = length
		return rt.ToValue(n)
	})
	define("objsLeftInBatch", func(goja.FunctionCall) goja.Value { return rt.ToValue(remaining()) })
	define("isExhausted", func(goja.FunctionCall) goja.Value { return rt.ToValue(remaining() == 0) })
	define("isClosed", func(goja.FunctionCall) goja.Value { return rt.ToValue(remaining() == 0) })
	define("close", func(goja.FunctionCall) goja.Value {
		closed = true
		return goja.Undefined()
	})
	define("pretty", func(goja.FunctionCall) goja.Value { return obj })
	// Array's map, whose result stays a cursor so that calls can keep chaining.
	define("map", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(rt.NewTypeError("map argument must be a function"))
		}
		mapped := make([]any, length)
		for i := range length {
			val, err := fn(call.Argument(1), obj.Get(strconv.Itoa(i)), rt.ToValue(i), obj)
			if err != nil {
				panic(err)
			}
			mapped[i] = val
		}
		return withCursorMethods(rt, method, rt.NewArray(mapped...))
	})
	// A callback that returns false stops the loop, unlike Array's forEach.
	define("forEach", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(rt.NewTypeError("forEach argument must be a function"))
		}
		for remaining() > 0 {
			result, err := fn(goja.Undefined(), next())
			if err != nil {
				panic(err)
			}
			if result.StrictEquals(rt.ToValue(false)) {
				break
			}
		}
		return goja.Undefined()
	})
	return obj
}

// runEager dispatches a collection method called from a script, throwing a
// JS error if it fails.
func runEager(ec *execContext, collName, method string, call goja.FunctionCall) models.QueryResult {
	requireClient(ec)
	op := CapturedOp{Collection: collName, Method: method, Args: exportArgs(call)}
	result, err := dispatch(ec.ctx, ec.client, ec.dbName, op)
	if err != nil {
		panic(ec.rt.NewGoError(err))
	}
	return result
}

// newCollectionProxy creates a Goja object with methods for each supported
// MongoDB operation. Write methods and findOne execute eagerly via dispatch(). find
// returns a lazyCursor for deferred execution.
func newCollectionProxy(ec *execContext, collName string) goja.Value {
	obj := ec.rt.NewObject()

	// find — returns lazyCursor, or the plan when its explain option is set
	_ = obj.Set("find", func(call goja.FunctionCall) goja.Value {
		rejectNullFilter(ec, call)
		op, err := parseFindArgs(collName, "find", exportArgs(call))
		if err != nil {
			panic(ec.rt.NewGoError(err))
		}
		if op.Explain != "" {
			requireClient(ec)
			result, err := explainFindOp(ec.ctx, ec.client, ec.dbName, op)
			if err != nil {
				panic(ec.rt.NewGoError(err))
			}
			return toGojaValue(ec.rt, result)
		}
		return newLazyCursor(ec, op).toGojaObject()
	})

	// findOne — runs immediately and returns the document, or null when none
	// matches, as in mongosh, so scripts can read fields from it and pass it on.
	_ = obj.Set("findOne", func(call goja.FunctionCall) goja.Value {
		rejectNullFilter(ec, call)
		result := runEager(ec, collName, "findOne", call)
		if len(result.Documents) == 0 {
			return goja.Null()
		}
		val := toGojaValue(ec.rt, result)
		ec.rememberRead(val, result)
		return val
	})

	// Eager methods — execute immediately, return real results
	for _, method := range eagerMethods {
		m := method
		_ = obj.Set(m, func(call goja.FunctionCall) goja.Value {
			result := runEager(ec, collName, m, call)
			if findAndModifyMethods[m] && len(result.Documents) == 0 {
				return goja.Null()
			}
			val := withCursorMethods(ec.rt, m, toGojaValue(ec.rt, result))
			if m == "aggregate" && result.Namespace != nil {
				ec.rememberRead(val, result)
			}
			return val
		})
	}

	setCollectionInfoMethods(obj, ec, collName)

	return obj
}
