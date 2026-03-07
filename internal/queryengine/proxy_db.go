package queryengine

import "github.com/dop251/goja"

type databaseProxy struct {
	runtime *goja.Runtime
	dbName  string
}

// newDatabaseProxy creates a goja Proxy object that intercepts property access.
// Accessing db.someCollection returns a collection proxy for "someCollection".
// Accessing db.getName returns a function that returns the database name.
func newDatabaseProxy(rt *goja.Runtime, dbName string) goja.Value {
	dp := &databaseProxy{runtime: rt, dbName: dbName}

	proxy := rt.NewProxy(rt.NewObject(), &goja.ProxyTrapConfig{
		Get: func(target *goja.Object, property string, receiver goja.Value) (value goja.Value) {
			if property == "getName" {
				return rt.ToValue(func() string { return dp.dbName })
			}
			return newCollectionProxy(rt, property)
		},
	})

	return rt.ToValue(proxy)
}
