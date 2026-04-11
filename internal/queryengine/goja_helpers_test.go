package queryengine

import (
	"testing"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	m, ok := args[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bob", m["name"])
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

	regex, ok := w.Value.(primitive.Regex)
	require.True(t, ok, "expected primitive.Regex, got %T", w.Value)
	assert.Equal(t, "foo", regex.Pattern)
	assert.Equal(t, "i", regex.Options)
}

func TestExportValue_NestedRegex_PreservedInDocument(t *testing.T) {
	rt := goja.New()
	val, err := rt.RunString(`({ name: /bar/im })`)
	require.NoError(t, err)

	exported := exportValue(val)
	m, ok := exported.(map[string]any)
	require.True(t, ok, "expected map, got %T", exported)

	nameVal, ok := m["name"].(map[string]any)
	require.True(t, ok, "expected name to be map, got %T", m["name"])

	w, ok := nameVal["__bsonValue"].(*bsonWrapper)
	require.True(t, ok)

	regex, ok := w.Value.(primitive.Regex)
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
		regex, ok := w.Value.(primitive.Regex)
		require.True(t, ok)
		assert.Equal(t, expected.pattern, regex.Pattern)
		assert.Equal(t, expected.options, regex.Options)
	}
}

func TestMongoRegexOptions_MapsJSFlags(t *testing.T) {
	assert.Equal(t, "im", mongoRegexOptions("gi" + "m"))
	assert.Equal(t, "ims", mongoRegexOptions("gimsu"))
	assert.Equal(t, "", mongoRegexOptions("guy"))
	assert.Equal(t, "i", mongoRegexOptions("i"))
}
