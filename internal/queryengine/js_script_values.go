package queryengine

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Query results are round-tripped through canonical Extended JSON (docsToResult)
// so the frontend can render BSON types. Handing those raw maps to a script means
// `doc.CustomerId` is `{$numberLong: "42"}` rather than a value, so String(),
// date comparisons and _id.getTimestamp() all silently misbehave.
//
// scriptValue converts an Extended JSON tree back into the shapes mongosh gives a
// script: real Dates, and objects whose toString() is the value. Each wrapper
// keeps a __bsonValue so it converts straight back to BSON when used in a filter.

// scriptValue converts a normalized (see normalizeForJS) result value into the
// JavaScript value a mongosh script expects.
func scriptValue(rt *goja.Runtime, v any) goja.Value {
	switch val := v.(type) {
	case map[string]any:
		if converted, ok := ejsonToJS(rt, val); ok {
			return converted
		}
		obj := rt.NewObject()
		for k, elem := range val {
			_ = obj.Set(k, scriptValue(rt, elem))
		}
		return obj
	case []any:
		// NewArray rather than ToValue: a reflected Go slice is not a real JS
		// Array, so Array.prototype methods and instanceof checks misbehave.
		items := make([]any, len(val))
		for i, elem := range val {
			items[i] = scriptValue(rt, elem)
		}
		return rt.NewArray(items...)
	default:
		return rt.ToValue(v)
	}
}

// ejsonToJS converts a single Extended JSON wrapper map into its JavaScript
// equivalent. The second return is false when the map is an ordinary document.
func ejsonToJS(rt *goja.Runtime, m map[string]any) (goja.Value, bool) {
	if len(m) == 0 {
		return nil, false
	}

	switch {
	case has(m, "$oid"):
		return objectIDValue(rt, str(m["$oid"])), true
	case has(m, "$numberLong"):
		n, err := strconv.ParseInt(str(m["$numberLong"]), 10, 64)
		if err != nil {
			return nil, false
		}
		return longValue(rt, n), true
	case has(m, "$numberInt"):
		n, err := strconv.ParseInt(str(m["$numberInt"]), 10, 32)
		if err != nil {
			return nil, false
		}
		return rt.ToValue(n), true
	case has(m, "$numberDouble"):
		return doubleValue(rt, str(m["$numberDouble"])), true
	case has(m, "$numberDecimal"):
		return decimalValue(rt, str(m["$numberDecimal"])), true
	case has(m, "$date"):
		return dateValue(rt, m["$date"]), true
	case has(m, "$binary"):
		return binaryValue(rt, m["$binary"]), true
	case has(m, "$timestamp"):
		return timestampValue(rt, m["$timestamp"]), true
	case has(m, "$regularExpression"):
		return regexValue(rt, m["$regularExpression"]), true
	case has(m, "$minKey"):
		return namedBSONValue(rt, bson.MinKey{}, "MinKey()"), true
	case has(m, "$maxKey"):
		return namedBSONValue(rt, bson.MaxKey{}, "MaxKey()"), true
	case has(m, "$undefined"):
		return goja.Undefined(), true
	case has(m, "$symbol"):
		return rt.ToValue(str(m["$symbol"])), true
	case has(m, "$code"):
		return namedBSONValue(rt, bson.JavaScript(str(m["$code"])), str(m["$code"])), true
	}

	return nil, false
}

func has(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// bsonObject builds the base wrapper: a JS object carrying the Go BSON value in
// __bsonValue (so convertToBson can unwrap it) plus a toString/toJSON pair, which
// is what makes String(x) and `${x}` produce the value rather than [object Object].
func bsonObject(rt *goja.Runtime, val any, text string) *goja.Object {
	obj := rt.NewObject()
	_ = obj.Set("__bsonValue", &bsonWrapper{Value: val})
	_ = obj.Set("toString", func() string { return text })
	_ = obj.Set("toJSON", func() string { return text })
	return obj
}

// namedBSONValue wraps a BSON value that needs nothing beyond its string form.
func namedBSONValue(rt *goja.Runtime, val any, text string) goja.Value {
	return bsonObject(rt, val, text)
}

func objectIDValue(rt *goja.Runtime, hexStr string) goja.Value {
	oid, err := bson.ObjectIDFromHex(hexStr)
	if err != nil {
		return rt.ToValue(hexStr)
	}
	obj := bsonObject(rt, oid, hexStr)
	_ = obj.Set("toHexString", func() string { return hexStr })
	_ = obj.Set("getTimestamp", func() goja.Value {
		return jsDate(rt, oid.Timestamp())
	})
	return obj
}

// jsDate builds a real JS Date. goja reflects a time.Time as an opaque Go
// object, so the Date constructor has to be called explicitly for scripts to
// get .toISOString(), comparisons and `instanceof Date`.
func jsDate(rt *goja.Runtime, t time.Time) goja.Value {
	ctor, ok := goja.AssertConstructor(rt.Get("Date"))
	if !ok {
		return rt.ToValue(t)
	}
	d, err := ctor(nil, rt.ToValue(float64(t.UnixMilli())))
	if err != nil {
		return rt.ToValue(t)
	}
	return d
}

// longValue mirrors mongosh's Long: an object, not a JS number, so 64-bit ids
// survive without float64 rounding. toString gives the digits.
func longValue(rt *goja.Runtime, n int64) goja.Value {
	return numberValue(rt, n, n)
}

// numberValue wraps an integral BSON number (int32 or int64), keeping the Go
// type for the trip back to BSON while stringifying to plain digits.
func numberValue(rt *goja.Runtime, val any, n int64) goja.Value {
	obj := bsonObject(rt, val, strconv.FormatInt(n, 10))
	_ = obj.Set("toNumber", func() float64 { return float64(n) })
	return obj
}

// binaryObject wraps a bson.Binary, rendering subtype-4 values as a canonical
// UUID string and everything else as base64 — matching mongosh's toString.
func binaryObject(rt *goja.Runtime, bin bson.Binary) goja.Value {
	b64 := base64.StdEncoding.EncodeToString(bin.Data)
	text := b64
	isUUID := bin.Subtype == 0x04 && len(bin.Data) == 16
	if isUUID {
		text = uuidString(bin.Data)
	}

	obj := bsonObject(rt, bin, text)
	_ = obj.Set("base64", b64)
	_ = obj.Set("subType", bin.Subtype)
	if isUUID {
		_ = obj.Set("toUUID", func() string { return text })
	}
	return obj
}

// timestampObject wraps a bson.Timestamp with its mongosh string form.
func timestampObject(rt *goja.Runtime, ts bson.Timestamp) goja.Value {
	obj := bsonObject(rt, ts, fmt.Sprintf("Timestamp({ t: %d, i: %d })", ts.T, ts.I))
	_ = obj.Set("t", ts.T)
	_ = obj.Set("i", ts.I)
	return obj
}

func doubleValue(rt *goja.Runtime, s string) goja.Value {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return rt.ToValue(s)
	}
	return rt.ToValue(f)
}

func decimalValue(rt *goja.Runtime, s string) goja.Value {
	d, err := bson.ParseDecimal128(s)
	if err != nil {
		return rt.ToValue(s)
	}
	obj := bsonObject(rt, d, s)
	_ = obj.Set("toNumber", func() goja.Value {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return goja.NaN()
		}
		return rt.ToValue(f)
	})
	return obj
}

// dateValue accepts both canonical ({$numberLong: millis}) and relaxed (ISO
// string) forms of $date and returns a real JS Date.
func dateValue(rt *goja.Runtime, v any) goja.Value {
	switch d := v.(type) {
	case map[string]any:
		if ms, ok := d["$numberLong"]; ok {
			n, err := strconv.ParseInt(str(ms), 10, 64)
			if err == nil {
				return jsDate(rt, time.UnixMilli(n))
			}
		}
	case string:
		if t, err := time.Parse(time.RFC3339, d); err == nil {
			return jsDate(rt, t)
		}
	case float64:
		return jsDate(rt, time.UnixMilli(int64(d)))
	}
	return rt.ToValue(v)
}

func binaryValue(rt *goja.Runtime, v any) goja.Value {
	spec, ok := v.(map[string]any)
	if !ok {
		return rt.ToValue(v)
	}
	b64 := str(spec["base64"])
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return rt.ToValue(b64)
	}
	subtype := byte(0)
	if st, err := strconv.ParseUint(str(spec["subType"]), 16, 8); err == nil {
		subtype = byte(st)
	}
	return binaryObject(rt, bson.Binary{Subtype: subtype, Data: data})
}

// uuidString renders 16 raw bytes in canonical 8-4-4-4-12 form.
func uuidString(data []byte) string {
	h := hex.EncodeToString(data)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

func timestampValue(rt *goja.Runtime, v any) goja.Value {
	spec, ok := v.(map[string]any)
	if !ok {
		return rt.ToValue(v)
	}
	return timestampObject(rt, bson.Timestamp{
		T: uint32(toUint64(spec["t"])),
		I: uint32(toUint64(spec["i"])),
	})
}

func toUint64(v any) uint64 {
	switch n := v.(type) {
	case float64:
		return uint64(n)
	case int64:
		return uint64(n)
	case string:
		parsed, err := strconv.ParseUint(n, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

// regexValue rebuilds a JS RegExp so scripts can call .test() on it.
func regexValue(rt *goja.Runtime, v any) goja.Value {
	spec, ok := v.(map[string]any)
	if !ok {
		return rt.ToValue(v)
	}
	pattern := str(spec["pattern"])
	options := str(spec["options"])

	ctor, ok := goja.AssertConstructor(rt.Get("RegExp"))
	if !ok {
		return rt.ToValue(pattern)
	}
	re, err := ctor(nil, rt.ToValue(pattern), rt.ToValue(options))
	if err != nil {
		return rt.ToValue(pattern)
	}
	return re
}
