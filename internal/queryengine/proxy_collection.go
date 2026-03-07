package queryengine

import "github.com/dop251/goja"

var supportedMethods = map[string]bool{
	"find":           true,
	"findOne":        true,
	"insertOne":      true,
	"insertMany":     true,
	"updateOne":      true,
	"updateMany":     true,
	"deleteOne":      true,
	"deleteMany":     true,
	"replaceOne":     true,
	"countDocuments": true,
	"aggregate":      true,
}

// cursorMethods are chainable methods that modify the CapturedOp (limit, skip, sort).
var cursorMethods = map[string]bool{
	"limit": true,
	"skip":  true,
	"sort":  true,
}

// wrapCapturedOp wraps a CapturedOp in a goja object that supports cursor
// modifier chaining (limit, skip, sort). The underlying CapturedOp is stored
// in a hidden __capturedOp property for retrieval during dispatch.
func wrapCapturedOp(rt *goja.Runtime, op *CapturedOp) goja.Value {
	obj := rt.NewObject()
	_ = obj.Set("__capturedOp", op)

	_ = obj.Set("limit", func(n int64) goja.Value {
		op.Limit = n
		return obj
	})
	_ = obj.Set("skip", func(n int64) goja.Value {
		op.Skip = n
		return obj
	})
	_ = obj.Set("sort", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			op.Sort = call.Arguments[0].Export()
		}
		return obj
	})

	return obj
}

// newCollectionProxy creates a goja object with methods for each supported
// MongoDB operation. Each method captures its arguments into a CapturedOp
// rather than executing immediately.
func newCollectionProxy(rt *goja.Runtime, collName string) goja.Value {
	obj := rt.NewObject()

	for method := range supportedMethods {
		m := method
		_ = obj.Set(m, func(call goja.FunctionCall) goja.Value {
			args := make([]any, len(call.Arguments))
			for i, arg := range call.Arguments {
				args[i] = arg.Export()
			}
			op := &CapturedOp{
				Collection: collName,
				Method:     m,
				Args:       args,
			}
			return wrapCapturedOp(rt, op)
		})
	}

	return obj
}
