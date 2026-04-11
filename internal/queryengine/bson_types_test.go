package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func setupRuntimeWithBSON(t *testing.T) *goja.Runtime {
	t.Helper()
	rt := goja.New()
	err := registerBSONTypes(rt)
	require.NoError(t, err)
	return rt
}

// extractBSONValue unwraps a __bsonValue from a Goja-exported map.
func extractBSONValue(t *testing.T, val goja.Value) any {
	t.Helper()
	exported := val.Export()
	m, ok := exported.(map[string]any)
	require.True(t, ok, "expected wrapped map, got %T", exported)
	bsonVal, ok := m["__bsonValue"]
	require.True(t, ok, "expected __bsonValue key in map")
	if w, ok := bsonVal.(*bsonWrapper); ok {
		return w.Value
	}
	return bsonVal
}

func TestObjectId_NoArgs_ReturnsNewObjectID(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`ObjectId()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	_, ok := bsonVal.(primitive.ObjectID)
	assert.True(t, ok, "expected primitive.ObjectID, got %T", bsonVal)
}

func TestObjectId_WithHex_ReturnsCorrectID(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`ObjectId("507f1f77bcf86cd799439011")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	oid, ok := bsonVal.(primitive.ObjectID)
	require.True(t, ok)
	assert.Equal(t, "507f1f77bcf86cd799439011", oid.Hex())
}

func TestObjectId_InvalidHex_Panics(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	_, err := rt.RunString(`ObjectId("invalid")`)
	assert.Error(t, err)
}

func TestISODate_NoArgs_ReturnsNow(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`ISODate()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	_, ok := bsonVal.(primitive.DateTime)
	assert.True(t, ok, "expected primitive.DateTime, got %T", bsonVal)
}

func TestISODate_WithRFC3339_ReturnsCorrectDate(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`ISODate("2024-01-15T10:30:00Z")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	dt, ok := bsonVal.(primitive.DateTime)
	require.True(t, ok)
	tm := dt.Time()
	assert.Equal(t, 2024, tm.Year())
	assert.Equal(t, 1, int(tm.Month()))
	assert.Equal(t, 15, tm.Day())
}

func TestISODate_DateOnly_Works(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`ISODate("2024-01-15")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	dt, ok := bsonVal.(primitive.DateTime)
	require.True(t, ok)
	assert.Equal(t, 2024, dt.Time().Year())
}

func TestISODate_Invalid_Panics(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	_, err := rt.RunString(`ISODate("not-a-date")`)
	assert.Error(t, err)
}

func TestNumberInt_WithInt_ReturnsInt32(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberInt(42)`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int32(42), bsonVal)
}

func TestNumberInt_WithString_ParsesInt(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberInt("123")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int32(123), bsonVal)
}

func TestNumberInt_NoArgs_ReturnsZero(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberInt()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int32(0), bsonVal)
}

func TestNumberLong_WithInt_ReturnsInt64(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberLong(42)`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int64(42), bsonVal)
}

func TestNumberLong_WithString_ParsesInt(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberLong("9007199254740993")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int64(9007199254740993), bsonVal)
}

func TestNumberLong_NoArgs_ReturnsZero(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberLong()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	assert.Equal(t, int64(0), bsonVal)
}

func TestNumberDecimal_Valid_ReturnsDecimal128(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`NumberDecimal("123.456")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	_, ok := bsonVal.(primitive.Decimal128)
	assert.True(t, ok, "expected primitive.Decimal128, got %T", bsonVal)
}

func TestNumberDecimal_NoArgs_Panics(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	_, err := rt.RunString(`NumberDecimal()`)
	assert.Error(t, err)
}

func TestUUID_Valid_ReturnsBinary(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`UUID("550e8400-e29b-41d4-a716-446655440000")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	bin, ok := bsonVal.(primitive.Binary)
	require.True(t, ok)
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)
}

func TestUUID_NoArgs_GeneratesRandomUUID(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`UUID()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	bin, ok := bsonVal.(primitive.Binary)
	require.True(t, ok)
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)

	// Calling UUID() again should produce a different value
	val2, err := rt.RunString(`UUID()`)
	require.NoError(t, err)
	bsonVal2 := extractBSONValue(t, val2)
	bin2, ok := bsonVal2.(primitive.Binary)
	require.True(t, ok)
	assert.NotEqual(t, bin.Data, bin2.Data)
}

func TestTimestamp_Valid_ReturnsTimestamp(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`Timestamp(1700000000, 1)`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	ts, ok := bsonVal.(primitive.Timestamp)
	require.True(t, ok)
	assert.Equal(t, uint32(1700000000), ts.T)
	assert.Equal(t, uint32(1), ts.I)
}

func TestTimestamp_TooFewArgs_Panics(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	_, err := rt.RunString(`Timestamp(1)`)
	assert.Error(t, err)
}

func TestMinKey_ReturnsMinKey(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`MinKey()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	_, ok := bsonVal.(primitive.MinKey)
	assert.True(t, ok, "expected primitive.MinKey, got %T", bsonVal)
}

func TestMaxKey_ReturnsMaxKey(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`MaxKey()`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	_, ok := bsonVal.(primitive.MaxKey)
	assert.True(t, ok, "expected primitive.MaxKey, got %T", bsonVal)
}

func TestBinData_Valid_ReturnsBinary(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`BinData(0, "aGVsbG8=")`)
	require.NoError(t, err)
	bsonVal := extractBSONValue(t, val)
	bin, ok := bsonVal.(primitive.Binary)
	require.True(t, ok)
	assert.Equal(t, byte(0), bin.Subtype)
	assert.Equal(t, []byte("hello"), bin.Data)
}

func TestBinData_TooFewArgs_Panics(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	_, err := rt.RunString(`BinData(0)`)
	assert.Error(t, err)
}

// Test that regex values survive exportValue + convertToBson roundtrip
func TestConvertToBson_UnwrapsRegex(t *testing.T) {
	rt := goja.New()
	val, err := rt.RunString(`({ name: /foo/i })`)
	require.NoError(t, err)

	exported := exportValue(val)
	doc := convertToBson(exported)

	bsonDoc, ok := doc.(bson.D)
	require.True(t, ok, "expected bson.D, got %T", doc)

	for _, elem := range bsonDoc {
		if elem.Key == "name" {
			regex, ok := elem.Value.(primitive.Regex)
			require.True(t, ok, "expected primitive.Regex, got %T", elem.Value)
			assert.Equal(t, "foo", regex.Pattern)
			assert.Equal(t, "i", regex.Options)
		}
	}
}

// Test that BSON values survive convertToBson roundtrip
func TestConvertToBson_UnwrapsBSONValue(t *testing.T) {
	rt := setupRuntimeWithBSON(t)
	val, err := rt.RunString(`({ _id: ObjectId("507f1f77bcf86cd799439011"), count: NumberLong("42") })`)
	require.NoError(t, err)

	exported := val.Export()
	doc := convertToBson(exported)

	// Should be a bson.D with unwrapped values
	bsonDoc, ok := doc.(bson.D)
	require.True(t, ok, "expected bson.D, got %T", doc)

	for _, elem := range bsonDoc {
		switch elem.Key {
		case "_id":
			oid, ok := elem.Value.(primitive.ObjectID)
			require.True(t, ok, "expected ObjectID, got %T", elem.Value)
			assert.Equal(t, "507f1f77bcf86cd799439011", oid.Hex())
		case "count":
			assert.Equal(t, int64(42), elem.Value)
		}
	}
}
