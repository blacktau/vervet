package jsmodules

import (
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func registerOS(r *require.Registry) {
	r.RegisterNativeModule("os", func(rt *goja.Runtime, m *goja.Object) {
		exports := m.Get("exports").(*goja.Object)
		eol := "\n"
		if runtime.GOOS == "windows" {
			eol = "\r\n"
		}
		_ = exports.Set("EOL", eol)
		_ = exports.Set("homedir", func(call goja.FunctionCall) goja.Value {
			h, err := os.UserHomeDir()
			if err != nil {
				return rt.ToValue("")
			}
			return rt.ToValue(h)
		})
		_ = exports.Set("tmpdir", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(os.TempDir())
		})
		_ = exports.Set("platform", func(call goja.FunctionCall) goja.Value {
			p := runtime.GOOS
			if p == "windows" {
				p = "win32"
			}
			return rt.ToValue(p)
		})
		_ = exports.Set("arch", func(call goja.FunctionCall) goja.Value {
			a := runtime.GOARCH
			switch a {
			case "amd64":
				a = "x64"
			case "386":
				a = "ia32"
			}
			return rt.ToValue(a)
		})
		_ = exports.Set("hostname", func(call goja.FunctionCall) goja.Value {
			h, err := os.Hostname()
			if err != nil {
				return rt.ToValue("")
			}
			return rt.ToValue(h)
		})
		_ = exports.Set("userInfo", func(call goja.FunctionCall) goja.Value {
			u, err := user.Current()
			if err != nil {
				panic(nodeError(rt, "EUNKNOWN", err.Error()))
			}
			uid, _ := strconv.Atoi(u.Uid)
			gid, _ := strconv.Atoi(u.Gid)
			shell := os.Getenv("SHELL")
			return rt.ToValue(map[string]any{
				"username": u.Username,
				"homedir":  u.HomeDir,
				"shell":    shell,
				"uid":      uid,
				"gid":      gid,
			})
		})
	})
}
