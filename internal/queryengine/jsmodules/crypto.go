package jsmodules

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/google/uuid"
)

func registerCrypto(r *require.Registry) {
	r.RegisterNativeModule("crypto", func(rt *goja.Runtime, m *goja.Object) {
		exports := m.Get("exports").(*goja.Object)

		_ = exports.Set("createHash", func(call goja.FunctionCall) goja.Value {
			alg := call.Argument(0).String()
			var h hash.Hash
			switch alg {
			case "md5":
				h = md5.New()
			case "sha1":
				h = sha1.New()
			case "sha256":
				h = sha256.New()
			case "sha512":
				h = sha512.New()
			default:
				panic(nodeError(rt, "ERR_OSSL_EVP_UNSUPPORTED", "unsupported hash algorithm: "+alg))
			}
			obj := rt.NewObject()
			_ = obj.Set("update", func(call goja.FunctionCall) goja.Value {
				h.Write([]byte(call.Argument(0).String()))
				return obj
			})
			_ = obj.Set("digest", func(call goja.FunctionCall) goja.Value {
				sum := h.Sum(nil)
				if len(call.Arguments) == 0 {
					return rt.ToValue(rt.NewArrayBuffer(sum))
				}
				switch call.Argument(0).String() {
				case "hex":
					return rt.ToValue(hex.EncodeToString(sum))
				case "base64":
					return rt.ToValue(base64.StdEncoding.EncodeToString(sum))
				default:
					panic(nodeError(rt, "ERR_INVALID_ARG_VALUE", "unsupported digest encoding"))
				}
			})
			return obj
		})

		_ = exports.Set("randomBytes", func(call goja.FunctionCall) goja.Value {
			n := int(call.Argument(0).ToInteger())
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				panic(nodeError(rt, "EUNKNOWN", err.Error()))
			}
			return rt.ToValue(rt.NewArrayBuffer(buf))
		})

		_ = exports.Set("randomUUID", func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(uuid.NewString())
		})
	})
}
