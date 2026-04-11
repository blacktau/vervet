package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEJSONRuntime(t *testing.T) *goja.Runtime {
	t.Helper()
	rt := goja.New()
	require.NoError(t, registerBSONTypes(rt))
	require.NoError(t, registerEJSON(rt))
	return rt
}

func TestEJSON_Stringify_SimpleObject(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`EJSON.stringify({ name: "alice", age: 30 })`)
	require.NoError(t, err)
	s := val.String()
	assert.Contains(t, s, `"name"`)
	assert.Contains(t, s, `"alice"`)
}

func TestEJSON_Stringify_WithIndent(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`EJSON.stringify({ x: 1 }, null, 2)`)
	require.NoError(t, err)
	s := val.String()
	// Indented output should contain newlines
	assert.Contains(t, s, "\n")
}

func TestEJSON_Stringify_NoArgs_Panics(t *testing.T) {
	rt := setupEJSONRuntime(t)
	_, err := rt.RunString(`EJSON.stringify()`)
	assert.Error(t, err)
}

func TestEJSON_Stringify_WithObjectId(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`EJSON.stringify({ _id: ObjectId("507f1f77bcf86cd799439011") })`)
	require.NoError(t, err)
	s := val.String()
	assert.Contains(t, s, `$oid`)
	assert.Contains(t, s, `507f1f77bcf86cd799439011`)
}

func TestEJSON_Parse_SimpleObject(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var obj = EJSON.parse('{"name": "bob", "age": 25}');
		obj.name
	`)
	require.NoError(t, err)
	assert.Equal(t, "bob", val.String())
}

func TestEJSON_Parse_WithExtendedJSON(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var obj = EJSON.parse('{"_id": {"$oid": "507f1f77bcf86cd799439011"}}');
		typeof obj._id
	`)
	require.NoError(t, err)
	// After parsing extended JSON, _id should be an ObjectID (object type in JS)
	assert.Equal(t, "object", val.String())
}

func TestEJSON_Parse_NoArgs_Panics(t *testing.T) {
	rt := setupEJSONRuntime(t)
	_, err := rt.RunString(`EJSON.parse()`)
	assert.Error(t, err)
}

func TestEJSON_Serialize_ReturnsObject(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var result = EJSON.serialize({ _id: ObjectId("507f1f77bcf86cd799439011") });
		typeof result
	`)
	require.NoError(t, err)
	// serialize returns an object, not a string
	assert.Equal(t, "object", val.String())
}

func TestEJSON_Serialize_ContainsExtendedJSONKeys(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var result = EJSON.serialize({ _id: ObjectId("507f1f77bcf86cd799439011") });
		result._id["$oid"]
	`)
	require.NoError(t, err)
	assert.Equal(t, "507f1f77bcf86cd799439011", val.String())
}

func TestEJSON_Serialize_NoArgs_Panics(t *testing.T) {
	rt := setupEJSONRuntime(t)
	_, err := rt.RunString(`EJSON.serialize()`)
	assert.Error(t, err)
}

func TestEJSON_Deserialize_FromExtendedJSON(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var obj = EJSON.deserialize({ _id: { "$oid": "507f1f77bcf86cd799439011" } });
		typeof obj._id
	`)
	require.NoError(t, err)
	// After deserializing, the $oid should become an ObjectID
	assert.Equal(t, "object", val.String())
}

func TestEJSON_Deserialize_NoArgs_Panics(t *testing.T) {
	rt := setupEJSONRuntime(t)
	_, err := rt.RunString(`EJSON.deserialize()`)
	assert.Error(t, err)
}

func TestEJSON_Roundtrip_StringifyParse(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var original = { name: "test", count: 42 };
		var json = EJSON.stringify(original);
		var parsed = EJSON.parse(json);
		parsed.name === "test"
	`)
	require.NoError(t, err)
	assert.Equal(t, true, val.Export())
}

func TestEJSON_Roundtrip_SerializeDeserialize(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`
		var original = { _id: ObjectId("507f1f77bcf86cd799439011"), name: "test" };
		var serialized = EJSON.serialize(original);
		var deserialized = EJSON.deserialize(serialized);
		typeof deserialized._id === "object" && deserialized.name === "test"
	`)
	require.NoError(t, err)
	assert.Equal(t, true, val.Export())
}

func TestEJSON_IsRegisteredAsGlobal(t *testing.T) {
	rt := setupEJSONRuntime(t)
	val, err := rt.RunString(`typeof EJSON`)
	require.NoError(t, err)
	assert.Equal(t, "object", val.String())
}

func TestEJSON_HasAllMethods(t *testing.T) {
	rt := setupEJSONRuntime(t)
	methods := []string{"stringify", "parse", "serialize", "deserialize"}
	for _, m := range methods {
		val, err := rt.RunString(`typeof EJSON.` + m)
		require.NoError(t, err, "method %s", m)
		assert.Equal(t, "function", val.Export(), "EJSON.%s should be a function", m)
	}
}
