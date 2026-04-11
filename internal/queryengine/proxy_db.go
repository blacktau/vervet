package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

// newDatabaseProxy creates a Goja Proxy object that intercepts property access.
// Accessing db.someCollection returns a collection proxy for "someCollection".
// Known db-level methods (getName, getCollection, runCommand, adminCommand)
// are intercepted and return Go-backed functions.
func newDatabaseProxy(ec *execContext) goja.Value {
	proxy := ec.rt.NewProxy(ec.rt.NewObject(), &goja.ProxyTrapConfig{
		Get: func(target *goja.Object, property string, receiver goja.Value) (value goja.Value) {
			switch property {
			case "getName":
				return ec.rt.ToValue(func() string { return ec.dbName })
			case "getCollection":
				return ec.rt.ToValue(func(name string) goja.Value {
					return newCollectionProxy(ec, name)
				})
			case "runCommand":
				return ec.rt.ToValue(dbRunCommand(ec, ec.dbName))
			case "adminCommand":
				return ec.rt.ToValue(dbRunCommand(ec, "admin"))
			}
			return newCollectionProxy(ec, property)
		},
	})

	return ec.rt.ToValue(proxy)
}

// dbRunCommand returns a Goja-callable function that executes a command document
// against the specified database via client.Database(dbName).RunCommand().
func dbRunCommand(ec *execContext, dbName string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if ec.client == nil {
			panic(ec.rt.NewGoError(fmt.Errorf("no MongoDB client available")))
		}
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("runCommand requires a command document")))
		}

		cmdRaw := exportValue(call.Arguments[0])
		cmdDoc := convertToBson(cmdRaw)

		var result bson.M
		err := ec.client.Database(dbName).RunCommand(ec.ctx, cmdDoc).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("runCommand: %w", err)))
		}

		return ec.rt.ToValue(result)
	}
}
