package queryengine

import "github.com/dop251/goja"

// newDatabaseProxy creates a Goja Proxy object that intercepts property access.
// Accessing db.someCollection returns a collection proxy for "someCollection".
// Accessing db.getName returns a function that returns the database name.
func newDatabaseProxy(ec *execContext) goja.Value {
	proxy := ec.rt.NewProxy(ec.rt.NewObject(), &goja.ProxyTrapConfig{
		Get: func(target *goja.Object, property string, receiver goja.Value) (value goja.Value) {
			if property == "getName" {
				return ec.rt.ToValue(func() string { return ec.dbName })
			}
			if property == "getCollection" {
				return ec.rt.ToValue(func(name string) goja.Value {
					return newCollectionProxy(ec, name)
				})
			}
			return newCollectionProxy(ec, property)
		},
	})

	return ec.rt.ToValue(proxy)
}
