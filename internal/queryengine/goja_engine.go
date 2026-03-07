package queryengine

import (
	"context"
	"fmt"
	"strings"
	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/mongo"
)

// GojaEngine implements QueryEngine using the goja JavaScript runtime.
// It creates proxy objects for db/collection access that capture operations
// as CapturedOp values, then dispatches them to the real MongoDB driver.
type GojaEngine struct {
	client *mongo.Client
}

func NewGojaEngine(client *mongo.Client) *GojaEngine {
	return &GojaEngine{client: client}
}

func (e *GojaEngine) ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error) {
	rt := goja.New()

	db := newDatabaseProxy(rt, dbName)
	if err := rt.Set("db", db); err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to set db global: %w", err)
	}

	var printed []string
	if err := rt.Set("print", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			printed = append(printed, arg.String())
		}
		return goja.Undefined()
	}); err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to set print function: %w", err)
	}

	val, err := rt.RunString(query)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("script error: %w", err)
	}

	op := extractCapturedOp(rt, val)
	if op != nil {
		if e.client == nil {
			return models.QueryResult{}, fmt.Errorf("no MongoDB client available")
		}
		return dispatch(ctx, e.client, dbName, *op)
	}

	if len(printed) > 0 {
		return models.QueryResult{RawOutput: strings.Join(printed, "\n")}, nil
	}

	raw := val.Export()
	if raw != nil {
		return models.QueryResult{RawOutput: fmt.Sprintf("%v", raw)}, nil
	}

	return models.QueryResult{}, nil
}

// extractCapturedOp attempts to retrieve a *CapturedOp from a goja value.
// It checks both direct *CapturedOp values and wrapped objects with __capturedOp.
func extractCapturedOp(rt *goja.Runtime, val goja.Value) *CapturedOp {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}

	exported := val.Export()
	if op, ok := exported.(*CapturedOp); ok {
		return op
	}

	if obj, ok := val.(*goja.Object); ok {
		inner := obj.Get("__capturedOp")
		if inner != nil && !goja.IsUndefined(inner) {
			if op, ok := inner.Export().(*CapturedOp); ok {
				return op
			}
		}
	}

	return nil
}
