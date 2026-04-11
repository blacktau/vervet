package queryengine

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

// registerEJSON registers the EJSON global object in the Goja runtime,
// providing mongosh-compatible Extended JSON methods:
//   - EJSON.stringify(value, replacer?, space?) → string
//   - EJSON.parse(str) → object
//   - EJSON.serialize(value) → object (Extended JSON representation)
//   - EJSON.deserialize(value) → object (BSON types from Extended JSON)
func registerEJSON(rt *goja.Runtime) error {
	ejson := rt.NewObject()

	if err := ejson.Set("stringify", ejsonStringify(rt)); err != nil {
		return fmt.Errorf("failed to set EJSON.stringify: %w", err)
	}
	if err := ejson.Set("parse", ejsonParse(rt)); err != nil {
		return fmt.Errorf("failed to set EJSON.parse: %w", err)
	}
	if err := ejson.Set("serialize", ejsonSerialize(rt)); err != nil {
		return fmt.Errorf("failed to set EJSON.serialize: %w", err)
	}
	if err := ejson.Set("deserialize", ejsonDeserialize(rt)); err != nil {
		return fmt.Errorf("failed to set EJSON.deserialize: %w", err)
	}

	return rt.Set("EJSON", ejson)
}

// ejsonStringify converts a JS value to an Extended JSON string (relaxed mode).
// Usage: EJSON.stringify(value) or EJSON.stringify(value, null, 2) for indented output.
func ejsonStringify(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("EJSON.stringify requires a value argument")))
		}

		raw := exportValue(call.Arguments[0])
		bsonDoc := convertToBson(raw)

		// Check for indent (third argument, like JSON.stringify)
		indent := ""
		if len(call.Arguments) >= 3 && !goja.IsUndefined(call.Arguments[2]) && !goja.IsNull(call.Arguments[2]) {
			switch v := call.Arguments[2].Export().(type) {
			case int64:
				for i := int64(0); i < v; i++ {
					indent += " "
				}
			case float64:
				for i := 0; i < int(v); i++ {
					indent += " "
				}
			case string:
				indent = v
			}
		}

		var data []byte
		var err error
		if indent != "" {
			data, err = bson.MarshalExtJSONIndent(bsonDoc, false, false, "", indent)
		} else {
			data, err = bson.MarshalExtJSON(bsonDoc, false, false)
		}
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.stringify: %w", err)))
		}

		return rt.ToValue(string(data))
	}
}

// ejsonParse parses an Extended JSON string into a JS object.
// Usage: EJSON.parse(str)
func ejsonParse(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("EJSON.parse requires a string argument")))
		}

		str := call.Arguments[0].String()

		var result bson.M
		if err := bson.UnmarshalExtJSON([]byte(str), false, &result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.parse: %w", err)))
		}

		return rt.ToValue(result)
	}
}

// ejsonSerialize converts a JS value to its Extended JSON object representation.
// Unlike stringify, this returns a JS object (not a string) where BSON types
// are represented as Extended JSON objects (e.g. {$oid: "..."}, {$numberLong: "..."}).
// Usage: EJSON.serialize(value)
func ejsonSerialize(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("EJSON.serialize requires a value argument")))
		}

		raw := exportValue(call.Arguments[0])
		bsonDoc := convertToBson(raw)

		// Marshal to Extended JSON, then unmarshal to a generic map
		data, err := bson.MarshalExtJSON(bsonDoc, false, false)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.serialize: %w", err)))
		}

		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.serialize: %w", err)))
		}

		return rt.ToValue(result)
	}
}

// ejsonDeserialize converts an Extended JSON object representation back to
// a JS object with native BSON types. This is the inverse of serialize.
// Usage: EJSON.deserialize(obj)
func ejsonDeserialize(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("EJSON.deserialize requires a value argument")))
		}

		raw := call.Arguments[0].Export()

		// Marshal the JS object to JSON, then unmarshal via Extended JSON parser
		data, err := json.Marshal(raw)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.deserialize: %w", err)))
		}

		var result bson.M
		if err := bson.UnmarshalExtJSON(data, false, &result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.deserialize: %w", err)))
		}

		return rt.ToValue(result)
	}
}
