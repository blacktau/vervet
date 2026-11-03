package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
)

// dbGetSiblingDB returns a function: db.getSiblingDB(name) → db proxy for that database
func dbGetSiblingDB(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("getSiblingDB requires a database name")))
		}

		return newDatabaseProxy(ec.sibling(call.Arguments[0].String()))
	}
}

// dbGetMongo returns a function: db.getMongo() → a simple object representing the connection.
// In Vervet this is a stub since connections are managed by the app, not scripts.
func dbGetMongo(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		obj := ec.rt.NewObject()
		_ = obj.Set("getDB", func(name string) goja.Value {
			return newDatabaseProxy(ec.sibling(name))
		})
		return obj
	}
}
