package queryengine

import (
	"testing"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
