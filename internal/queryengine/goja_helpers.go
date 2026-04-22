package queryengine

import (
	"fmt"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// toGojaValue converts a QueryResult into a Goja-native value so scripts
// can use query results as real JavaScript data.
func toGojaValue(rt *goja.Runtime, result models.QueryResult) goja.Value {
	if result.RawOutput != "" {
		return rt.ToValue(result.RawOutput)
	}
	if result.Single && len(result.Documents) > 0 {
		return rt.ToValue(result.Documents[0])
	}
	return rt.ToValue(result.Documents)
}

// exportArgs converts Goja function call arguments to plain Go values.
// RegExp objects are converted to primitive.Regex so they survive Export()
// and flow through convertToBson correctly.
func exportArgs(call goja.FunctionCall) []any {
	args := make([]any, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = exportValue(arg)
	}
	return args
}

// exportValue converts a single goja.Value to a Go value, recursively walking
// objects and arrays to convert RegExp values to wrapped primitive.Regex before
// they are lost to Export().
func exportValue(val goja.Value) any {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return val.Export()
	}
	obj, ok := val.(*goja.Object)
	if !ok {
		return val.Export()
	}

	switch obj.ClassName() {
	case "RegExp":
		return regexpToBSON(obj)
	case "Array":
		return exportArray(obj)
	case "Object":
		// Already-wrapped BSON values pass through as-is
		if bv := obj.Get("__bsonValue"); bv != nil && !goja.IsUndefined(bv) {
			return val.Export()
		}
		return exportObject(obj)
	default:
		return val.Export()
	}
}

// regexpToBSON converts a goja RegExp object into a BSON-wrapped primitive.Regex.
func regexpToBSON(obj *goja.Object) map[string]any {
	pattern := obj.Get("source").String()
	flags := obj.Get("flags").String()
	return map[string]any{
		"__bsonValue": &bsonWrapper{Value: primitive.Regex{
			Pattern: pattern,
			Options: mongoRegexOptions(flags),
		}},
	}
}

// exportObject walks a goja Object's own properties, recursively exporting
// values so nested RegExp objects are preserved.
func exportObject(obj *goja.Object) map[string]any {
	keys := obj.Keys()
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		result[key] = exportValue(obj.Get(key))
	}
	return result
}

// exportArray walks a goja Array, recursively exporting each element.
func exportArray(obj *goja.Object) []any {
	length := int(obj.Get("length").ToInteger())
	result := make([]any, length)
	for i := 0; i < length; i++ {
		result[i] = exportValue(obj.Get(fmt.Sprintf("%d", i)))
	}
	return result
}

// mongoRegexOptions converts JS regex flags to MongoDB regex options.
// JS flags: g, i, m, s, u, y
// MongoDB options: i, m, s, x, l, u
func mongoRegexOptions(jsFlags string) string {
	var opts string
	for _, f := range jsFlags {
		switch f {
		case 'i', 'm', 's':
			opts += string(f)
		}
	}
	return opts
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
