package jsmodules

import (
	"errors"
	"io/fs"
	"os"
	"syscall"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// fsErrCode maps a Go error to a Node-style errno string.
func fsErrCode(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "ENOENT"
	case errors.Is(err, fs.ErrPermission):
		return "EACCES"
	case errors.Is(err, fs.ErrExist):
		return "EEXIST"
	case errors.Is(err, syscall.EISDIR):
		return "EISDIR"
	case errors.Is(err, syscall.ENOTDIR):
		return "ENOTDIR"
	default:
		return ""
	}
}

func panicFSErr(rt *goja.Runtime, err error) {
	panic(nodeError(rt, fsErrCode(err), err.Error()))
}

func registerFS(r *require.Registry) {
	r.RegisterNativeModule("fs", func(rt *goja.Runtime, m *goja.Object) {
		exports := m.Get("exports").(*goja.Object)

		_ = exports.Set("readFileSync", func(call goja.FunctionCall) goja.Value {
			path := call.Argument(0).String()
			encoding := ""
			if len(call.Arguments) > 1 {
				arg := call.Argument(1)
				if s, ok := arg.Export().(string); ok {
					encoding = s
				} else if obj, ok := arg.Export().(map[string]any); ok {
					if e, ok := obj["encoding"].(string); ok {
						encoding = e
					}
				}
			}
			data, err := os.ReadFile(path)
			if err != nil {
				panicFSErr(rt, err)
			}
			if encoding == "" {
				return rt.ToValue(rt.NewArrayBuffer(data))
			}
			return rt.ToValue(string(data))
		})

		_ = exports.Set("existsSync", func(call goja.FunctionCall) goja.Value {
			_, err := os.Stat(call.Argument(0).String())
			return rt.ToValue(err == nil)
		})

		_ = exports.Set("statSync", func(call goja.FunctionCall) goja.Value {
			info, err := os.Stat(call.Argument(0).String())
			if err != nil {
				panicFSErr(rt, err)
			}
			obj := rt.NewObject()
			_ = obj.Set("size", info.Size())
			_ = obj.Set("mtime", info.ModTime())
			_ = obj.Set("mtimeMs", info.ModTime().UnixMilli())
			_ = obj.Set("mode", uint32(info.Mode().Perm()))
			isFile := info.Mode().IsRegular()
			isDir := info.IsDir()
			isSymlink := info.Mode()&os.ModeSymlink != 0
			_ = obj.Set("isFile", func(goja.FunctionCall) goja.Value { return rt.ToValue(isFile) })
			_ = obj.Set("isDirectory", func(goja.FunctionCall) goja.Value { return rt.ToValue(isDir) })
			_ = obj.Set("isSymbolicLink", func(goja.FunctionCall) goja.Value { return rt.ToValue(isSymlink) })
			return obj
		})

		_ = exports.Set("readdirSync", func(call goja.FunctionCall) goja.Value {
			entries, err := os.ReadDir(call.Argument(0).String())
			if err != nil {
				panicFSErr(rt, err)
			}
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name()
			}
			return rt.ToValue(names)
		})
	})
}
