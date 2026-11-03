package queryengine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// collStatsDoc runs the collStats command and returns the result document.
func collStatsDoc(ec *execContext, collName string, scale int) (bson.D, error) {
	cmd := bson.D{{Key: "collStats", Value: collName}}
	if scale > 0 {
		cmd = append(cmd, bson.E{Key: "scale", Value: scale})
	}
	var result bson.D
	if err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// collStatsField runs collStats and returns a single numeric field from the result.
func collStatsField(ec *execContext, collName, field string) (any, error) {
	stats, err := collStatsDoc(ec, collName, 0)
	if err != nil {
		return nil, err
	}
	return docField(stats, field), nil
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
		result, err := collStatsDoc(ec, collName, scale)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("stats: %w", err)))
		}
		return toJSValue(rt, result)
	})

	_ = obj.Set("isCapped", func() goja.Value {
		requireClient(ec)
		stats, err := collStatsDoc(ec, collName, 0)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("isCapped: %w", err)))
		}
		if v, ok := docField(stats, "capped").(bool); ok {
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
		stats, err := collStatsDoc(ec, collName, 0)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("totalSize: %w", err)))
		}
		total := coerceInt64(docField(stats, "storageSize")) + coerceInt64(docField(stats, "totalIndexSize"))
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
			filter = toBsonDoc(exportValue(call.Arguments[0]))
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
		var result bson.D
		if err := ec.client.Database("admin").RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("renameCollection: %w", err)))
		}
		return toJSValue(rt, result)
	})

	_ = obj.Set("validate", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "validate", Value: collName}}
		if len(call.Arguments) > 0 {
			arg := exportValue(call.Arguments[0])
			if full, ok := arg.(bool); ok {
				cmd = append(cmd, bson.E{Key: "full", Value: full})
			} else if opts, ok := asBsonDoc(arg); ok {
				cmd = append(cmd, opts...)
			}
		}
		var result bson.D
		if err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result); err != nil {
			panic(rt.NewGoError(fmt.Errorf("validate: %w", err)))
		}
		return toJSValue(rt, result)
	})

	_ = obj.Set("findAndModify", func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		op, err := findAndModifyOp(collName, exportDefinedFields(call.Argument(0)))
		if err != nil {
			panic(rt.NewGoError(err))
		}
		result, err := dispatch(ec.ctx, ec.client, ec.dbName, op)
		if err != nil {
			panic(rt.NewGoError(findAndModifyError(op.Method, err)))
		}
		if len(result.Documents) == 0 {
			return goja.Null()
		}
		return toGojaValue(rt, result)
	})
}

// exportDefinedFields exports an argument, leaving out the fields of a plain object
// that are undefined. Exported values can't tell undefined from null, and for
// findAndModify a field set to undefined counts as absent, but one set to null doesn't.
func exportDefinedFields(arg goja.Value) any {
	exported := exportValue(arg)
	doc, ok := exported.(bson.D)
	if !ok {
		return exported
	}
	obj := arg.(*goja.Object)
	return slices.DeleteFunc(doc, func(e bson.E) bool { return goja.IsUndefined(obj.Get(e.Key)) })
}

// findAndModifyError names findAndModify in an error from the method it ran, since
// that is the one the script called. The error's chain is kept.
func findAndModifyError(method string, err error) error {
	rest, ok := strings.CutPrefix(err.Error(), method)
	if !ok {
		return err
	}
	return renamedError{message: "findAndModify" + rest, err: err}
}

type renamedError struct {
	message string
	err     error
}

func (e renamedError) Error() string { return e.message }
func (e renamedError) Unwrap() error { return e.err }

// findAndModifyOp turns a findAndModify spec into the findOneAnd* method mongosh
// would call for it, so that its options behave as they do on that method. Without
// a usable query it must fail, as treating it as {} would change the first document.
func findAndModifyOp(collName string, rawSpec any) (CapturedOp, error) {
	spec, ok := asBsonDoc(rawSpec)
	if !ok {
		return CapturedOp{}, fmt.Errorf("findAndModify requires a spec document")
	}
	query := docField(spec, "query")
	if query == nil {
		return CapturedOp{}, fmt.Errorf("findAndModify requires a query; use query: {} to match every document")
	}
	filter, ok := asBsonDoc(query)
	if !ok {
		return CapturedOp{}, fmt.Errorf("findAndModify query must be a document")
	}
	fields, hasFields := lookupOption(spec, "fields")
	if _, hasProjection := lookupOption(spec, "projection"); hasFields && hasProjection {
		return CapturedOp{}, fmt.Errorf("findAndModify cannot have both fields and projection")
	}

	opts := make(bson.D, 0, len(spec))
	for _, e := range spec {
		switch e.Key {
		case "query", "update", "remove":
		case "fields":
			if jsTruthy(fields) {
				opts = append(opts, bson.E{Key: "projection", Value: fields})
			}
		default:
			opts = append(opts, e)
		}
	}

	if jsTruthy(docField(spec, "remove")) {
		return CapturedOp{Collection: collName, Method: "findOneAndDelete", Args: []any{filter, opts}}, nil
	}
	update := docField(spec, "update")
	if !jsTruthy(update) {
		return CapturedOp{}, fmt.Errorf("findAndModify requires either remove or an update")
	}
	if isUpdateDocument(update) {
		return CapturedOp{Collection: collName, Method: "findOneAndUpdate", Args: []any{filter, update, opts}}, nil
	}
	if _, ok := update.(bson.D); !ok {
		return CapturedOp{}, fmt.Errorf("findAndModify update must be a document or a pipeline")
	}
	return CapturedOp{Collection: collName, Method: "findOneAndReplace", Args: []any{filter, update, opts}}, nil
}

// isUpdateDocument reports whether update is a pipeline or has an update operator,
// rather than being a replacement document. As in mongosh, any key can make it one.
func isUpdateDocument(update any) bool {
	switch u := update.(type) {
	case bson.A:
		return true
	case bson.D:
		return slices.ContainsFunc(u, func(e bson.E) bool { return strings.HasPrefix(e.Key, "$") })
	}
	return false
}
