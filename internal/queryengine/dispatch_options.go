package queryengine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Every option a script passes to a collection method is either applied or
// rejected. Dropping one silently is how `{ upsert: true }` used to write
// nothing and `{ sort: ... }` used to delete the wrong document.

// optionSetters maps each option a method supports to what applies it.
type optionSetters map[string]func(v any) error

// applyOptions applies an options document in the order the script wrote it, so
// the first bad option is the one reported. Null and undefined options count as
// absent. Unlike mongosh, an option it does not know is an error, so a typo such
// as `upsrt` cannot quietly change what a write does.
func applyOptions(method string, raw any, setters optionSetters) error {
	if raw == nil {
		return nil
	}
	doc, ok := asBsonDoc(raw)
	if !ok {
		return fmt.Errorf("%s options must be an object", method)
	}
	for _, e := range doc {
		if e.Value == nil {
			continue
		}
		set, ok := setters[e.Key]
		if !ok {
			return fmt.Errorf("%s option %q is not supported", method, e.Key)
		}
		if err := set(e.Value); err != nil {
			return fmt.Errorf("%s %s %w", method, e.Key, err)
		}
	}
	return nil
}

func boolOpt[B any](set func(bool) B) func(any) error {
	return func(v any) error {
		b, ok := v.(bool)
		if !ok {
			return errors.New("must be a boolean")
		}
		set(b)
		return nil
	}
}

func docOpt[B any](set func(any) B) func(any) error {
	return func(v any) error {
		if _, ok := v.(bson.D); !ok {
			return errors.New("must be an object")
		}
		set(v)
		return nil
	}
}

func anyOpt[B any](set func(any) B) func(any) error {
	return func(v any) error {
		set(v)
		return nil
	}
}

func hintOpt[B any](set func(any) B) func(any) error {
	return func(v any) error {
		switch v.(type) {
		case string, bson.D:
			set(v)
			return nil
		}
		return errors.New("must be an index name or key pattern")
	}
}

func collationOpt[B any](set func(*options.Collation) B) func(any) error {
	return func(v any) error {
		m, ok := asMap(v)
		if !ok {
			return errors.New("must be an object")
		}
		set(toCollation(m))
		return nil
	}
}

func arrayFiltersOpt[B any](set func([]any) B) func(any) error {
	return func(v any) error {
		filters, ok := v.(bson.A)
		if !ok {
			return errors.New("must be an array")
		}
		set(filters)
		return nil
	}
}

func countOpt[B any](set func(int64) B) func(any) error {
	return func(v any) error {
		n, err := wholeNumber(v)
		if err != nil {
			return err
		}
		set(n)
		return nil
	}
}

// maxTimeOpt reads maxTimeMS. The driver's options have no such setting: it is
// a deadline on the context (see withMaxTime), as for a find.
func maxTimeOpt(d *time.Duration) func(any) error {
	return func(v any) error {
		n, err := wholeNumber(v)
		if err != nil {
			return err
		}
		// The server's limit, which also keeps the Duration from overflowing.
		if n > math.MaxInt32 {
			return fmt.Errorf("must be at most %d", math.MaxInt32)
		}
		*d = time.Duration(n) * time.Millisecond
		return nil
	}
}

func withMaxTime(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// wholeNumber reads a non-negative whole number given as a plain number or a
// NumberInt/NumberLong.
func wholeNumber(v any) (int64, error) {
	var n int64
	switch x := convertToBson(v).(type) {
	case int32:
		n = int64(x)
	case int64:
		n = x
	case float64:
		if x != math.Trunc(x) {
			return 0, errors.New("must be a whole number")
		}
		// Converting a float64 beyond int64, or Infinity, gives MinInt64.
		if math.Abs(x) >= math.MaxInt64 {
			return 0, errors.New("is too large")
		}
		n = int64(x)
	default:
		return 0, errors.New("must be a number")
	}
	if n < 0 {
		return 0, errors.New("must not be negative")
	}
	return n, nil
}

// argAt returns the argument at i, or nil when the script did not pass it.
func argAt(args []any, i int) any {
	if i < len(args) {
		return args[i]
	}
	return nil
}

// filterDoc reads the filter of a write. A write always needs one, and it must
// be a document: treating anything else as {} would apply the write to the whole
// collection, so db.c.deleteMany() or a misspelt bulkWrite `filter` would empty
// it. A script that means every document writes {}.
func filterDoc(method string, v any) (bson.D, error) {
	if v == nil {
		return nil, fmt.Errorf("%s requires a filter document", method)
	}
	doc, ok := asBsonDoc(v)
	if !ok {
		return nil, fmt.Errorf("%s filter must be a document", method)
	}
	return doc, nil
}

// replacementDoc reads the replacement of a write, which must be a document.
func replacementDoc(method string, v any) (bson.D, error) {
	doc, ok := asBsonDoc(v)
	if !ok {
		return nil, fmt.Errorf("%s replacement must be a document", method)
	}
	return doc, nil
}

// writeConcernArg gives the position of the options argument of each method that
// takes a writeConcern.
var writeConcernArg = map[string]int{
	"insertOne": 1, "insertMany": 1,
	"updateOne": 2, "updateMany": 2, "replaceOne": 2,
	"deleteOne": 1, "deleteMany": 1,
	"bulkWrite":        1,
	"findOneAndDelete": 1, "findOneAndUpdate": 2, "findOneAndReplace": 2,
}

// takeWriteConcern removes the writeConcern from op's options and returns it. The
// driver's per-operation options have no write concern: it belongs to the
// collection the operation runs on, so dispatch applies it there. Invalid options
// documents are left for the method's own decoder to report.
func takeWriteConcern(op CapturedOp) (CapturedOp, *writeconcern.WriteConcern, error) {
	i, ok := writeConcernArg[op.Method]
	if !ok {
		return op, nil, nil
	}
	doc, ok := asBsonDoc(argAt(op.Args, i))
	if !ok {
		return op, nil, nil
	}
	rest := make(bson.D, 0, len(doc))
	var wc *writeconcern.WriteConcern
	for _, e := range doc {
		if e.Key != "writeConcern" {
			rest = append(rest, e)
			continue
		}
		var err error
		if wc, err = parseWriteConcern(op.Method, e.Value); err != nil {
			return op, nil, err
		}
	}
	args := slices.Clone(op.Args)
	args[i] = rest
	op.Args = args
	return op, wc, nil
}

// parseWriteConcern reads { w, j }. wtimeout is rejected rather than ignored:
// this driver version has no such setting, and a script that sets one expects
// the wait for replication to be bounded.
func parseWriteConcern(method string, v any) (*writeconcern.WriteConcern, error) {
	if v == nil {
		return nil, nil
	}
	doc, ok := v.(bson.D)
	if !ok {
		return nil, fmt.Errorf("%s writeConcern must be an object", method)
	}
	wc := &writeconcern.WriteConcern{}
	for _, e := range doc {
		if e.Value == nil {
			continue
		}
		switch e.Key {
		case "w":
			if name, ok := e.Value.(string); ok {
				wc.W = name
				continue
			}
			n, err := wholeNumber(e.Value)
			if err != nil {
				return nil, fmt.Errorf("%s writeConcern w %w", method, err)
			}
			if n > math.MaxInt32 {
				return nil, fmt.Errorf("%s writeConcern w must be at most %d", method, math.MaxInt32)
			}
			wc.W = int(n)
		case "j":
			j, ok := e.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("%s writeConcern j must be a boolean", method)
			}
			wc.Journal = &j
		default:
			return nil, fmt.Errorf("%s writeConcern option %q is not supported", method, e.Key)
		}
	}
	return wc, nil
}

func insertOneOptions(raw any) (*options.InsertOneOptionsBuilder, error) {
	o := options.InsertOne()
	return o, applyOptions("insertOne", raw, optionSetters{
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
	})
}

func insertManyOptions(raw any) (*options.InsertManyOptionsBuilder, error) {
	o := options.InsertMany()
	return o, applyOptions("insertMany", raw, optionSetters{
		"ordered":                  boolOpt(o.SetOrdered),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
	})
}

func updateOneOptions(raw any) (*options.UpdateOneOptionsBuilder, error) {
	o := options.UpdateOne()
	return o, applyOptions("updateOne", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"sort":                     docOpt(o.SetSort),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
	})
}

func updateManyOptions(raw any) (*options.UpdateManyOptionsBuilder, error) {
	o := options.UpdateMany()
	return o, applyOptions("updateMany", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
	})
}

func replaceOptions(raw any) (*options.ReplaceOptionsBuilder, error) {
	o := options.Replace()
	return o, applyOptions("replaceOne", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"sort":                     docOpt(o.SetSort),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
	})
}

func deleteOneOptions(raw any) (*options.DeleteOneOptionsBuilder, error) {
	o := options.DeleteOne()
	return o, applyOptions("deleteOne", raw, optionSetters{
		"collation": collationOpt(o.SetCollation),
		"hint":      hintOpt(o.SetHint),
		"let":       docOpt(o.SetLet),
		"comment":   anyOpt(o.SetComment),
	})
}

func deleteManyOptions(raw any) (*options.DeleteManyOptionsBuilder, error) {
	o := options.DeleteMany()
	return o, applyOptions("deleteMany", raw, optionSetters{
		"collation": collationOpt(o.SetCollation),
		"hint":      hintOpt(o.SetHint),
		"let":       docOpt(o.SetLet),
		"comment":   anyOpt(o.SetComment),
	})
}

func bulkWriteOptions(raw any) (*options.BulkWriteOptionsBuilder, error) {
	o := options.BulkWrite()
	return o, applyOptions("bulkWrite", raw, optionSetters{
		"ordered":                  boolOpt(o.SetOrdered),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"let":                      docOpt(o.SetLet),
		"comment":                  anyOpt(o.SetComment),
	})
}

func findOneAndDeleteOptions(raw any) (*options.FindOneAndDeleteOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndDelete()
	var maxTime time.Duration
	err := applyOptions("findOneAndDelete", raw, optionSetters{
		"sort":       docOpt(o.SetSort),
		"projection": docOpt(o.SetProjection),
		"collation":  collationOpt(o.SetCollation),
		"hint":       hintOpt(o.SetHint),
		"let":        docOpt(o.SetLet),
		"comment":    anyOpt(o.SetComment),
		"maxTimeMS":  maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

// returnDocumentOpts are the two spellings of "which version to return".
func returnDocumentOpts[B any](set func(options.ReturnDocument) B) optionSetters {
	return optionSetters{
		"returnDocument": func(v any) error {
			switch v {
			case "before":
				set(options.Before)
			case "after":
				set(options.After)
			default:
				return errors.New(`must be "before" or "after"`)
			}
			return nil
		},
		"returnNewDocument": func(v any) error {
			after, ok := v.(bool)
			if !ok {
				return errors.New("must be a boolean")
			}
			if after {
				set(options.After)
			}
			return nil
		},
	}
}

func findOneAndUpdateOptions(raw any) (*options.FindOneAndUpdateOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndUpdate()
	var maxTime time.Duration
	setters := optionSetters{
		"sort":                     docOpt(o.SetSort),
		"projection":               docOpt(o.SetProjection),
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	}
	for k, v := range returnDocumentOpts(o.SetReturnDocument) {
		setters[k] = v
	}
	return o, maxTime, applyOptions("findOneAndUpdate", raw, setters)
}

func findOneAndReplaceOptions(raw any) (*options.FindOneAndReplaceOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndReplace()
	var maxTime time.Duration
	setters := optionSetters{
		"sort":                     docOpt(o.SetSort),
		"projection":               docOpt(o.SetProjection),
		"upsert":                   boolOpt(o.SetUpsert),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	}
	for k, v := range returnDocumentOpts(o.SetReturnDocument) {
		setters[k] = v
	}
	return o, maxTime, applyOptions("findOneAndReplace", raw, setters)
}

func countOptions(raw any) (*options.CountOptionsBuilder, time.Duration, error) {
	o := options.Count()
	var maxTime time.Duration
	err := applyOptions("countDocuments", raw, optionSetters{
		"limit":     countOpt(o.SetLimit),
		"skip":      countOpt(o.SetSkip),
		"collation": collationOpt(o.SetCollation),
		"hint":      hintOpt(o.SetHint),
		"comment":   anyOpt(o.SetComment),
		"maxTimeMS": maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func aggregateOptions(raw any) (*options.AggregateOptionsBuilder, time.Duration, error) {
	o := options.Aggregate()
	var maxTime time.Duration
	err := applyOptions("aggregate", raw, optionSetters{
		"allowDiskUse":             boolOpt(o.SetAllowDiskUse),
		"batchSize":                batchSizeOpt(o),
		"cursor":                   cursorOpt(o),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      docOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func batchSizeOpt(o *options.AggregateOptionsBuilder) func(any) error {
	return func(v any) error {
		n, err := wholeNumber(v)
		if err != nil {
			return err
		}
		if n > math.MaxInt32 {
			return fmt.Errorf("must be at most %d", math.MaxInt32)
		}
		o.SetBatchSize(int32(n))
		return nil
	}
}

// cursorOpt reads the { cursor: { batchSize } } form of batchSize.
func cursorOpt(o *options.AggregateOptionsBuilder) func(any) error {
	return func(v any) error {
		doc, ok := v.(bson.D)
		if !ok {
			return errors.New("must be an object")
		}
		return applyOptions("aggregate cursor", doc, optionSetters{"batchSize": batchSizeOpt(o)})
	}
}
