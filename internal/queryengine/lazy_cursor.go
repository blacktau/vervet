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
	opType     string
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
//
// paged is true only for the cursor left as the script's final value, which the
// UI renders a page at a time. Script-side terminals (toArray, forEach, ...)
// pass false: mongosh gives them every matching document, and silently
// truncating to a page turns `find(...).toArray().length` into a wrong answer.
func (c *lazyCursor) execute(paged bool) (models.QueryResult, error) {
	if c.resolved {
		return c.cachedResult(paged)
	}

	// Only a paged result needs a PageContext; script-side terminals would
	// pay for encoding it and never use it.
	var pageCtx *models.PageContext
	if paged {
		var err error
		if pageCtx, err = c.buildPageContext(); err != nil {
			return models.QueryResult{}, err
		}
	}

	effSkip := c.skip
	effLimit := c.limit
	if pageCtx != nil && c.ec.pageSize > 0 {
		if c.limit > 0 && c.limit < c.ec.pageSize {
			effLimit = c.limit
		} else {
			effLimit = c.ec.pageSize
		}
	}

	op := CapturedOp{
		Collection: c.collection,
		Method:     "find",
		Args:       []any{c.filter, c.projection},
		Limit:      effLimit,
		Skip:       effSkip,
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
	c.opType = result.OperationType
	result.PageContext = pageCtx
	return result, nil
}

// cachedResult returns an already-executed cursor's remaining documents.
// Script-side reads (paged=false) get everything not yet consumed by next().
// A cursor read during the script ran unpaged; left as the script's final
// value it is shown paged like a fresh cursor: the first page comes from the
// cached documents, and the PageContext skips the ones next() consumed so
// later pages continue from there. A used-up cursor shows an empty result
// with no pager: a PageContext would re-query (UserLimit 0 means unlimited).
func (c *lazyCursor) cachedResult(paged bool) (models.QueryResult, error) {
	remaining := c.results[c.index:]
	result := models.QueryResult{Documents: remaining, OperationType: c.opType}
	if !paged || len(remaining) == 0 {
		return result, nil
	}
	pageCtx, err := c.buildPageContext()
	if err != nil || pageCtx == nil {
		return result, err
	}
	consumed := int64(c.index)
	pageCtx.UserSkip += consumed
	if pageCtx.UserLimit > 0 {
		// remaining is non-empty, so the limit was not used up.
		pageCtx.UserLimit -= consumed
	}
	if c.ec.pageSize > 0 && int64(len(remaining)) > c.ec.pageSize {
		result.Documents = remaining[:c.ec.pageSize]
	}
	result.PageContext = pageCtx
	return result, nil
}

// consumeRest returns the documents next() has not consumed yet and leaves the
// cursor exhausted, as toArray/forEach/map do in mongosh.
func (c *lazyCursor) consumeRest() ([]any, error) {
	if _, err := c.execute(false); err != nil {
		return nil, err
	}
	rest := c.results[c.index:]
	c.index = len(c.results)
	return rest, nil
}

// eachRemaining calls fn for each document next() has not consumed yet,
// consuming one at a time, so if fn fails the documents it never reached can
// still be read (as in mongosh).
func (c *lazyCursor) eachRemaining(fn func(doc any) error) error {
	if _, err := c.execute(false); err != nil {
		return err
	}
	for c.index < len(c.results) {
		doc := c.results[c.index]
		c.index++
		if err := fn(doc); err != nil {
			return err
		}
	}
	return nil
}

// showsDocuments reports whether this cursor, left as the script's final
// value, is shown as its documents rather than giving way to what the script
// printed. It gives way once the script has read from it (next, toArray,
// forEach, map) or when its read found nothing to show. hasNext() alone does
// not count, so `if (c.hasNext()) print("found"); c` still shows documents,
// while on an empty result `while (c.hasNext()) ...; print("done"); c` shows
// the message.
func (c *lazyCursor) showsDocuments(printed bool) bool {
	readFrom := c.resolved && (c.index > 0 || len(c.results) == 0)
	return !printed || !readFrom
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
			if err := c.setSort(exportValue(call.Arguments[0])); err != nil {
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
			c.hint = exportValue(call.Arguments[0])
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
			spec, ok := asMap(exportValue(call.Arguments[0]))
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
		docs, err := c.consumeRest()
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return toJSValue(rt, docs)
	})

	_ = obj.Set("forEach", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(rt.NewGoError(fmt.Errorf("forEach requires a callback function")))
		}
		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panic(rt.NewGoError(fmt.Errorf("forEach argument must be a function")))
		}
		err := c.eachRemaining(func(doc any) error {
			_, err := fn(goja.Undefined(), toJSValue(rt, doc))
			return err
		})
		if err != nil {
			panic(rt.NewGoError(err))
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
		var mapped []any
		err := c.eachRemaining(func(doc any) error {
			val, err := fn(goja.Undefined(), toJSValue(rt, doc))
			mapped = append(mapped, val)
			return err
		})
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return rt.NewArray(mapped...)
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
		return toGojaValue(rt, result)
	})

	_ = obj.Set("hasNext", func() goja.Value {
		if !c.resolved {
			if _, err := c.execute(false); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		return rt.ToValue(c.index < len(c.results))
	})

	_ = obj.Set("next", func() goja.Value {
		if !c.resolved {
			if _, err := c.execute(false); err != nil {
				panic(rt.NewGoError(err))
			}
		}
		if c.index >= len(c.results) {
			panic(rt.NewGoError(fmt.Errorf("cursor exhausted — no more documents")))
		}
		doc := c.results[c.index]
		c.index++
		return toJSValue(rt, doc)
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
