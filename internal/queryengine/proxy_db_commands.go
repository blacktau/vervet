package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// dbRunCommand returns a Goja-callable function that executes a command document
// against the specified database via client.Database(dbName).RunCommand().
func dbRunCommand(ec *execContext, dbName string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("runCommand requires a command document")))
		}

		cmdDoc := convertToBson(exportValue(call.Arguments[0]))

		var result bson.D
		err := ec.client.Database(dbName).RunCommand(ec.ctx, cmdDoc).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("runCommand: %w", err)))
		}

		return toJSValue(ec.rt, markCursorBatches(cmdDoc, result))
	}
}

// markCursorBatches marks the documents in a find, aggregate or getMore reply as
// read from a collection, so they convert as find() results do and their longs
// stay Long when the script writes them back. The rest of the reply stays in
// command mode, as do the batches of other cursor commands (listCollections,
// listIndexes), whose documents are metadata rather than stored documents.
func markCursorBatches(cmd any, reply bson.D) bson.D {
	doc, ok := cmd.(bson.D)
	if !ok || len(doc) == 0 {
		return reply
	}
	switch doc[0].Key {
	case "find", "aggregate", "getMore":
	default:
		return reply
	}
	cursor, ok := docField(reply, "cursor").(bson.D)
	if !ok {
		return reply
	}
	marked := make(bson.D, len(cursor))
	for i, e := range cursor {
		marked[i] = e
		if batch, ok := e.Value.(bson.A); ok && (e.Key == "firstBatch" || e.Key == "nextBatch") {
			marked[i].Value = resultValues(batch)
		}
	}
	out := make(bson.D, len(reply))
	for i, e := range reply {
		out[i] = e
		if e.Key == "cursor" {
			out[i].Value = marked
		}
	}
	return out
}

// dbStats returns a function: db.stats() → object (runs {dbStats: 1})
func dbStats(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)

		var result bson.D
		err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("stats: %w", err)))
		}

		return toJSValue(ec.rt, result)
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
