package jsmodules

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	gojarequire "github.com/dop251/goja_nodejs/require"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRuntime(t *testing.T) *goja.Runtime {
	t.Helper()
	return newTestRuntimeIn(t, "")
}

// newTestRuntimeIn builds a runtime whose relative paths resolve against base.
func newTestRuntimeIn(t *testing.T, base string) *goja.Runtime {
	t.Helper()
	registry := gojarequire.NewRegistry()
	RegisterAll(registry, base)
	rt := goja.New()
	registry.Enable(rt)
	buffer.Enable(rt)
	return rt
}

func TestRegisterAll_EnablesRequire(t *testing.T) {
	rt := newTestRuntime(t)
	val, err := rt.RunString(`typeof require`)
	require.NoError(t, err)
	assert.Equal(t, "function", val.Export())
}
