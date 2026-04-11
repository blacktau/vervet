package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

// dbMethodNames lists all intercepted db-level method names so the proxy
// can distinguish them from collection name access.
var dbMethodNames = map[string]bool{
	"getName":            true,
	"getCollection":      true,
	"runCommand":         true,
	"adminCommand":       true,
	"getCollectionNames": true,
	"getCollectionInfos": true,
	"createCollection":   true,
	"dropDatabase":       true,
	"stats":              true,
	"version":            true,
	"getSiblingDB":       true,
	"getMongo":           true,
	"aggregate":          true,
}

// newDatabaseProxy creates a Goja Proxy object that intercepts property access.
// Accessing db.someCollection returns a collection proxy for "someCollection".
// Known db-level methods are intercepted and return Go-backed functions.
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
			case "getCollectionNames":
				return ec.rt.ToValue(dbGetCollectionNames(ec))
			case "getCollectionInfos":
				return ec.rt.ToValue(dbGetCollectionInfos(ec))
			case "createCollection":
				return ec.rt.ToValue(dbCreateCollection(ec))
			case "dropDatabase":
				return ec.rt.ToValue(dbDropDatabase(ec))
			case "stats":
				return ec.rt.ToValue(dbStats(ec))
			case "version":
				return ec.rt.ToValue(dbVersion(ec))
			case "getSiblingDB":
				return ec.rt.ToValue(dbGetSiblingDB(ec))
			case "getMongo":
				return ec.rt.ToValue(dbGetMongo(ec))
			case "aggregate":
				return ec.rt.ToValue(dbAggregate(ec))
			}
			return newCollectionProxy(ec, property)
		},
	})

	return ec.rt.ToValue(proxy)
}

func requireClient(ec *execContext) {
	if ec.client == nil {
		panic(ec.rt.NewGoError(fmt.Errorf("no MongoDB client available")))
	}
}

// dbRunCommand returns a Goja-callable function that executes a command document
// against the specified database via client.Database(dbName).RunCommand().
func dbRunCommand(ec *execContext, dbName string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
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

// dbGetCollectionNames returns a function: db.getCollectionNames(filter?) → string[]
func dbGetCollectionNames(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		filter := bson.D{}
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			raw := exportValue(call.Arguments[0])
			if converted, ok := convertToBson(raw).(bson.D); ok {
				filter = converted
			}
		}

		names, err := ec.client.Database(ec.dbName).ListCollectionNames(ec.ctx, filter)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("getCollectionNames: %w", err)))
		}

		return ec.rt.ToValue(names)
	}
}

// dbGetCollectionInfos returns a function: db.getCollectionInfos(filter?) → object[]
func dbGetCollectionInfos(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		filter := bson.D{}
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			raw := exportValue(call.Arguments[0])
			if converted, ok := convertToBson(raw).(bson.D); ok {
				filter = converted
			}
		}

		cursor, err := ec.client.Database(ec.dbName).ListCollections(ec.ctx, filter)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("getCollectionInfos: %w", err)))
		}

		var results []bson.M
		if err := cursor.All(ec.ctx, &results); err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("getCollectionInfos: %w", err)))
		}

		// Convert []bson.M to []any for Goja
		out := make([]any, len(results))
		for i, r := range results {
			out[i] = r
		}

		return ec.rt.ToValue(out)
	}
}

// dbCreateCollection returns a function: db.createCollection(name) → { ok: 1 }
func dbCreateCollection(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("createCollection requires a name argument")))
		}

		name := call.Arguments[0].String()
		err := ec.client.Database(ec.dbName).CreateCollection(ec.ctx, name)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("createCollection: %w", err)))
		}

		return ec.rt.ToValue(map[string]any{"ok": 1})
	}
}

// dbDropDatabase returns a function: db.dropDatabase() → { ok: 1, dropped: "dbName" }
func dbDropDatabase(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		err := ec.client.Database(ec.dbName).Drop(ec.ctx)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("dropDatabase: %w", err)))
		}

		return ec.rt.ToValue(map[string]any{"ok": 1, "dropped": ec.dbName})
	}
}

// dbStats returns a function: db.stats() → object (runs {dbStats: 1})
func dbStats(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		var result bson.M
		err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("stats: %w", err)))
		}

		return ec.rt.ToValue(result)
	}
}

// dbVersion returns a function: db.version() → string
func dbVersion(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		var result bson.M
		err := ec.client.Database("admin").RunCommand(ec.ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("version: %w", err)))
		}

		ver, _ := result["version"].(string)
		return ec.rt.ToValue(ver)
	}
}

// dbGetSiblingDB returns a function: db.getSiblingDB(name) → db proxy for that database
func dbGetSiblingDB(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("getSiblingDB requires a database name")))
		}

		siblingName := call.Arguments[0].String()
		siblingEC := &execContext{
			ctx:    ec.ctx,
			client: ec.client,
			dbName: siblingName,
			rt:     ec.rt,
		}
		return newDatabaseProxy(siblingEC)
	}
}

// dbGetMongo returns a function: db.getMongo() → a simple object representing the connection.
// In Vervet this is a stub since connections are managed by the app, not scripts.
func dbGetMongo(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		obj := ec.rt.NewObject()
		_ = obj.Set("getDB", func(name string) goja.Value {
			siblingEC := &execContext{
				ctx:    ec.ctx,
				client: ec.client,
				dbName: name,
				rt:     ec.rt,
			}
			return newDatabaseProxy(siblingEC)
		})
		return obj
	}
}

// dbAggregate returns a function: db.aggregate(pipeline) → results array
func dbAggregate(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("aggregate requires a pipeline argument")))
		}

		pipelineRaw := exportValue(call.Arguments[0])
		pipeline := convertToBson(pipelineRaw)

		cursor, err := ec.client.Database(ec.dbName).Aggregate(ec.ctx, pipeline)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("aggregate: %w", err)))
		}

		var results []bson.M
		if err := cursor.All(ec.ctx, &results); err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("aggregate: %w", err)))
		}

		out := make([]any, len(results))
		for i, r := range results {
			out[i] = r
		}

		return ec.rt.ToValue(out)
	}
}
