package queryengine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
	"vervet/internal/models"
	"vervet/internal/queryengine/jsmodules"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/process"
	"github.com/dop251/goja_nodejs/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// GojaEngine implements QueryEngine using the goja JavaScript runtime.
// Write methods execute eagerly during script execution. find/findOne return
// lazy cursors that execute on terminal method calls or implicit resolve.
type GojaEngine struct {
	client   *mongo.Client
	pageSize int64
	registry *require.Registry
}

func NewGojaEngine(client *mongo.Client, pageSize int64) *GojaEngine {
	registry := require.NewRegistry()
	jsmodules.RegisterAll(registry)
	return &GojaEngine{client: client, pageSize: pageSize, registry: registry}
}

func (e *GojaEngine) ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error) {
	rt := goja.New()
	e.registry.Enable(rt)
	buffer.Enable(rt)
	process.Enable(rt)
	ec := &execContext{ctx: ctx, client: e.client, dbName: dbName, rt: rt, pageSize: e.pageSize}

	if err := registerBSONTypes(rt); err != nil {
		return models.QueryResult{}, err
	}

	if err := registerEJSON(rt); err != nil {
		return models.QueryResult{}, err
	}

	db := newDatabaseProxy(ec)
	if err := rt.Set("db", db); err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to set db global: %w", err)
	}

	out := &scriptOutput{}
	if err := registerOutput(rt, out); err != nil {
		return models.QueryResult{}, err
	}

	var result models.QueryResult
	err := func() (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				if gojaErr, ok := r.(*goja.Exception); ok {
					retErr = scriptError(out, fmt.Errorf("%s", gojaErr.Value().String()))
				} else {
					panic(r)
				}
			}
		}()

		val, err := rt.RunString(rewriteTopLevelDeclarations(query))
		if err != nil {
			return scriptError(out, err)
		}

		// Check if return value is an unresolved lazy cursor
		if cursor := extractLazyCursor(val); cursor != nil && !cursor.resolved {
			result, retErr = cursor.execute(true)
			return
		}

		if len(out.lines) > 0 {
			result = models.QueryResult{RawOutput: out.text()}
			return
		}

		if raw := exportValue(val); raw != nil {
			result = exportedToResult(raw)
		} else if val != nil && !goja.IsUndefined(val) && goja.IsNull(val) {
			result = models.QueryResult{RawOutput: "null"}
		}
		return
	}()

	return result, err
}

// displayValue prepares a value returned by the script for the frontend.
// Script-facing BSON values are objects carrying __bsonValue plus the methods
// that make them usable in JS; handed to the UI as-is, the document would show
// those internals instead of the value. Only those wrappers are rewritten (to
// the same Extended JSON the query paths produce) — plain JS values are left
// exactly as the script built them.
func displayValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		if bv, ok := val["__bsonValue"]; ok {
			if w, ok := bv.(*bsonWrapper); ok {
				return ejsonScalar(w.Value)
			}
			return ejsonScalar(bv)
		}
		out := make(map[string]any, len(val))
		for k, elem := range val {
			out[k] = displayValue(elem)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = displayValue(elem)
		}
		return out
	case time.Time:
		return ejsonScalar(val)
	default:
		return v
	}
}

// ejsonScalar renders a single BSON value as canonical Extended JSON, matching
// how query results are encoded for the frontend.
func ejsonScalar(v any) any {
	data, err := bson.MarshalExtJSON(bson.D{{Key: "v", Value: v}}, true, false)
	if err != nil {
		return fmt.Sprint(v)
	}
	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Sprint(v)
	}
	return wrapper["v"]
}

// scriptError formats a failed script run. Anything the script printed before
// it failed is kept: mongosh writes that output before the error appears, and a
// script that fails half way through its checks is much harder to diagnose
// without it.
func scriptError(out *scriptOutput, err error) error {
	if len(out.lines) == 0 {
		return fmt.Errorf("script error: %w", err)
	}
	return fmt.Errorf("%s\n\nscript error: %w", out.text(), err)
}

// exportedToResult converts a value exported from the Goja runtime back into a
// QueryResult. Structured values (arrays/objects) are preserved as Documents so
// the frontend can render them as a tree/table; scalars fall back to RawOutput.
// The reflect fallback handles named map/slice types (e.g. bson.M) which Go's
// type switch does not match against their unnamed underlying types.
func exportedToResult(raw any) models.QueryResult {
	switch v := raw.(type) {
	case []any:
		docs := make([]any, len(v))
		for i, doc := range v {
			docs[i] = displayValue(doc)
		}
		return models.QueryResult{Documents: docs}
	case map[string]any:
		return models.QueryResult{Documents: []any{displayValue(v)}}
	case string:
		return models.QueryResult{RawOutput: v}
	}

	rv := reflect.ValueOf(raw)
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			m := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				m[iter.Key().String()] = iter.Value().Interface()
			}
			return models.QueryResult{Documents: []any{m}}
		}
	case reflect.Slice, reflect.Array:
		docs := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			docs[i] = rv.Index(i).Interface()
		}
		return models.QueryResult{Documents: docs}
	}

	return models.QueryResult{RawOutput: fmt.Sprintf("%v", raw)}
}
