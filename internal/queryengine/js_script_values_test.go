package queryengine

import (
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// evalWithValue converts a document field the way a query result is converted
// for a script, binds the document as `doc`, and evaluates expr against it.
func evalWithValue(t *testing.T, v any, expr string) goja.Value {
	t.Helper()
	rt := goja.New()
	require.NoError(t, rt.Set("doc", toJSValue(rt, resultDoc{{Key: "v", Value: v}})))
	val, err := rt.RunString(expr)
	require.NoError(t, err)
	return val
}

// Results reach scripts as native driver values. Left as reflected Go values,
// `String(id)` yields "[object Object]" and every id collapses to the same map
// key, which silently produces wrong answers rather than an error.
func TestScriptValue_LongStringifiesToItsDigits(t *testing.T) {
	v := int64(2309825)
	assert.Equal(t, "2309825", evalWithValue(t, v, `String(doc.v)`).Export())
	assert.Equal(t, "2309825", evalWithValue(t, v, "`${doc.v}`").Export())
}

func TestScriptValue_ObjectIDHasHexStringAndTimestamp(t *testing.T) {
	v, err := bson.ObjectIDFromHex("6a8007006ae0e594998563b1")
	require.NoError(t, err)
	assert.Equal(t, "6a8007006ae0e594998563b1", evalWithValue(t, v, `String(doc.v)`).Export())
	assert.Equal(t, true, evalWithValue(t, v, `doc.v.getTimestamp() instanceof Date`).Export())
}

func TestScriptValue_DateIsARealJSDate(t *testing.T) {
	v := bson.NewDateTimeFromTime(time.UnixMilli(1786393408000))
	assert.Equal(t, true, evalWithValue(t, v, `doc.v instanceof Date`).Export())
	assert.Equal(t, "2026-08-10T20:23:28.000Z", evalWithValue(t, v, `doc.v.toISOString()`).Export())
}

func TestScriptValue_NestedDocumentKeepsPlainFields(t *testing.T) {
	v := bson.D{
		{Key: "CustomerId", Value: int64(1107812)},
		{Key: "IsError", Value: false},
		{Key: "Tags", Value: bson.A{"a", "b"}},
		{Key: "Count", Value: int32(3)},
		{Key: "Ratio", Value: 0.5},
	}
	assert.Equal(t, "1107812|false|a,b|4|1",
		evalWithValue(t, v, `[String(doc.v.CustomerId), doc.v.IsError, doc.v.Tags.join(","), doc.v.Count + 1, doc.v.Ratio * 2].join("|")`).Export())
}

func TestScriptValue_BinarySubtype4RendersAsUUID(t *testing.T) {
	v := bson.Binary{Subtype: 4, Data: []byte{0xce, 0x93, 0x13, 0xd1, 0x4c, 0xaa, 0x45, 0xb6, 0x96, 0x9d, 0x11, 0x61, 0xe0, 0x0a, 0xd8, 0x53}}
	assert.Equal(t, "ce9313d1-4caa-45b6-969d-1161e00ad853", evalWithValue(t, v, `String(doc.v)`).Export())
}

func TestScriptValue_RegexBecomesUsableRegExp(t *testing.T) {
	v := bson.Regex{Pattern: "^ab", Options: "i"}
	assert.Equal(t, true, evalWithValue(t, v, `doc.v.test("ABc")`).Export())
}

func TestScriptValue_TimestampAndDecimal(t *testing.T) {
	assert.Equal(t, "Timestamp({ t: 5, i: 2 })", evalWithValue(t, bson.Timestamp{T: 5, I: 2}, `String(doc.v)`).Export())
	d, err := bson.ParseDecimal128("1.25")
	require.NoError(t, err)
	assert.Equal(t, "1.25", evalWithValue(t, d, `String(doc.v)`).Export())
}

// Converted values must still convert back to BSON when used in a filter.
func TestScriptValue_RoundTripsBackToBSON(t *testing.T) {
	rt := goja.New()
	require.NoError(t, rt.Set("doc", toJSValue(rt, resultDoc{{Key: "v", Value: int64(2309825)}})))
	val, err := rt.RunString(`({ CustomerId: doc.v })`)
	require.NoError(t, err)

	doc, ok := convertToBson(exportValue(val)).(bson.D)
	require.True(t, ok, "expected a bson.D filter")
	require.Len(t, doc, 1)
	assert.Equal(t, "CustomerId", doc[0].Key)
	assert.Equal(t, int64(2309825), doc[0].Value, "Long must survive as int64, not become a float")
}

// Command results are not documents read from a collection: their scalars stay
// plain, so arithmetic like db.stats().storageSize / 1024 keeps working.
func TestScriptValue_CommandResultScalarsStayPlain(t *testing.T) {
	rt := goja.New()
	require.NoError(t, rt.Set("r", toJSValue(rt, bson.D{{Key: "storageSize", Value: int64(4096)}})))
	val, err := rt.RunString(`r.storageSize / 1024`)
	require.NoError(t, err)
	assert.Equal(t, int64(4), val.Export())
}
