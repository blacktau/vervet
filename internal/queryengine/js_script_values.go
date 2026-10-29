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

// Documents read from MongoDB reach scripts as resultDoc values holding native
// driver types. Handing those to goja as reflected Go values means
// `doc.CustomerId` is an opaque int64 or ObjectID rather than a script value,
// so String(), date comparisons and _id.getTimestamp() all silently misbehave.
//
// jsValue converts them into the shapes mongosh gives a script: real Dates,
// and objects whose toString() is the value. Each wrapper keeps a __bsonValue
// so it converts straight back to BSON when used in a filter.

// scriptValue converts a normalized (see normalizeForJS) value into the
// JavaScript value a mongosh script expects.
func scriptValue(rt *goja.Runtime, v any) goja.Value {
	return jsValue(rt, v, false)
}

// jsValue builds JS objects and arrays from a value tree, keeping bson.D field
// order. Inside a resultDoc or resultValues (fromDB), BSON scalars become
// script values as well; elsewhere, such as command results and
// EJSON.serialize output, scalars are handed to goja as they are.
func jsValue(rt *goja.Runtime, v any, fromDB bool) goja.Value {
	switch val := v.(type) {
	case resultDoc:
		return jsValue(rt, bson.D(val), true)
	case resultValues:
		return jsValue(rt, bson.A(val), true)
	case map[string]any:
		obj := rt.NewObject()
		for k, elem := range val {
			_ = obj.Set(k, jsValue(rt, elem, fromDB))
		}
		return obj
	case bson.D:
		obj := rt.NewObject()
		for _, e := range val {
			_ = obj.Set(e.Key, jsValue(rt, e.Value, fromDB))
		}
		return obj
	case bson.A:
		return jsArray(rt, val, fromDB)
	case []any:
		return jsArray(rt, val, fromDB)
	}
	if fromDB {
		if converted, ok := bsonScalarToJS(rt, v); ok {
			return converted
		}
	}
	return rt.ToValue(v)
}

// jsArray uses NewArray rather than ToValue: a reflected Go slice is not a
// real JS Array, so Array.prototype methods and instanceof checks misbehave.
func jsArray(rt *goja.Runtime, val []any, fromDB bool) goja.Value {
	items := make([]any, len(val))
	for i, elem := range val {
		items[i] = jsValue(rt, elem, fromDB)
	}
	return rt.NewArray(items...)
}

// bsonScalarToJS converts a native BSON value from a document into its
// mongosh script form. The second return is false for values goja can take
// as they are (strings, booleans, nil).
func bsonScalarToJS(rt *goja.Runtime, v any) (goja.Value, bool) {
	switch x := v.(type) {
	case bson.ObjectID:
		return objectIDValue(rt, x.Hex()), true
	case int64:
		return longValue(rt, x), true
	case int32:
		return rt.ToValue(int64(x)), true
	case float64:
		return rt.ToValue(x), true
	case bson.Decimal128:
		return decimalValue(rt, x.String()), true
	case bson.DateTime:
		return jsDate(rt, x.Time()), true
	case bson.Binary:
		return binaryObject(rt, x), true
	case bson.Timestamp:
		return timestampObject(rt, x), true
	case bson.Regex:
		return regexObject(rt, x.Pattern, x.Options), true
	case bson.MinKey:
		return namedBSONValue(rt, x, "MinKey()"), true
	case bson.MaxKey:
		return namedBSONValue(rt, x, "MaxKey()"), true
	case bson.Undefined:
		return goja.Undefined(), true
	case bson.Null:
		return goja.Null(), true
	case bson.Symbol:
		return rt.ToValue(string(x)), true
	case bson.JavaScript:
		return namedBSONValue(rt, x, string(x)), true
	}
	return nil, false
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

// uuidString renders 16 raw bytes in canonical 8-4-4-4-12 form.
func uuidString(data []byte) string {
	h := hex.EncodeToString(data)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// regexObject builds a JS RegExp so scripts can call .test() on it. MongoDB
// options JS does not support (e.g. x) leave the pattern as a plain string.
func regexObject(rt *goja.Runtime, pattern, options string) goja.Value {
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
