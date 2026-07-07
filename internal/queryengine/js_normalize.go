package queryengine

import (
	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// normalizeForJS converts driver result values into plain Go containers before
// they cross into the goja runtime. In mongo-driver v2, bson.M and bson.D gained
// methods (String, MarshalJSON), and goja only reflects a map as a JS object when
// the map type has zero methods (see goja runtime.go: `NumMethod() == 0`). Without
// this, a bson.M is wrapped as an opaque object exposing "String" instead of its
// keys. Stripping M/D/A down to method-less map[string]any / []any restores the
// v1 reflection behaviour. Scalar BSON types (ObjectID, DateTime, ...) reflected
// identically in v1 and v2, so they pass through untouched.
func normalizeForJS(v any) any {
	switch val := v.(type) {
	case bson.M:
		out := make(map[string]any, len(val))
		for k, elem := range val {
			out[k] = normalizeForJS(elem)
		}
		return out
	case bson.D:
		// ponytail: D is flattened to an object (last-wins on duplicate keys). v1
		// exposed D as an array to JS, but D is not on the result path (results are
		// bson.M); this branch is defensive.
		out := make(map[string]any, len(val))
		for _, e := range val {
			out[e.Key] = normalizeForJS(e.Value)
		}
		return out
	case bson.A:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = normalizeForJS(elem)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = normalizeForJS(elem)
		}
		return out
	case []bson.M:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = normalizeForJS(elem)
		}
		return out
	default:
		return v
	}
}

// toJSValue normalizes a driver result value and hands it to the goja runtime.
func toJSValue(rt *goja.Runtime, v any) goja.Value {
	return rt.ToValue(normalizeForJS(v))
}
