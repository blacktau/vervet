package jsmodules

import (
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func registerPath(r *require.Registry) {
	r.RegisterNativeModule("path", func(rt *goja.Runtime, m *goja.Object) {
		exports := m.Get("exports").(*goja.Object)
		_ = exports.Set("sep", string(filepath.Separator))
		_ = exports.Set("join", func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, a := range call.Arguments {
				parts[i] = a.String()
			}
			return rt.ToValue(filepath.Join(parts...))
		})
		_ = exports.Set("resolve", func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, a := range call.Arguments {
				parts[i] = a.String()
			}
			abs, err := filepath.Abs(filepath.Join(parts...))
			if err != nil {
				panic(rt.ToValue(err.Error()))
			}
			return rt.ToValue(abs)
		})
		_ = exports.Set("dirname", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(filepath.Dir(call.Argument(0).String()))
		})
		_ = exports.Set("basename", func(call goja.FunctionCall) goja.Value {
			base := filepath.Base(call.Argument(0).String())
			if len(call.Arguments) > 1 {
				ext := call.Argument(1).String()
				base = strings.TrimSuffix(base, ext)
			}
			return rt.ToValue(base)
		})
		_ = exports.Set("extname", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(filepath.Ext(call.Argument(0).String()))
		})
		_ = exports.Set("isAbsolute", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(filepath.IsAbs(call.Argument(0).String()))
		})
		_ = exports.Set("normalize", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(filepath.Clean(call.Argument(0).String()))
		})
		_ = exports.Set("relative", func(call goja.FunctionCall) goja.Value {
			rel, err := filepath.Rel(call.Argument(0).String(), call.Argument(1).String())
			if err != nil {
				panic(rt.ToValue(err.Error()))
			}
			return rt.ToValue(rel)
		})
		_ = exports.Set("parse", func(call goja.FunctionCall) goja.Value {
			p := call.Argument(0).String()
			dir := filepath.Dir(p)
			base := filepath.Base(p)
			ext := filepath.Ext(p)
			name := strings.TrimSuffix(base, ext)
			root := ""
			if filepath.IsAbs(p) {
				root = string(filepath.Separator)
			}
			return rt.ToValue(map[string]any{
				"root": root,
				"dir":  dir,
				"base": base,
				"ext":  ext,
				"name": name,
			})
		})
	})
}
