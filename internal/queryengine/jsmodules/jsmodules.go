// Package jsmodules registers Node-compatible built-in modules
// (fs, path, os, crypto) onto a goja_nodejs require.Registry.
package jsmodules

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// RegisterAll registers fs, path, os, and crypto on the given registry.
// baseDir is the directory relative paths resolve against — the running
// script's own directory. An empty baseDir leaves relative paths to the
// process working directory.
func RegisterAll(r *require.Registry, baseDir string) {
	registerPath(r, baseDir)
	registerOS(r)
	registerCrypto(r)
	registerFS(r, baseDir)
}

// resolve makes a script-supplied path absolute against baseDir. Absolute
// paths are used as given, so a script can always reach outside its own
// directory by saying so explicitly.
func resolve(baseDir, p string) string {
	if baseDir == "" || filepath.IsAbs(p) {
		return p
	}
	if isRooted(p) {
		// Windows `\temp\x` or `/temp/x`: absolute to Node, on the script's drive.
		return filepath.VolumeName(baseDir) + p
	}
	return filepath.Join(baseDir, p)
}

// isAbs reports whether Node treats p as absolute. On Windows Node counts a
// path starting with a separator as absolute (on the current drive), but
// filepath.IsAbs needs a drive letter or UNC prefix.
func isAbs(p string) bool {
	return filepath.IsAbs(p) || isRooted(p)
}

// isRooted reports whether p starts with a path separator and has no volume.
func isRooted(p string) bool {
	return p != "" && os.IsPathSeparator(p[0]) && filepath.VolumeName(p) == ""
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
