package queryengine

import (
	"vervet/internal/models"

	"github.com/dop251/goja"
)

// toGojaValue converts a QueryResult into a Goja-native value so scripts
// can use query results as real JavaScript data.
func toGojaValue(rt *goja.Runtime, result models.QueryResult) goja.Value {
	if result.RawOutput != "" {
		return rt.ToValue(result.RawOutput)
	}
	return rt.ToValue(result.Documents)
}

// exportArgs converts Goja function call arguments to plain Go values.
func exportArgs(call goja.FunctionCall) []any {
	args := make([]any, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	return args
}

// extractLazyCursor checks if a Goja value wraps an unresolved lazyCursor.
// Returns nil if the value is not a cursor.
func extractLazyCursor(val goja.Value) *lazyCursor {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}
	if obj, ok := val.(*goja.Object); ok {
		inner := obj.Get("__lazyCursor")
		if inner != nil && !goja.IsUndefined(inner) {
			if cursor, ok := inner.Export().(*lazyCursor); ok {
				return cursor
			}
		}
	}
	return nil
}
