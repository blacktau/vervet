package queryengine

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeScript drops a file in dir and returns its full path.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

func TestGojaEngine_Load_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	helper := writeScript(t, dir, "helper.js", `function helperFn() { return 42; }`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`load(`+strconv.Quote(helper)+`); helperFn()`)
	require.NoError(t, err)
	assert.Equal(t, "42", res.RawOutput)
}

func TestGojaEngine_Load_RelativeToScriptDir(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "helper.js", `function helperFn() { return 7; }`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`load('helper.js'); helperFn()`)
	require.NoError(t, err)
	assert.Equal(t, "7", res.RawOutput)
}

func TestGojaEngine_Load_DotSlashRelative(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "helper.js", `function helperFn() { return 1; }`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`load('./helper.js'); helperFn()`)
	require.NoError(t, err)
	assert.Equal(t, "1", res.RawOutput)
}

// mongosh exposes a loaded file's top-level const/let/var and function
// declarations to the caller, and load() itself evaluates to true.
func TestGojaEngine_Load_SharesGlobalsAndReturnsTrue(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "helper.js", `
		const HELPER_CONST = 7;
		let helperLet = 8;
		var helperVar = 9;
		function helperFn() { return 10; }
	`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb", `
		const returned = load('helper.js');
		[returned, HELPER_CONST, helperLet, helperVar, helperFn()]
	`)
	require.NoError(t, err)
	assert.Equal(t, []any{true, int64(7), int64(8), int64(9), int64(10)}, res.Documents)
}

// A loaded file resolves its own load() calls the same way the top-level
// script does — against the script's directory, matching mongosh's
// cwd-based behaviour where the base never rebases per file.
func TestGojaEngine_Load_NestedUsesSameBase(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o755))
	writeScript(t, dir, "leaf.js", `function leafFn() { return 3; }`)
	writeScript(t, filepath.Join(dir, "nested"), "inner.js", `load('leaf.js');`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`load('nested/inner.js'); leafFn()`)
	require.NoError(t, err)
	assert.Equal(t, "3", res.RawOutput)
}

func TestGojaEngine_Load_MissingFileErrors(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	_, err := eng.ExecuteQuery(context.Background(), "", "testdb", `load('nope.js')`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not open file")
	assert.Contains(t, err.Error(), filepath.Join(dir, "nope.js"))
}

// An error thrown by a loaded file propagates to the caller rather than
// being swallowed.
func TestGojaEngine_Load_PropagatesScriptError(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "boom.js", `throw new Error('helper exploded');`)
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	_, err := eng.ExecuteQuery(context.Background(), "", "testdb", `load('boom.js')`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "helper exploded")
}

func TestGojaEngine_ScriptGlobals_SavedTab(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`[__filename, __dirname, process.cwd()]`)
	require.NoError(t, err)
	assert.Equal(t, []any{main, dir, dir}, res.Documents)
}

// An unsaved tab has no path: __filename is empty and the base falls back
// to the process working directory.
func TestGojaEngine_ScriptGlobals_UnsavedTab(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	eng := NewGojaEngine(nil, 100, "")
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`[__filename, __dirname, process.cwd()]`)
	require.NoError(t, err)
	assert.Equal(t, []any{"", wd, wd}, res.Documents)
}

func TestGojaEngine_FSRelativePath_ResolvesAgainstScriptDir(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "data.csv", "a,b\n1,2\n")
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`require('fs').readFileSync('data.csv', 'utf8').trim().split('\n').length`)
	require.NoError(t, err)
	assert.Equal(t, "2", res.RawOutput)
}

func TestGojaEngine_FSWriteRelativePath_LandsInScriptDir(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	_, err := eng.ExecuteQuery(context.Background(), "", "testdb",
		`require('fs').writeFileSync('out.txt', 'written')`)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "written", string(data))
}

// The canonical mongosh idiom for reading a data file next to a script.
func TestGojaEngine_DirnameJoin_ReadsDataFile(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "data.csv", "name,age\nalice,30\n")
	main := filepath.Join(dir, "main.js")

	eng := NewGojaEngine(nil, 100, main)
	res, err := eng.ExecuteQuery(context.Background(), "", "testdb", `
		const fs = require('fs');
		const path = require('path');
		const rows = fs.readFileSync(path.join(__dirname, 'data.csv'), 'utf8').trim().split('\n');
		rows[1]
	`)
	require.NoError(t, err)
	assert.Equal(t, "alice,30", res.RawOutput)
}

// Each execution gets its own base — an engine built for one script must not
// leak its directory into the next.
func TestGojaEngine_BaseDirIsPerEngine(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeScript(t, dirA, "helper.js", `function which() { return 'A'; }`)
	writeScript(t, dirB, "helper.js", `function which() { return 'B'; }`)

	engA := NewGojaEngine(nil, 100, filepath.Join(dirA, "main.js"))
	resA, err := engA.ExecuteQuery(context.Background(), "", "testdb", `load('helper.js'); which()`)
	require.NoError(t, err)
	assert.Equal(t, "A", resA.RawOutput)

	engB := NewGojaEngine(nil, 100, filepath.Join(dirB, "main.js"))
	resB, err := engB.ExecuteQuery(context.Background(), "", "testdb", `load('helper.js'); which()`)
	require.NoError(t, err)
	assert.Equal(t, "B", resB.RawOutput)
}
