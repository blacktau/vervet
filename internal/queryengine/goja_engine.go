package queryengine

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
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
// Write methods and findOne execute eagerly during script execution. find
// returns lazy cursors that execute on terminal method calls or implicit resolve.
type GojaEngine struct {
	client   *mongo.Client
	pageSize int64
	// scriptPath is the file the query was loaded from, empty for an unsaved
	// tab. It gives the script __filename/__dirname and fixes the directory
	// that load() and relative fs paths resolve against.
	scriptPath string
}

func NewGojaEngine(client *mongo.Client, pageSize int64, scriptPath string) *GojaEngine {
	return &GojaEngine{client: client, pageSize: pageSize, scriptPath: scriptPath}
}

func (e *GojaEngine) ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error) {
	scriptPath, baseDir := scriptLocation(e.scriptPath)

	rt := goja.New()
	// The registry is built per execution because the modules it registers
	// close over baseDir, which differs from one script to the next.
	registry := require.NewRegistry()
	jsmodules.RegisterAll(registry, baseDir)
	registry.Enable(rt)
	buffer.Enable(rt)
	process.Enable(rt)
	if err := registerScriptEnv(rt, scriptPath, baseDir); err != nil {
		return models.QueryResult{}, err
	}
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

		// A lazy cursor left as the final value is shown as its documents,
		// unless it gives way to what the script printed (see showsDocuments).
		if cursor := extractLazyCursor(val); cursor != nil && cursor.showsDocuments(len(out.lines) > 0) {
			result, retErr = cursor.execute(true)
			return
		}

		// An unchanged findOne document left as the final value is shown even
		// when the script printed, as a cursor it has not read from is.
		var raw any
		if ec.isLastFindOne(val) {
			raw = exportValue(val)
			if found, ok := ec.findOneDisplay(raw); ok {
				result = found
				return
			}
		}

		if len(out.lines) > 0 {
			result = models.QueryResult{RawOutput: out.text()}
			return
		}

		if raw == nil {
			raw = exportValue(val)
		}
		if raw != nil {
			switch raw.(type) {
			case int32, float64:
				// A number shows as JS prints it (5000000001, not 5.000000001e+09).
				result = models.QueryResult{RawOutput: val.String()}
			default:
				result = exportedToResult(raw)
			}
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
// those internals instead of the value. Those wrappers are rewritten to the
// same Extended JSON the query paths produce. A plain number is shown as the
// type it would be stored as (see scriptNumber): the frontend shows an untyped
// whole number as int32, so a whole-number double (5000000000) is sent typed.
// Other plain JS values are left exactly as the script built them.
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
	case bson.D:
		// bson.D encodes to an ordered JSON object, so the frontend shows the
		// fields in the order the script built them.
		out := make(bson.D, len(val))
		for i, e := range val {
			out[i] = bson.E{Key: e.Key, Value: displayValue(e.Value)}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = displayValue(elem)
		}
		return out
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return ejsonScalar(val)
		}
		if val == math.Trunc(val) {
			// Plain digits, where the driver would write 5E+09.
			return bson.D{{Key: "$numberDouble", Value: strconv.FormatFloat(val, 'f', -1, 64)}}
		}
		return val
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
	wrapper, err := decodeOrderedJSON(data)
	if err != nil {
		return fmt.Sprint(v)
	}
	// Ordered so multi-field wrappers keep their form, e.g. {$timestamp: {t, i}}.
	return docField(wrapper.(bson.D), "v")
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
	case bson.D, map[string]any:
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
