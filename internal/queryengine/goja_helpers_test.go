package queryengine

import (
	"testing"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestToGojaValue_EmptyDocuments(t *testing.T) {
	rt := goja.New()
	result := models.QueryResult{Documents: []any{}}
	val := toGojaValue(rt, result)
	exported := val.Export()
	arr, ok := exported.([]any)
	require.True(t, ok)
	assert.Empty(t, arr)
}

func TestToGojaValue_SingleDocument(t *testing.T) {
	rt := goja.New()
	result := models.QueryResult{Documents: []any{
		map[string]any{"name": "alice", "age": 30},
	}}
	val := toGojaValue(rt, result)
	exported := val.Export()
	arr, ok := exported.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 1)
}

func TestToGojaValue_SingleFlag_UnwrapsToObject(t *testing.T) {
	rt := goja.New()
	result := models.QueryResult{
		Single: true,
		Documents: []any{
			map[string]any{"acknowledged": true, "insertedIds": []any{"a", "b", "c"}},
		},
	}
	val := toGojaValue(rt, result)
	m, ok := val.Export().(map[string]any)
	require.True(t, ok, "Single=true should unwrap to a map, got %T", val.Export())
	assert.Equal(t, true, m["acknowledged"])
	ids, ok := m["insertedIds"].([]any)
	require.True(t, ok)
	assert.Len(t, ids, 3)
}

func TestToGojaValue_SingleFlagEmpty_ReturnsEmptyArray(t *testing.T) {
	rt := goja.New()
	result := models.QueryResult{Single: true, Documents: []any{}}
	val := toGojaValue(rt, result)
	arr, ok := val.Export().([]any)
	require.True(t, ok)
	assert.Empty(t, arr)
}

func TestToGojaValue_RawOutput(t *testing.T) {
	rt := goja.New()
	result := models.QueryResult{RawOutput: "hello world"}
	val := toGojaValue(rt, result)
	assert.Equal(t, "hello world", val.Export())
}

func TestExportArgs_Empty(t *testing.T) {
	call := goja.FunctionCall{Arguments: []goja.Value{}}
	args := exportArgs(call)
	assert.Empty(t, args)
}

func TestExportArgs_SingleArg(t *testing.T) {
	rt := goja.New()
	call := goja.FunctionCall{Arguments: []goja.Value{rt.ToValue(map[string]any{"name": "bob"})}}
	args := exportArgs(call)
	assert.Len(t, args, 1)
	assert.Equal(t, bson.D{{Key: "name", Value: "bob"}}, args[0])
}

func TestExportValue_Regex_ConvertsToBSONRegex(t *testing.T) {
	rt := goja.New()
	val, err := rt.RunString(`/foo/i`)
	require.NoError(t, err)

	exported := exportValue(val)
	m, ok := exported.(map[string]any)
	require.True(t, ok, "expected map, got %T", exported)

	bsonVal, ok := m["__bsonValue"]
	require.True(t, ok, "expected __bsonValue key")

	w, ok := bsonVal.(*bsonWrapper)
	require.True(t, ok, "expected *bsonWrapper, got %T", bsonVal)

	regex, ok := w.Value.(bson.Regex)
	require.True(t, ok, "expected bson.Regex, got %T", w.Value)
	assert.Equal(t, "foo", regex.Pattern)
	assert.Equal(t, "i", regex.Options)
}

func TestExportValue_NestedRegex_PreservedInDocument(t *testing.T) {
	rt := goja.New()
	val, err := rt.RunString(`({ name: /bar/im })`)
	require.NoError(t, err)

	exported := exportValue(val)
	doc, ok := exported.(bson.D)
	require.True(t, ok, "expected bson.D, got %T", exported)
	m, _ := asMap(doc)

	nameVal, ok := m["name"].(map[string]any)
	require.True(t, ok, "expected name to be map, got %T", m["name"])

	w, ok := nameVal["__bsonValue"].(*bsonWrapper)
	require.True(t, ok)

	regex, ok := w.Value.(bson.Regex)
	require.True(t, ok)
	assert.Equal(t, "bar", regex.Pattern)
	assert.Equal(t, "im", regex.Options)
}

func TestExportValue_RegexInArray_Preserved(t *testing.T) {
	rt := goja.New()
	val, err := rt.RunString(`[/abc/, /def/s]`)
	require.NoError(t, err)

	exported := exportValue(val)
	arr, ok := exported.([]any)
	require.True(t, ok, "expected []any, got %T", exported)
	require.Len(t, arr, 2)

	for i, expected := range []struct {
		pattern string
		options string
	}{
		{"abc", ""},
		{"def", "s"},
	} {
		m, ok := arr[i].(map[string]any)
		require.True(t, ok)
		w, ok := m["__bsonValue"].(*bsonWrapper)
		require.True(t, ok)
		regex, ok := w.Value.(bson.Regex)
		require.True(t, ok)
		assert.Equal(t, expected.pattern, regex.Pattern)
		assert.Equal(t, expected.options, regex.Options)
	}
}

func TestMongoRegexOptions_MapsJSFlags(t *testing.T) {
	assert.Equal(t, "im", mongoRegexOptions("gi"+"m"))
	assert.Equal(t, "ims", mongoRegexOptions("gimsu"))
	assert.Equal(t, "", mongoRegexOptions("guy"))
	assert.Equal(t, "i", mongoRegexOptions("i"))
}

// TestExportValue_PreservesKeyOrder guards issue #301: MongoDB reads the
// first key of a command document as the command name, so a randomly ordered
// map turned { update: ..., updates: [...], ordered: false } into "no such
// command: 'ordered'".
func TestExportValue_PreservesKeyOrder(t *testing.T) {
	rt := goja.New()
	registerBSONTypes(rt)
	val, err := rt.RunString(`({
		update: "coll",
		updates: [{ q: { a: 1, b: 2, c: 3 }, u: { $set: { z: 1, y: 2, x: 3 } }, upsert: true }],
		ordered: false,
		n: NumberLong("5")
	})`)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		converted := convertToBson(exportValue(val))
		doc, ok := converted.(bson.D)
		require.True(t, ok, "expected bson.D, got %T", converted)
		require.Len(t, doc, 4)
		assert.Equal(t, []string{"update", "updates", "ordered", "n"}, docKeys(doc))
		assert.Equal(t, int64(5), doc[3].Value)

		updates, ok := doc[1].Value.(bson.A)
		require.True(t, ok)
		stmt, ok := updates[0].(bson.D)
		require.True(t, ok)
		assert.Equal(t, []string{"q", "u", "upsert"}, docKeys(stmt))
		assert.Equal(t, []string{"a", "b", "c"}, docKeys(stmt[0].Value.(bson.D)))
		set := stmt[1].Value.(bson.D)[0].Value.(bson.D)
		assert.Equal(t, []string{"z", "y", "x"}, docKeys(set))
	}
}

func docKeys(doc bson.D) []string {
	keys := make([]string, len(doc))
	for i, e := range doc {
		keys[i] = e.Key
	}
	return keys
}
