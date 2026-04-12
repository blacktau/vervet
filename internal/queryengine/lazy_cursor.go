package queryengine

import (
	"fmt"

	"vervet/internal/models"

	"github.com/dop251/goja"
)

// lazyCursor represents a lazy MongoDB find cursor. It accumulates query
// options (limit, skip, sort) and executes only when a terminal method is
// called or when implicitly resolved at script end.
type lazyCursor struct {
	ec         *execContext
	collection string
	filter     any
	projection any
	limit      int64
	skip       int64
	sort       any
	resolved   bool
	results    []any
	index      int // for hasNext/next iteration
	isFindOne  bool
	hint       any
	maxTimeMS  int64
	batchSize  int32
	collation  map[string]any
	comment    string
}

func (c *lazyCursor) setLimit(n int64) error {
	if c.resolved {
		return fmt.Errorf("cursor already executed — cannot set limit")
	}
	c.limit = n
	return nil
}

func (c *lazyCursor) setSkip(n int64) error {
	if c.resolved {
		return fmt.Errorf("cursor already executed — cannot set skip")
	}
	c.skip = n
	return nil
}

func (c *lazyCursor) setSort(spec any) error {
	if c.resolved {
		return fmt.Errorf("cursor already executed — cannot set sort")
	}
	c.sort = spec
	return nil
}

// execute runs the query against MongoDB and caches the results.
// Subsequent calls return cached results.
func (c *lazyCursor) execute() (models.QueryResult, error) {
	if c.resolved {
		return models.QueryResult{Documents: c.results}, nil
	}

	op := CapturedOp{
		Collection: c.collection,
		Method:     "find",
		Args:       []any{c.filter, c.projection},
		Limit:      c.limit,
		Skip:       c.skip,
		Sort:       c.sort,
		Hint:       c.hint,
		MaxTimeMS:  c.maxTimeMS,
		BatchSize:  c.batchSize,
		Collation:  c.collation,
		Comment:    c.comment,
	}

	if c.isFindOne {
		op.Method = "findOne"
		op.Args = []any{c.filter}
	}

	result, err := dispatch(c.ec.ctx, c.ec.client, c.ec.dbName, op)
	if err != nil {
		return models.QueryResult{}, err
	}

	c.results = result.Documents
	c.resolved = true
	return result, nil
}

// explain runs an explain command for this cursor's find/findOne query.
func (c *lazyCursor) explain(verbosity string) (models.QueryResult, error) {
	op := CapturedOp{
		Collection: c.collection,
		Method:     "explainFind",
		Args:       []any{c.filter, c.projection, verbosity},
		Limit:      c.limit,
		Skip:       c.skip,
		Sort:       c.sort,
		Hint:       c.hint,
		MaxTimeMS:  c.maxTimeMS,
		BatchSize:  c.batchSize,
		Collation:  c.collation,
		Comment:    c.comment,
	}
	if c.isFindOne {
		op.Limit = 1
	}
	return dispatch(c.ec.ctx, c.ec.client, c.ec.dbName, op)
}

// toGojaObject wraps this lazyCursor as a Goja object with chainable and
// terminal methods exposed to JavaScript.
func (c *lazyCursor) toGojaObject() goja.Value {
	rt := c.ec.rt
	obj := rt.NewObject()

	// Hidden property for extractLazyCursor to find this cursor
	_ = obj.Set("__lazyCursor", c)

	// Chaining methods
	_ = obj.Set("limit", func(n int64) goja.Value {
		if err := c.setLimit(n); err != nil {
			panic(rt.NewGoError(err))
		}
		return obj
	})

	_ = obj.Set("skip", func(n int64) goja.Value {
		if err := c.setSkip(n); err != nil {
			panic(rt.NewGoError(err))
		}
		return obj
	})

	_ = obj.Set("sort", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if err := c.setSort(call.Arguments[0].Export()); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		return obj
	})

	_ = obj.Set("hint", func(call goja.FunctionCall) goja.Value {
		if c.resolved {
			panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set hint")))
		}
		if len(call.Arguments) > 0 {
			c.hint = call.Arguments[0].Export()
		}
		return obj
	})

	_ = obj.Set("maxTimeMS", func(n int64) goja.Value {
		if c.resolved {
			panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set maxTimeMS")))
		}
		c.maxTimeMS = n
		return obj
	})

	_ = obj.Set("batchSize", func(n int32) goja.Value {
		if c.resolved {
			panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set batchSize")))
		}
		c.batchSize = n
		return obj
	})

	_ = obj.Set("collation", func(call goja.FunctionCall) goja.Value {
		if c.resolved {
			panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set collation")))
		}
		if len(call.Arguments) > 0 {
			spec, ok := call.Arguments[0].Export().(map[string]any)
			if !ok {
				panic(rt.NewGoError(fmt.Errorf("collation argument must be an object")))
			}
			c.collation = spec
		}
		return obj
	})

	_ = obj.Set("comment", func(s string) goja.Value {
		if c.resolved {
			panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set comment")))
		}
		c.comment = s
		return obj
	})

	// Terminal methods
	_ = obj.Set("toArray", func() goja.Value {
		result, err := c.execute()
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return toGojaValue(rt, result)
	})

	_ = obj.Set("forEach", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("forEach requires a callback function")))
		}
		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panic(rt.NewGoError(fmt.Errorf("forEach argument must be a function")))
		}
		_, err := c.execute()
		if err != nil {
			panic(rt.NewGoError(err))
		}
		for _, doc := range c.results {
			if _, err := fn(goja.Undefined(), rt.ToValue(doc)); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		return goja.Undefined()
	})

	_ = obj.Set("map", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("map requires a callback function")))
		}
		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panic(rt.NewGoError(fmt.Errorf("map argument must be a function")))
		}
		_, err := c.execute()
		if err != nil {
			panic(rt.NewGoError(err))
		}
		mapped := make([]any, len(c.results))
		for i, doc := range c.results {
			val, err := fn(goja.Undefined(), rt.ToValue(doc))
			if err != nil {
				panic(rt.NewGoError(err))
			}
			mapped[i] = val.Export()
		}
		return rt.ToValue(mapped)
	})

	_ = obj.Set("count", func() goja.Value {
		op := CapturedOp{
			Collection: c.collection,
			Method:     "countDocuments",
			Args:       []any{c.filter},
		}
		result, err := dispatch(c.ec.ctx, c.ec.client, c.ec.dbName, op)
		if err != nil {
			panic(rt.NewGoError(err))
		}
		// countDocuments returns {"count": N} as a single document
		if len(result.Documents) > 0 {
			if doc, ok := result.Documents[0].(map[string]any); ok {
				return rt.ToValue(doc["count"])
			}
		}
		return rt.ToValue(0)
	})

	_ = obj.Set("hasNext", func() goja.Value {
		if !c.resolved {
			if _, err := c.execute(); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		return rt.ToValue(c.index < len(c.results))
	})

	_ = obj.Set("next", func() goja.Value {
		if !c.resolved {
			if _, err := c.execute(); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		if c.index >= len(c.results) {
			panic(rt.NewGoError(fmt.Errorf("cursor exhausted — no more documents")))
		}
		doc := c.results[c.index]
		c.index++
		return rt.ToValue(doc)
	})

	_ = obj.Set("explain", func(call goja.FunctionCall) goja.Value {
		verbosity := "queryPlanner"
		if len(call.Arguments) > 0 {
			if s, ok := call.Arguments[0].Export().(string); ok && s != "" {
				verbosity = s
			}
		}
		result, err := c.explain(verbosity)
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return toGojaValue(rt, result)
	})

	// No-op
	_ = obj.Set("pretty", func() goja.Value {
		return obj
	})

	return obj
}
