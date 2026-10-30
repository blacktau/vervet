package queryengine

import (
	"fmt"
	"reflect"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// toGojaValue converts a QueryResult into a Goja-native value so scripts
// can use query results as real JavaScript data.
func toGojaValue(rt *goja.Runtime, result models.QueryResult) goja.Value {
	if result.JSValue != nil {
		return toJSValue(rt, result.JSValue)
	}
	if result.RawOutput != "" {
		return rt.ToValue(result.RawOutput)
	}
	if result.Single && len(result.Documents) > 0 {
		return toJSValue(rt, result.Documents[0])
	}
	return toJSValue(rt, result.Documents)
}

// reflectTypeMap is the type goja exports a plain JS object as.
var reflectTypeMap = reflect.TypeOf(map[string]any{})

// exportArgs converts Goja function call arguments to plain Go values.
// RegExp objects are converted to bson.Regex so they survive Export()
// and flow through convertToBson correctly.
func exportArgs(call goja.FunctionCall) []any {
	args := make([]any, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = exportValue(arg)
	}
	return args
}

// exportValue converts a single goja.Value to a Go value, recursively walking
// objects and arrays to convert RegExp values to wrapped bson.Regex before
// they are lost to Export(). Plain objects become an ordered bson.D (see
// exportObject); wrapped BSON values stay as their __bsonValue map.
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
		// Reflected Go values (e.g. a bson.Binary from a db command result)
		// export as themselves; walking their properties would turn them into
		// a map of fields and methods. ExportType answers this without
		// exporting the whole subtree, which Export() would do at every level.
		if obj.ExportType() != reflectTypeMap {
			return obj.Export()
		}
		return exportObject(obj)
	default:
		return val.Export()
	}
}

// regexpToBSON converts a goja RegExp object into a BSON-wrapped bson.Regex.
func regexpToBSON(obj *goja.Object) map[string]any {
	pattern := obj.Get("source").String()
	flags := obj.Get("flags").String()
	return map[string]any{
		"__bsonValue": &bsonWrapper{Value: bson.Regex{
			Pattern: pattern,
			Options: mongoRegexOptions(flags),
		}},
	}
}

// exportObject walks a goja Object's own properties, recursively exporting
// values so nested RegExp objects are preserved. It builds a bson.D rather than
// a map because key order is significant to MongoDB: the first key of a command
// names the command, and sort specs, index keys and embedded-document matches
// all depend on field order. goja's Keys() returns properties in insertion order.
func exportObject(obj *goja.Object) bson.D {
	keys := obj.Keys()
	doc := make(bson.D, 0, len(keys))
	for _, key := range keys {
		doc = append(doc, bson.E{Key: key, Value: exportValue(obj.Get(key))})
	}
	return doc
}

// docField returns the value of key in d, or nil when it is absent.
func docField(d bson.D, key string) any {
	for _, e := range d {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

// asMap returns the fields of an exported document for lookup by name. It
// accepts the bson.D that exportValue produces for plain objects as well as
// maps, so callers that only read named options need not care which they got.
func asMap(v any) (map[string]any, bool) {
	if doc, ok := v.(resultDoc); ok {
		v = bson.D(doc)
	}
	switch d := v.(type) {
	case bson.D:
		m := make(map[string]any, len(d))
		for _, e := range d {
			m[e.Key] = e.Value
		}
		return m, true
	case bson.M:
		return map[string]any(d), true
	case map[string]any:
		return d, true
	default:
		return nil, false
	}
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
