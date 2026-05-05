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

func TestFS_WriteFileSyncRoundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.txt")

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	_, err := rt.RunString(`require('fs').writeFileSync(P, 'hello')`)
	require.NoError(t, err)

	got, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestFS_AppendFileSync(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "log.txt")
	require.NoError(t, os.WriteFile(p, []byte("a"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	_, err := rt.RunString(`require('fs').appendFileSync(P, 'b')`)
	require.NoError(t, err)

	got, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "ab", string(got))
}

func TestFS_MkdirSyncRecursive(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", target))
	_, err := rt.RunString(`require('fs').mkdirSync(P, {recursive: true})`)
	require.NoError(t, err)

	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFS_RmSyncRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "f"), []byte(""), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", sub))
	_, err := rt.RunString(`require('fs').rmSync(P, {recursive: true, force: true})`)
	require.NoError(t, err)

	_, err = os.Stat(sub)
	assert.True(t, os.IsNotExist(err))
}

func TestFS_UnlinkSync(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	require.NoError(t, os.WriteFile(p, []byte(""), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("P", p))
	_, err := rt.RunString(`require('fs').unlinkSync(P)`)
	require.NoError(t, err)

	_, err = os.Stat(p)
	assert.True(t, os.IsNotExist(err))
}

func TestFS_RenameSync(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	require.NoError(t, os.WriteFile(a, []byte("x"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("A", a))
	require.NoError(t, rt.Set("B", b))
	_, err := rt.RunString(`require('fs').renameSync(A, B)`)
	require.NoError(t, err)

	got, err := os.ReadFile(b)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))
}

func TestFS_CopyFileSync(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	require.NoError(t, os.WriteFile(a, []byte("x"), 0o644))

	rt := newTestRuntime(t)
	require.NoError(t, rt.Set("A", a))
	require.NoError(t, rt.Set("B", b))
	_, err := rt.RunString(`require('fs').copyFileSync(A, B)`)
	require.NoError(t, err)

	got, err := os.ReadFile(b)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))
	_, err = os.Stat(a)
	require.NoError(t, err)
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
