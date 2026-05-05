package jsmodules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS_ReadFileSyncUtf8(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(p, []byte("hello"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	val, err := rt.RunString(`require('fs').readFileSync(P, 'utf8')`)
	require.NoError(t, err)
	assert.Equal(t, "hello", val.Export())
}

func TestFS_ReadFileSyncOptionsObject(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(p, []byte("hi"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	val, err := rt.RunString(`require('fs').readFileSync(P, {encoding: 'utf8'})`)
	require.NoError(t, err)
	assert.Equal(t, "hi", val.Export())
}

func TestFS_ReadFileSyncMissingErrors(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`
		try {
			require('fs').readFileSync('/definitely/not/here.txt', 'utf8');
			'no-error';
		} catch (e) {
			e.code;
		}
	`)
	require.NoError(t, err)
	assert.Equal(t, "ENOENT", val.Export())
}

func TestFS_ExistsSync(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(p, []byte("hi"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	val, err := rt.RunString(`[require('fs').existsSync(P), require('fs').existsSync('/nope/nope')]`)
	require.NoError(t, err)
	assert.Equal(t, []any{true, false}, val.Export())
}

func TestFS_StatSyncFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(p, []byte("12345"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	val, err := rt.RunString(`
		const s = require('fs').statSync(P);
		[s.size, s.isFile(), s.isDirectory()]
	`)
	require.NoError(t, err)
	assert.Equal(t, []any{int64(5), true, false}, val.Export())
}

func TestFS_StatSyncDirectory(t *testing.T) {
	dir := t.TempDir()
	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", dir))
	val, err := rt.RunString(`
		const s = require('fs').statSync(P);
		[s.isFile(), s.isDirectory()]
	`)
	require.NoError(t, err)
	assert.Equal(t, []any{false, true}, val.Export())
}

func TestFS_ReaddirSync(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b"), []byte(""), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", dir))
	val, err := rt.RunString(`require('fs').readdirSync(P).sort()`)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, val.Export())
}
