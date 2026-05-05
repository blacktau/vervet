package jsmodules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath_Join(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('path').join('a', 'b', 'c.txt')`)
	require.NoError(t, err)
	assert.Equal(t, "a/b/c.txt", val.Export())
}

func TestPath_BasenameAndExtname(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`
		const p = require('path');
		[p.basename('/foo/bar.txt'), p.extname('/foo/bar.txt'), p.dirname('/foo/bar.txt')]
	`)
	require.NoError(t, err)
	assert.Equal(t, []any{"bar.txt", ".txt", "/foo"}, val.Export())
}

func TestPath_IsAbsolute(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`[require('path').isAbsolute('/x'), require('path').isAbsolute('x')]`)
	require.NoError(t, err)
	assert.Equal(t, []any{true, false}, val.Export())
}

func TestPath_Parse(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('path').parse('/foo/bar.txt')`)
	require.NoError(t, err)
	m := val.Export().(map[string]any)
	assert.Equal(t, "/", m["root"])
	assert.Equal(t, "/foo", m["dir"])
	assert.Equal(t, "bar.txt", m["base"])
	assert.Equal(t, ".txt", m["ext"])
	assert.Equal(t, "bar", m["name"])
}

func TestPath_Sep(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('path').sep`)
	require.NoError(t, err)
	assert.Equal(t, "/", val.Export())
}
