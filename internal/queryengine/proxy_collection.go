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
			return rt.ToValue(op)
		})
	}

	return obj
}
