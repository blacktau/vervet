package queryengine

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// registerBSONTypes registers mongosh-compatible BSON type constructors as globals
// in the Goja runtime: ObjectId, ISODate, NumberLong, NumberDecimal, UUID,
// Timestamp, MinKey, MaxKey, BinData.
func registerBSONTypes(rt *goja.Runtime) error {
	types := map[string]func(goja.FunctionCall) goja.Value{
		"ObjectId":      bsonObjectId(rt),
		"ISODate":       bsonISODate(rt),
		"NumberLong":    bsonNumberLong(rt),
		"NumberDecimal": bsonNumberDecimal(rt),
		"UUID":          bsonUUID(rt),
		"Timestamp":     bsonTimestamp(rt),
		"MinKey":        bsonMinKey(rt),
		"MaxKey":        bsonMaxKey(rt),
		"BinData":       bsonBinData(rt),
	}

	for name, fn := range types {
		if err := rt.Set(name, fn); err != nil {
			return fmt.Errorf("failed to set %s global: %w", name, err)
		}
	}
	return nil
}

// bsonObjectId returns a function that creates a primitive.ObjectID.
// Usage: ObjectId() — new random ID, or ObjectId("hex") — from hex string.
func bsonObjectId(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) {
			return wrapBSONValue(rt, primitive.NewObjectID())
		}
		hex := call.Arguments[0].String()
		oid, err := primitive.ObjectIDFromHex(hex)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("ObjectId: %w", err)))
		}
		return wrapBSONValue(rt, oid)
	}
}

// bsonISODate returns a function that creates a primitive.DateTime.
// Usage: ISODate() — now, or ISODate("2024-01-15T00:00:00Z") — from ISO string.
func bsonISODate(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) {
			return wrapBSONValue(rt, primitive.NewDateTimeFromTime(time.Now()))
		}
		str := call.Arguments[0].String()
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			// Try without timezone
			t, err = time.Parse("2006-01-02T15:04:05", str)
			if err != nil {
				// Try date only
				t, err = time.Parse("2006-01-02", str)
				if err != nil {
					panic(rt.NewGoError(fmt.Errorf("ISODate: invalid date string %q", str)))
				}
			}
		}
		return wrapBSONValue(rt, primitive.NewDateTimeFromTime(t))
	}
}

// bsonNumberLong returns a function that creates an int64 wrapped in a Goja object.
// The int64 is stored as __bsonValue so convertToBson can extract it without
// JS float64 precision loss.
// Usage: NumberLong(123) or NumberLong("123").
func bsonNumberLong(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		var n int64
		if len(call.Arguments) == 0 {
			n = 0
		} else {
			arg := call.Arguments[0]
			exported := arg.Export()
			switch v := exported.(type) {
			case int64:
				n = v
			case float64:
				n = int64(v)
			case string:
				_, err := fmt.Sscanf(v, "%d", &n)
				if err != nil {
					panic(rt.NewGoError(fmt.Errorf("NumberLong: cannot parse %q as integer", v)))
				}
			default:
				n = arg.ToInteger()
			}
		}
		return wrapBSONValue(rt, n)
	}
}

// bsonWrapper holds a BSON value opaquely so Goja doesn't coerce it to a JS type.
type bsonWrapper struct {
	Value any
}

// wrapBSONValue wraps a Go BSON value in a Goja object with a __bsonValue property
// so that convertToBson can extract the original Go type without JS type coercion.
// The value is stored inside a bsonWrapper struct to prevent Goja from converting
// Go primitives (like int64) to JS number (float64).
func wrapBSONValue(rt *goja.Runtime, val any) goja.Value {
	obj := rt.NewObject()
	_ = obj.Set("__bsonValue", &bsonWrapper{Value: val})
	return obj
}

// bsonNumberDecimal returns a function that creates a primitive.Decimal128.
// Usage: NumberDecimal("123.456").
func bsonNumberDecimal(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("NumberDecimal requires a string argument")))
		}
		str := call.Arguments[0].String()
		d, err := primitive.ParseDecimal128(str)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("NumberDecimal: %w", err)))
		}
		return wrapBSONValue(rt, d)
	}
}

// bsonUUID returns a function that creates a primitive.Binary with subtype 4.
// Usage: UUID("550e8400-e29b-41d4-a716-446655440000").
func bsonUUID(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(rt.NewGoError(fmt.Errorf("UUID requires a string argument")))
		}
		str := call.Arguments[0].String()
		// Strip hyphens from UUID string
		cleaned := ""
		for _, c := range str {
			if c != '-' {
				cleaned += string(c)
			}
		}
		data, err := hex.DecodeString(cleaned)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("UUID: invalid hex string %q", str)))
		}
		if len(data) != 16 {
			panic(rt.NewGoError(fmt.Errorf("UUID: must be 16 bytes, got %d", len(data))))
		}
		return wrapBSONValue(rt, primitive.Binary{Subtype: 0x04, Data: data})
	}
}

// bsonTimestamp returns a function that creates a primitive.Timestamp.
// Usage: Timestamp(t, i) where t is seconds since epoch and i is an increment.
func bsonTimestamp(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(rt.NewGoError(fmt.Errorf("Timestamp requires two arguments: (seconds, increment)")))
		}
		t := uint32(call.Arguments[0].ToInteger())
		i := uint32(call.Arguments[1].ToInteger())
		return wrapBSONValue(rt, primitive.Timestamp{T: t, I: i})
	}
}

// bsonMinKey returns a function that creates a primitive.MinKey.
// Usage: MinKey()
func bsonMinKey(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return wrapBSONValue(rt, primitive.MinKey{})
	}
}

// bsonMaxKey returns a function that creates a primitive.MaxKey.
// Usage: MaxKey()
func bsonMaxKey(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return wrapBSONValue(rt, primitive.MaxKey{})
	}
}

// bsonBinData returns a function that creates a primitive.Binary.
// Usage: BinData(subtype, base64String).
func bsonBinData(rt *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(rt.NewGoError(fmt.Errorf("BinData requires two arguments: (subtype, base64String)")))
		}
		subtype := byte(call.Arguments[0].ToInteger())
		b64 := call.Arguments[1].String()

		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("BinData: %w", err)))
		}
		return wrapBSONValue(rt, primitive.Binary{Subtype: subtype, Data: data})
	}
}
