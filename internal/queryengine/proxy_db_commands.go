package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

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
