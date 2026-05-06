package jsmodules

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOS_Homedir(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').homedir()`)
	require.NoError(t, err)
	assert.NotEmpty(t, val.Export())
}

func TestOS_Tmpdir(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').tmpdir()`)
	require.NoError(t, err)
	assert.NotEmpty(t, val.Export())
}

func TestOS_Platform(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').platform()`)
	require.NoError(t, err)
	want := map[string]string{"linux": "linux", "darwin": "darwin", "windows": "win32"}[runtime.GOOS]
	if want == "" {
		want = runtime.GOOS
	}
	assert.Equal(t, want, val.Export())
}

func TestOS_Arch(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').arch()`)
	require.NoError(t, err)
	want := map[string]string{"amd64": "x64", "arm64": "arm64", "386": "ia32"}[runtime.GOARCH]
	if want == "" {
		want = runtime.GOARCH
	}
	assert.Equal(t, want, val.Export())
}

func TestOS_Hostname(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').hostname()`)
	require.NoError(t, err)
	assert.NotEmpty(t, val.Export())
}

func TestOS_EOL(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').EOL`)
	require.NoError(t, err)
	want := "\n"
	if runtime.GOOS == "windows" {
		want = "\r\n"
	}
	assert.Equal(t, want, val.Export())
}

func TestOS_UserInfo(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`require('os').userInfo()`)
	require.NoError(t, err)
	m := val.Export().(map[string]any)
	assert.NotEmpty(t, m["username"])
	assert.NotEmpty(t, m["homedir"])
	assert.Contains(t, m, "uid")
	assert.Contains(t, m, "gid")
}
