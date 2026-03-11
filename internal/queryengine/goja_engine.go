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
// Write methods execute eagerly during script execution. find/findOne return
// lazy cursors that execute on terminal method calls or implicit resolve.
type GojaEngine struct {
	client *mongo.Client
}

func NewGojaEngine(client *mongo.Client) *GojaEngine {
	return &GojaEngine{client: client}
}

func (e *GojaEngine) ExecuteQuery(ctx context.Context, uri, dbName, query string) (models.QueryResult, error) {
	rt := goja.New()
	ec := &execContext{ctx: ctx, client: e.client, dbName: dbName, rt: rt}

	db := newDatabaseProxy(ec)
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

	var result models.QueryResult
	err := func() (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				if gojaErr, ok := r.(*goja.Exception); ok {
					retErr = fmt.Errorf("script error: %s", gojaErr.Value().String())
				} else {
					panic(r)
				}
			}
		}()

		val, err := rt.RunString(query)
		if err != nil {
			return fmt.Errorf("script error: %w", err)
		}

		// Check if return value is an unresolved lazy cursor
		if cursor := extractLazyCursor(val); cursor != nil && !cursor.resolved {
			result, retErr = cursor.execute()
			return
		}

		if len(printed) > 0 {
			result = models.QueryResult{RawOutput: strings.Join(printed, "\n")}
			return
		}

		raw := val.Export()
		if raw != nil {
			result = models.QueryResult{RawOutput: fmt.Sprintf("%v", raw)}
		}
		return
	}()

	return result, err
}
