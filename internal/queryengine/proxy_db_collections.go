package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

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

// dbCreateView returns a function: db.createView(name, source, pipeline, options?) → { ok: 1 }
// Equivalent to db.runCommand({ create: name, viewOn: source, pipeline: [...], ...options }).
func dbCreateView(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 3 {
			panic(ec.rt.NewGoError(fmt.Errorf("createView requires name, source, and pipeline arguments")))
		}

		name := call.Arguments[0].String()
		source := call.Arguments[1].String()

		pipelineRaw := exportValue(call.Arguments[2])
		pipeline := convertToBson(pipelineRaw)

		cmd := bson.D{
			{Key: "create", Value: name},
			{Key: "viewOn", Value: source},
			{Key: "pipeline", Value: pipeline},
		}

		if len(call.Arguments) >= 4 && !goja.IsUndefined(call.Arguments[3]) {
			optsRaw := exportValue(call.Arguments[3])
			if optsDoc, ok := convertToBson(optsRaw).(bson.D); ok {
				cmd = append(cmd, optsDoc...)
			}
		}

		var result bson.M
		err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result)
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("createView: %w", err)))
		}

		return ec.rt.ToValue(result)
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
