package queryengine

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
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
		bsonDoc := relaxedExtJSONTree(convertToBson(raw), false)

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

// maxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1), the
// largest integer a JS number holds exactly.
const maxSafeInteger = 1<<53 - 1

// extJSONWrappers are the keys that open an Extended JSON type wrapper. The
// parser wants plain numbers inside some of them ($timestamp, $date), so longs
// are not spelled out there. Query operators such as $set are not wrappers.
var extJSONWrappers = map[string]bool{
	"$oid": true, "$symbol": true, "$numberInt": true, "$numberLong": true,
	"$numberDouble": true, "$numberDecimal": true, "$binary": true, "$uuid": true,
	"$code": true, "$scope": true, "$timestamp": true, "$regularExpression": true,
	"$dbPointer": true, "$date": true, "$minKey": true, "$maxKey": true,
	"$undefined": true,
}

// relaxedExtJSONTree prepares a BSON tree for relaxed Extended JSON. Whole-number
// doubles within ±2^53 become int64 so they print as plain integers, as mongosh
// does (1761782400123, not 1.761782400123E+12); -0 stays a double so its sign
// survives. With spellLongs, each int64 already in the tree (a Long) is written
// as {$numberLong: ...}, which the parser reads back as a long rather than an
// int32, except inside wrappers such as $timestamp (see extJSONWrappers).
func relaxedExtJSONTree(v any, spellLongs bool) any {
	switch val := v.(type) {
	case bson.D:
		if len(val) > 0 && extJSONWrappers[val[0].Key] {
			spellLongs = false
		}
		doc := make(bson.D, len(val))
		for i, e := range val {
			doc[i] = bson.E{Key: e.Key, Value: relaxedExtJSONTree(e.Value, spellLongs)}
		}
		return doc
	case bson.A:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = relaxedExtJSONTree(item, spellLongs)
		}
		return arr
	case int64:
		if spellLongs {
			return bson.D{{Key: "$numberLong", Value: strconv.FormatInt(val, 10)}}
		}
	case float64:
		if val == math.Trunc(val) && math.Abs(val) <= maxSafeInteger && !(val == 0 && math.Signbit(val)) {
			return int64(val)
		}
	}
	return v
}

// ejsonParse parses an Extended JSON string into a JS object.
// Usage: EJSON.parse(str)
func ejsonParse(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("EJSON.parse requires a string argument")))
		}

		str := call.Arguments[0].String()

		return parsedExtJSONValue(rt, []byte(str), "EJSON.parse")
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
		bsonDoc := relaxedExtJSONTree(convertToBson(raw), true)

		// Marshal to Extended JSON, then decode to an ordered generic tree
		data, err := bson.MarshalExtJSON(bsonDoc, false, false)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.serialize: %w", err)))
		}

		result, err := decodeOrderedJSON(data)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.serialize: %w", err)))
		}

		// Deliberately not toJSValue: serialize's contract is to expose the
		// Extended JSON form ({$oid: ...}), which toJSValue would convert back
		// into script values.
		return jsValue(rt, result, jsPlain)
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

		// convertToBson first so already-typed script values (ObjectId, Long,
		// Date wrappers) are understood as well as plain Extended JSON input.
		// Relaxed, because the parser rejects canonical {$numberInt: ...} inside
		// wrappers such as $timestamp and $date.
		data, err := bson.MarshalExtJSON(relaxedExtJSONTree(convertToBson(exportValue(call.Arguments[0])), true), false, false)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("EJSON.deserialize: %w", err)))
		}
		return parsedExtJSONValue(rt, data, "EJSON.deserialize")
	}
}

// parsedExtJSONValue parses Extended JSON into a script value. A value written
// as {$numberLong: ...} is a Long whatever its size: the text says it is a
// long, so writing it back must store a long. A plain integer is a number.
func parsedExtJSONValue(rt *goja.Runtime, data []byte, fn string) goja.Value {
	var result bson.D
	if err := bson.UnmarshalExtJSON(data, false, &result); err != nil {
		panic(rt.NewGoError(fmt.Errorf("%s: %w", fn, err)))
	}
	if !bytes.Contains(data, []byte("$numberLong")) {
		// No long is written out, so every int64 is a plain integer.
		return toJSValue(rt, markExplicitLongs(result, nil))
	}
	raw, err := decodeOrderedJSON(data)
	if err != nil {
		panic(rt.NewGoError(fmt.Errorf("%s: %w", fn, err)))
	}
	return toJSValue(rt, markExplicitLongs(result, raw))
}

// markExplicitLongs walks the parsed tree alongside the raw JSON tree. The
// driver decodes both {$numberLong: "7"} and a plain integer beyond int32 as
// int64; only in the raw tree is the first an object, so that tells them apart.
// A {$numberLong} stays int64 and becomes a Long. A plain integer becomes a
// float64, so it reaches the script as a plain number at any size, as js-bson's
// relaxed parse gives it. A nil raw tree means the text has no {$numberLong}.
func markExplicitLongs(parsed, raw any) any {
	switch p := parsed.(type) {
	case bson.D:
		// Both trees come from the same text, so their fields are in the same
		// order; walking by index also copes with repeated keys.
		rawDoc, _ := raw.(bson.D)
		if raw != nil && len(rawDoc) != len(p) {
			return p
		}
		out := make(bson.D, len(p))
		for i, e := range p {
			var rawVal any
			if raw != nil {
				rawVal = rawDoc[i].Value
			}
			out[i] = bson.E{Key: e.Key, Value: markExplicitLongs(e.Value, rawVal)}
		}
		return out
	case bson.A:
		rawArr, _ := raw.([]any)
		if raw != nil && len(rawArr) != len(p) {
			return p
		}
		out := make(bson.A, len(p))
		for i, item := range p {
			var rawVal any
			if raw != nil {
				rawVal = rawArr[i]
			}
			out[i] = markExplicitLongs(item, rawVal)
		}
		return out
	case int64:
		if _, ok := raw.(bson.D); ok {
			return p
		}
		return float64(p)
	}
	return parsed
}
