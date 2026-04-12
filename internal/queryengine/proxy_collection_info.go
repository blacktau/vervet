package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// collStatsMap runs the collStats command and returns the result document.
func collStatsMap(ec *execContext, collName string, scale int) (bson.M, error) {
	cmd := bson.D{{Key: "collStats", Value: collName}}
	if scale > 0 {
		cmd = append(cmd, bson.E{Key: "scale", Value: scale})
	}
	var result bson.M
	if err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// collStatsField runs collStats and returns a single numeric field from the result.
func collStatsField(ec *execContext, collName, field string) (any, error) {
	stats, err := collStatsMap(ec, collName, 0)
	if err != nil {
		return nil, err
	}
	return stats[field], nil
}

// setCollectionInfoMethods attaches info/stats/legacy methods to a collection proxy object.
func setCollectionInfoMethods(obj *goja.Object, ec *execContext, collName string) {
	rt := ec.rt

	_ = obj.Set("stats", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		scale := 0
		if len(call.Arguments) > 0 {
			// Accept either a scale number or an options object { scale: n }
			arg := call.Arguments[0].Export()
			switch v := arg.(type) {
			case int64:
				scale = int(v)
			case float64:
				scale = int(v)
			case map[string]any:
				if s, ok := v["scale"]; ok {
					switch n := s.(type) {
					case int64:
						scale = int(n)
					case float64:
						scale = int(n)
					}
				}
			}
		}
		result, err := collStatsMap(ec, collName, scale)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("stats: %w", err)))
		}
		return rt.ToValue(result)
	})

	_ = obj.Set("isCapped", func() goja.Value {
		requireClient(ec)
		stats, err := collStatsMap(ec, collName, 0)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("isCapped: %w", err)))
		}
		if v, ok := stats["capped"].(bool); ok {
			return rt.ToValue(v)
		}
		return rt.ToValue(false)
	})

	_ = obj.Set("dataSize", func() goja.Value {
		requireClient(ec)
		v, err := collStatsField(ec, collName, "size")
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("dataSize: %w", err)))
		}
		return rt.ToValue(v)
	})

	_ = obj.Set("storageSize", func() goja.Value {
		requireClient(ec)
		v, err := collStatsField(ec, collName, "storageSize")
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("storageSize: %w", err)))
		}
		return rt.ToValue(v)
	})

	_ = obj.Set("totalIndexSize", func() goja.Value {
		requireClient(ec)
		v, err := collStatsField(ec, collName, "totalIndexSize")
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("totalIndexSize: %w", err)))
		}
		return rt.ToValue(v)
	})

	_ = obj.Set("totalSize", func() goja.Value {
		requireClient(ec)
		stats, err := collStatsMap(ec, collName, 0)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("totalSize: %w", err)))
		}
		total := toInt64(stats["storageSize"]) + toInt64(stats["totalIndexSize"])
		return rt.ToValue(total)
	})

	_ = obj.Set("getIndexes", func() goja.Value {
		requireClient(ec)
		op := CapturedOp{Collection: collName, Method: "listIndexes"}
		result, err := dispatch(ec.ctx, ec.client, ec.dbName, op)
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return toGojaValue(rt, result)
	})

	_ = obj.Set("count", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		filter := bson.D{}
		if len(call.Arguments) > 0 {
			filter = toBsonDoc(call.Arguments[0].Export())
		}
		count, err := ec.client.Database(ec.dbName).Collection(collName).CountDocuments(ec.ctx, filter)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("count: %w", err)))
		}
		return rt.ToValue(count)
	})

	_ = obj.Set("renameCollection", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("renameCollection requires a new name")))
		}
		newName, ok := call.Arguments[0].Export().(string)
		if !ok {
			panic(rt.NewGoError(fmt.Errorf("renameCollection: new name must be a string")))
		}
		dropTarget := false
		if len(call.Arguments) > 1 {
			if b, ok := call.Arguments[1].Export().(bool); ok {
				dropTarget = b
			}
		}
		cmd := bson.D{
			{Key: "renameCollection", Value: ec.dbName + "." + collName},
			{Key: "to", Value: ec.dbName + "." + newName},
			{Key: "dropTarget", Value: dropTarget},
		}
		var result bson.M
		if err := ec.client.Database("admin").RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("renameCollection: %w", err)))
		}
		return rt.ToValue(result)
	})

	_ = obj.Set("validate", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "validate", Value: collName}}
		if len(call.Arguments) > 0 {
			arg := call.Arguments[0].Export()
			switch v := arg.(type) {
			case bool:
				cmd = append(cmd, bson.E{Key: "full", Value: v})
			case map[string]any:
				for k, val := range v {
					cmd = append(cmd, bson.E{Key: k, Value: convertToBson(val)})
				}
			}
		}
		var result bson.M
		if err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("validate: %w", err)))
		}
		return rt.ToValue(result)
	})

	_ = obj.Set("findAndModify", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("findAndModify requires a spec document")))
		}
		spec, ok := call.Arguments[0].Export().(map[string]any)
		if !ok {
			panic(rt.NewGoError(fmt.Errorf("findAndModify: spec must be an object")))
		}
		result, err := runFindAndModify(ec, collName, spec)
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return rt.ToValue(result)
	})
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

// runFindAndModify executes the legacy findAndModify operation against the named collection.
// Delegates to FindOneAndDelete when spec.remove is truthy, otherwise FindOneAndUpdate.
func runFindAndModify(ec *execContext, collName string, spec map[string]any) (bson.M, error) {
	coll := ec.client.Database(ec.dbName).Collection(collName)
	filter := bson.D{}
	if q, ok := spec["query"]; ok && q != nil {
		filter = toBsonDoc(q)
	}

	returnNew := false
	if v, ok := spec["new"].(bool); ok {
		returnNew = v
	}
	upsert := false
	if v, ok := spec["upsert"].(bool); ok {
		upsert = v
	}

	if remove, _ := spec["remove"].(bool); remove {
		opts := options.FindOneAndDelete()
		if sort, ok := spec["sort"]; ok && sort != nil {
			opts.SetSort(toBsonDoc(sort))
		}
		if fields, ok := spec["fields"]; ok && fields != nil {
			opts.SetProjection(toBsonDoc(fields))
		}
		var result bson.M
		err := coll.FindOneAndDelete(ec.ctx, filter, opts).Decode(&result)
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("findAndModify: %w", err)
		}
		return result, nil
	}

	update, ok := spec["update"]
	if !ok || update == nil {
		return nil, fmt.Errorf("findAndModify requires either remove: true or an update document")
	}

	opts := options.FindOneAndUpdate()
	if returnNew {
		opts.SetReturnDocument(options.After)
	} else {
		opts.SetReturnDocument(options.Before)
	}
	if upsert {
		opts.SetUpsert(true)
	}
	if sort, ok := spec["sort"]; ok && sort != nil {
		opts.SetSort(toBsonDoc(sort))
	}
	if fields, ok := spec["fields"]; ok && fields != nil {
		opts.SetProjection(toBsonDoc(fields))
	}

	var result bson.M
	err := coll.FindOneAndUpdate(ec.ctx, filter, convertToBson(update), opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("findAndModify: %w", err)
	}
	return result, nil
}
