// Package jsmodules registers Node-compatible built-in modules
// (fs, path, os, crypto) onto a goja_nodejs require.Registry.
package jsmodules

import (
	"errors"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// RegisterAll registers fs, path, os, and crypto on the given registry.
func RegisterAll(r *require.Registry) {
	registerPath(r)
	registerOS(r)
	registerCrypto(r)
	registerFS(r)
}

// nodeError builds a JS Error with a Node-style `code` property.
// Pass to panic() to surface as a script error from the runtime.
// The code is prefixed onto the message so it surfaces in Go errors too.
func nodeError(rt *goja.Runtime, code, msg string) *goja.Object {
	full := msg
	if code != "" {
		full = code + ": " + msg
	}
	obj := rt.NewGoError(errors.New(full))
	_ = obj.Set("code", code)
	return obj
}
