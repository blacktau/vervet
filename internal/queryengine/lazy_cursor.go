package queryengine

import (
	"errors"
	"fmt"

	"vervet/internal/models"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// errStopIteration ends a forEach whose callback returned false.
var errStopIteration = errors.New("stop iteration")

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
	hint       any
	maxTimeMS  int64
	batchSize  int32
	collation  map[string]any
	comment    any
	opType     string
	closed     bool
	findExtras
}

// newLazyCursor makes a cursor for a find whose arguments have been parsed.
func newLazyCursor(ec *execContext, op CapturedOp) *lazyCursor {
	return &lazyCursor{
		ec:         ec,
		collection: op.Collection,
		filter:     argAt(op.Args, 0),
		projection: argAt(op.Args, 1),
		limit:      op.Limit,
		skip:       op.Skip,
		sort:       op.Sort,
		hint:       op.Hint,
		maxTimeMS:  op.MaxTimeMS,
		batchSize:  op.BatchSize,
		collation:  op.Collation,
		comment:    op.Comment,
		findExtras: op.findExtras,
	}
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
	if c.closed {
		c.resolved = true
		c.results = []any{}
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
		findExtras: c.findExtras,
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

// usedUp reports whether the cursor has run and every document has been read.
func (c *lazyCursor) usedUp() bool {
	return c.resolved && c.index >= len(c.results)
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

// explain runs an explain command for this cursor's find query.
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
		findExtras: c.findExtras,
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

	// Chaining methods. Each returns the cursor, and throws once it has run.
	chain := func(name string, apply func(goja.FunctionCall)) {
		_ = obj.Set(name, func(call goja.FunctionCall) goja.Value {
			if c.resolved {
				panic(rt.NewGoError(fmt.Errorf("cursor already executed — cannot set %s", name)))
			}
			apply(call)
			return obj
		})
	}
	throw := func(err error) {
		if err != nil {
			panic(rt.NewGoError(err))
		}
	}
	// integer reads the argument of a method that insists on a plain number.
	integer := func(name string, call goja.FunctionCall) float64 {
		n, ok := jsNumber(call.Argument(0).Export())
		if !ok {
			throw(fmt.Errorf("operation %q requires an integer", name))
		}
		return n
	}
	// document reads an argument that must be a document when it is given.
	document := func(name string, call goja.FunctionCall) any {
		v := exportValue(call.Argument(0))
		if v == nil {
			return nil
		}
		if _, ok := asBsonDoc(v); !ok {
			throw(fmt.Errorf("expected field %s to be of type object", name))
		}
		return convertToBson(v)
	}

	chain("limit", func(call goja.FunctionCall) {
		throw(c.setLimit(cursorLimit(integer("limit", call))))
	})
	chain("skip", func(call goja.FunctionCall) {
		skip, err := cursorSkip(integer("skip", call))
		if err != nil {
			throw(fmt.Errorf("skip %w", err))
		}
		throw(c.setSkip(skip))
	})
	chain("sort", func(call goja.FunctionCall) {
		v := convertToBson(exportValue(call.Argument(0)))
		if v == nil {
			return
		}
		sort, err := normalizeSort(v)
		throw(err)
		throw(c.setSort(sort))
	})
	chain("hint", func(call goja.FunctionCall) {
		v := convertToBson(exportValue(call.Argument(0)))
		if v == nil {
			return
		}
		hint, err := checkHint(v)
		throw(err)
		c.hint = hint
	})
	chain("maxTimeMS", func(call goja.FunctionCall) {
		n, ok := jsNumber(call.Argument(0).Export())
		if !ok {
			throw(errors.New("argument for maxTimeMS must be a number"))
		}
		ms, err := cursorMaxTime(n)
		if err != nil {
			throw(fmt.Errorf("maxTimeMS %w", err))
		}
		c.maxTimeMS = ms
	})
	chain("batchSize", func(call goja.FunctionCall) {
		c.batchSize = cursorBatchSize(integer("batchSize", call))
	})
	chain("collation", func(call goja.FunctionCall) {
		if spec, ok := asMap(exportValue(call.Argument(0))); ok {
			c.collation = spec
		}
	})
	chain("comment", func(call goja.FunctionCall) {
		if v := exportValue(call.Argument(0)); v != nil {
			c.comment = convertToBson(v)
		}
	})
	chain("projection", func(call goja.FunctionCall) {
		if v := document("projection", call); v != nil {
			c.projection = v
		}
	})
	chain("min", func(call goja.FunctionCall) {
		if v := document("min", call); v != nil {
			c.Min = v
		}
	})
	chain("max", func(call goja.FunctionCall) {
		if v := document("max", call); v != nil {
			c.Max = v
		}
	})
	chain("returnKey", func(call goja.FunctionCall) {
		c.ReturnKey = call.Argument(0).StrictEquals(rt.ToValue(true))
	})
	// mongosh turns this on whatever the argument is.
	chain("showRecordId", func(goja.FunctionCall) { c.ShowRecordID = true })
	chain("readConcern", func(call goja.FunctionCall) {
		level, ok := call.Argument(0).Export().(string)
		if !ok {
			throw(errors.New(`"level" had the wrong type: expected a string`))
		}
		parsed, _, err := readConcernLevel(level)
		throw(err)
		c.ReadConcern = parsed
	})
	chain("readPref", func(call goja.FunctionCall) {
		name := call.Argument(0).String()
		_, err := readPreference(name)
		throw(err)
		c.ReadPreference = name
	})
	chain("allowDiskUse", func(call goja.FunctionCall) {
		if c.sort == nil {
			throw(errors.New(`option "allowDiskUse" requires a sort specification`))
		}
		allow := !call.Argument(0).StrictEquals(rt.ToValue(false))
		c.AllowDiskUse = &allow
	})
	chain("allowPartialResults", func(goja.FunctionCall) { c.AllowPartialResults = true })
	chain("noCursorTimeout", func(goja.FunctionCall) { c.NoCursorTimeout = true })
	chain("tailable", func(goja.FunctionCall) { c.Tailable = true })
	chain("addOption", func(call goja.FunctionCall) {
		switch n, _ := jsNumber(call.Argument(0).Export()); n {
		case 2:
			c.Tailable = true
		case 128:
			c.AllowPartialResults = true
		case 8, 64:
			// oplogReplay and exhaust change nothing about what is read here.
		case 32:
			if !c.Tailable {
				throw(errors.New("cannot set 'awaitData' without also setting 'tailable'"))
			}
		case 16:
			throw(errors.New("flag noTimeout is not one of tailable,oplogReplay,noCursorTimeout,awaitData,exhaust,partial"))
		case 4:
			throw(errors.New("the slaveOk option is not supported"))
		default:
			throw(fmt.Errorf("unknown option flag number: %s", call.Argument(0).String()))
		}
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
		// A callback that returns false stops the loop.
		err := c.eachRemaining(func(doc any) error {
			result, err := fn(goja.Undefined(), toJSValue(rt, doc))
			if err == nil && result.StrictEquals(rt.ToValue(false)) {
				return errStopIteration
			}
			return err
		})
		if err != nil && !errors.Is(err, errStopIteration) {
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
		// Already read, so the mapped values carry on as a read cursor.
		return withCursorMethods(rt, "aggregate", rt.NewArray(mapped...))
	})

	// count and size both respect the cursor's skip and limit in mongosh.
	count := func() goja.Value {
		countOpts := bson.D{}
		if c.skip > 0 {
			countOpts = append(countOpts, bson.E{Key: "skip", Value: c.skip})
		}
		if c.limit > 0 {
			countOpts = append(countOpts, bson.E{Key: "limit", Value: c.limit})
		}
		if c.hint != nil {
			countOpts = append(countOpts, bson.E{Key: "hint", Value: c.hint})
		}
		if c.collation != nil {
			countOpts = append(countOpts, bson.E{Key: "collation", Value: c.collation})
		}
		filter := c.filter
		if filter == nil {
			filter = bson.D{}
		}
		op := CapturedOp{
			Collection: c.collection,
			Method:     "countDocuments",
			Args:       []any{filter, countOpts},
		}
		result, err := dispatch(c.ec.ctx, c.ec.client, c.ec.dbName, op)
		if err != nil {
			panic(rt.NewGoError(err))
		}
		return toGojaValue(rt, result)
	}
	_ = obj.Set("count", count)
	_ = obj.Set("size", count)

	run := func() {
		if !c.resolved {
			if _, err := c.execute(false); err != nil {
				panic(rt.NewGoError(err))
			}
		}
	}
	remaining := func() int {
		if !c.resolved {
			return 0
		}
		return len(c.results) - c.index
	}
	next := func() goja.Value {
		run()
		if c.index >= len(c.results) {
			return goja.Null()
		}
		doc := c.results[c.index]
		c.index++
		return toJSValue(rt, doc)
	}

	_ = obj.Set("hasNext", func() goja.Value {
		run()
		return rt.ToValue(c.index < len(c.results))
	})
	_ = obj.Set("next", next)
	_ = obj.Set("tryNext", next)
	_ = obj.Set("itcount", func() goja.Value {
		run()
		n := remaining()
		c.index = len(c.results)
		return rt.ToValue(n)
	})
	_ = obj.Set("objsLeftInBatch", func() goja.Value { return rt.ToValue(remaining()) })
	used := func() goja.Value { return rt.ToValue(c.closed || c.usedUp()) }
	_ = obj.Set("isExhausted", used)
	_ = obj.Set("isClosed", used)
	_ = obj.Set("close", func() goja.Value {
		c.closed = true
		if c.resolved {
			c.index = len(c.results)
		}
		return goja.Undefined()
	})
	_ = obj.SetSymbol(goja.SymIterator, func(goja.FunctionCall) goja.Value {
		iterator := rt.NewObject()
		_ = iterator.Set("next", func() goja.Value {
			run()
			result := rt.NewObject()
			done := c.index >= len(c.results)
			_ = result.Set("done", done)
			if !done {
				_ = result.Set("value", next())
			}
			return result
		})
		return iterator
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
