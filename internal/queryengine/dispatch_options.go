package queryengine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Options behave as they do in mongosh, so a script that runs there runs here,
// against any server version: anything mongosh applies is applied, and anything
// it ignores is ignored. Values the server would refuse go to the server, so it
// is the one to decide what its version supports.

// optionSetters maps each option a method supports to what applies it.
type optionSetters map[string]func(v any) error

// applyOptions applies an options document in the order the script wrote it, so
// the first bad option is the one reported. As in mongosh, an options argument
// that is not a document and options it does not know are ignored, and null and
// undefined options count as absent.
func applyOptions(method string, raw any, setters optionSetters) error {
	if raw == nil {
		return nil
	}
	doc, ok := asBsonDoc(raw)
	if !ok {
		return nil
	}
	for _, e := range doc {
		if e.Value == nil {
			continue
		}
		set, ok := setters[e.Key]
		if !ok {
			continue
		}
		if err := set(e.Value); err != nil {
			return fmt.Errorf("%s %s %w", method, e.Key, err)
		}
	}
	return nil
}

// jsTruthy applies JavaScript truthiness to an exported option value.
func jsTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int32:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0 && !math.IsNaN(x)
	}
	return true
}

// boolOpt reads a flag. As in mongosh, a value that is not a boolean leaves it at
// its default: upsert: 1 does not upsert, and ordered: 0 stays ordered.
func boolOpt[B any](set func(bool) B) func(any) error {
	return func(v any) error {
		if b, ok := v.(bool); ok {
			set(b)
		}
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

// countOpt reads a whole number. As in mongosh, a value that is not a number is
// ignored rather than refused.
func countOpt[B any](set func(int64) B) func(any) error {
	return func(v any) error {
		if !isNumber(v) {
			return nil
		}
		n, err := wholeNumber(v)
		if err != nil {
			return err
		}
		set(n)
		return nil
	}
}

// maxTimeOpt reads maxTimeMS. The driver's options have no such setting: it is
// a deadline on the context (see withMaxTime), as for a find. The driver sends the
// server the time left on that deadline as maxTimeMS, so the server stops the
// operation itself. Unlike mongosh, the script doesn't wait for the server's reply
// once the deadline passes, so a write the server was finishing can land just after
// the script sees the error.
func maxTimeOpt(d *time.Duration) func(any) error {
	return func(v any) error {
		if !isNumber(v) {
			return nil
		}
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

// requiredMaxTimeOpt reads maxTimeMS for the findOneAnd* methods. These are the
// ones where mongosh refuses a value that is not a number rather than ignoring it.
func requiredMaxTimeOpt(d *time.Duration) func(any) error {
	return func(v any) error {
		if !isNumber(v) {
			return errors.New("must be a number")
		}
		return maxTimeOpt(d)(v)
	}
}

func isNumber(v any) bool {
	switch convertToBson(v).(type) {
	case int32, int64, float64:
		return true
	}
	return false
}

// sortOpt reads a sort in any form mongosh takes: a document, a field name, an
// array of field names, a [field, direction] pair, or an array of pairs.
func sortOpt[B any](set func(any) B) func(any) error {
	return func(v any) error {
		sort, err := normalizeSort(v)
		if err != nil {
			return err
		}
		set(sort)
		return nil
	}
}

func normalizeSort(v any) (any, error) {
	switch x := v.(type) {
	case string:
		return bson.D{{Key: x, Value: int32(1)}}, nil
	case bson.A:
		if len(x) == 2 {
			if field, ok := x[0].(string); ok {
				if dir, ok := sortDirection(x[1]); ok {
					return bson.D{{Key: field, Value: dir}}, nil
				}
			}
		}
		sort := make(bson.D, 0, len(x))
		for _, item := range x {
			entry, err := normalizeSort(item)
			if err != nil {
				return nil, err
			}
			sort = append(sort, entry.(bson.D)...)
		}
		return sort, nil
	case bson.D:
		return x, nil
	}
	return nil, errors.New("must be a document, a field name or an array of them")
}

func sortDirection(v any) (any, bool) {
	switch x := v.(type) {
	case string:
		switch x {
		case "asc", "ascending":
			return int32(1), true
		case "desc", "descending":
			return int32(-1), true
		}
	case int32, int64, float64:
		return x, true
	case bson.D:
		return x, true // { $meta: "textScore" }
	}
	return nil, false
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
// collection the operation runs on, so dispatch applies it there.
func takeWriteConcern(op CapturedOp) (CapturedOp, *writeconcern.WriteConcern) {
	i, ok := writeConcernArg[op.Method]
	if !ok {
		return op, nil
	}
	doc, ok := asBsonDoc(argAt(op.Args, i))
	if !ok {
		return op, nil
	}
	rest := make(bson.D, 0, len(doc))
	var wc *writeconcern.WriteConcern
	for _, e := range doc {
		if e.Key != "writeConcern" {
			rest = append(rest, e)
			continue
		}
		wc = parseWriteConcern(e.Value)
	}
	args := slices.Clone(op.Args)
	args[i] = rest
	op.Args = args
	return op, wc
}

// parseWriteConcern reads { w, j, fsync }, ignoring what mongosh ignores. A value
// the server would refuse, such as w: -1, goes to the server. fsync is the old
// spelling of j. wtimeout is accepted but has no effect, because this driver has
// no such setting, so a write waits on replication until the query is cancelled.
func parseWriteConcern(v any) *writeconcern.WriteConcern {
	doc, ok := v.(bson.D)
	if !ok {
		return nil
	}
	wc := &writeconcern.WriteConcern{}
	if w := docField(doc, "w"); w != nil {
		wc.W = writeConcernW(w)
	}
	for _, key := range []string{"fsync", "j"} {
		if flag := docField(doc, key); flag != nil {
			journal := jsTruthy(flag)
			wc.Journal = &journal
		}
	}
	return wc
}

// writeConcernW gives the driver w as an int or a tag name. Its own validation
// only takes those, and a fractional w is whole for the server.
func writeConcernW(w any) any {
	switch x := w.(type) {
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return w
}

// runOptions are the settings of a read that the driver takes from the collection
// or context rather than from the method's options.
type runOptions struct {
	maxTime        time.Duration
	readConcern    *readconcern.ReadConcern
	readPreference *readpref.ReadPref
	// explain is the verbosity of an aggregate's explain option, if it has one.
	explain string
}

func readConcernOpt(rc **readconcern.ReadConcern) func(any) error {
	return func(v any) error {
		level, ok, err := readConcernLevel(v)
		if ok {
			*rc = &readconcern.ReadConcern{Level: level}
		}
		return err
	}
}

func readPreferenceOpt(rp **readpref.ReadPref) func(any) error {
	return func(v any) error {
		name, ok := v.(string)
		if doc, isDoc := v.(bson.D); isDoc {
			name, ok = docField(doc, "mode").(string)
		}
		if !ok {
			return nil
		}
		pref, err := readPreference(name)
		if err != nil {
			return err
		}
		*rp = pref
		return nil
	}
}

func (r runOptions) collection(coll *mongo.Collection) *mongo.Collection {
	if r.readConcern == nil && r.readPreference == nil {
		return coll
	}
	opts := options.Collection()
	if r.readConcern != nil {
		opts.SetReadConcern(r.readConcern)
	}
	if r.readPreference != nil {
		opts.SetReadPreference(r.readPreference)
	}
	return coll.Clone(opts)
}

func (r runOptions) database(client *mongo.Client, name string) *mongo.Database {
	opts := options.Database()
	if r.readConcern != nil {
		opts.SetReadConcern(r.readConcern)
	}
	if r.readPreference != nil {
		opts.SetReadPreference(r.readPreference)
	}
	return client.Database(name, opts)
}

func insertOneOptions(raw any) (*options.InsertOneOptionsBuilder, time.Duration, error) {
	o := options.InsertOne()
	var maxTime time.Duration
	err := applyOptions("insertOne", raw, optionSetters{
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func insertManyOptions(raw any) (*options.InsertManyOptionsBuilder, time.Duration, error) {
	o := options.InsertMany()
	var maxTime time.Duration
	err := applyOptions("insertMany", raw, optionSetters{
		"ordered":                  boolOpt(o.SetOrdered),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func updateOneOptions(raw any) (*options.UpdateOneOptionsBuilder, time.Duration, error) {
	o := options.UpdateOne()
	var maxTime time.Duration
	err := applyOptions("updateOne", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"sort":                     sortOpt(o.SetSort),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func updateManyOptions(raw any) (*options.UpdateManyOptionsBuilder, time.Duration, error) {
	o := options.UpdateMany()
	var maxTime time.Duration
	err := applyOptions("updateMany", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func replaceOptions(raw any) (*options.ReplaceOptionsBuilder, time.Duration, error) {
	o := options.Replace()
	var maxTime time.Duration
	err := applyOptions("replaceOne", raw, optionSetters{
		"upsert":                   boolOpt(o.SetUpsert),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"sort":                     sortOpt(o.SetSort),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func deleteOneOptions(raw any) (*options.DeleteOneOptionsBuilder, time.Duration, error) {
	o := options.DeleteOne()
	var maxTime time.Duration
	err := applyOptions("deleteOne", raw, optionSetters{
		"collation": collationOpt(o.SetCollation),
		"hint":      hintOpt(o.SetHint),
		"let":       anyOpt(o.SetLet),
		"comment":   anyOpt(o.SetComment),
		"maxTimeMS": maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func deleteManyOptions(raw any) (*options.DeleteManyOptionsBuilder, time.Duration, error) {
	o := options.DeleteMany()
	var maxTime time.Duration
	err := applyOptions("deleteMany", raw, optionSetters{
		"collation": collationOpt(o.SetCollation),
		"hint":      hintOpt(o.SetHint),
		"let":       anyOpt(o.SetLet),
		"comment":   anyOpt(o.SetComment),
		"maxTimeMS": maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func bulkWriteOptions(raw any) (*options.BulkWriteOptionsBuilder, time.Duration, error) {
	o := options.BulkWrite()
	var maxTime time.Duration
	err := applyOptions("bulkWrite", raw, optionSetters{
		"ordered":                  boolOpt(o.SetOrdered),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"let":                      anyOpt(o.SetLet),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

func findOneAndDeleteOptions(raw any) (*options.FindOneAndDeleteOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndDelete()
	var maxTime time.Duration
	err := applyOptions("findOneAndDelete", raw, optionSetters{
		"sort":       sortOpt(o.SetSort),
		"projection": anyOpt(o.SetProjection),
		"collation":  collationOpt(o.SetCollation),
		"hint":       hintOpt(o.SetHint),
		"let":        anyOpt(o.SetLet),
		"comment":    anyOpt(o.SetComment),
		"maxTimeMS":  requiredMaxTimeOpt(&maxTime),
	})
	return o, maxTime, err
}

// lookupOption reports an option's value and whether the script gave that option
// at all, since mongosh goes by which options are present, even when null.
func lookupOption(doc bson.D, key string) (any, bool) {
	for _, e := range doc {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

// returnDocumentOption applies the option saying which version of the document to
// return. Of the spellings mongosh takes, whichever comes first here decides,
// wherever it is in the options, and the others are ignored.
func returnDocumentOption[B any](method string, raw any, set func(options.ReturnDocument) B) error {
	doc, ok := asBsonDoc(raw)
	if !ok {
		return nil
	}
	if v, ok := lookupOption(doc, "returnDocument"); ok {
		switch v {
		case "before":
			set(options.Before)
		case "after":
			set(options.After)
		default:
			return fmt.Errorf(`%s returnDocument must be "before" or "after"`, method)
		}
		return nil
	}
	if v, ok := lookupOption(doc, "returnOriginal"); ok {
		if jsTruthy(v) {
			set(options.Before)
		} else {
			set(options.After)
		}
		return nil
	}
	for _, key := range []string{"returnNewDocument", "new"} {
		if v, ok := lookupOption(doc, key); ok {
			if jsTruthy(v) {
				set(options.After)
			} else {
				set(options.Before)
			}
			return nil
		}
	}
	return nil
}

func findOneAndUpdateOptions(raw any) (*options.FindOneAndUpdateOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndUpdate()
	var maxTime time.Duration
	err := applyOptions("findOneAndUpdate", raw, optionSetters{
		"sort":                     sortOpt(o.SetSort),
		"projection":               anyOpt(o.SetProjection),
		"upsert":                   boolOpt(o.SetUpsert),
		"arrayFilters":             arrayFiltersOpt(o.SetArrayFilters),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                requiredMaxTimeOpt(&maxTime),
	})
	if err != nil {
		return o, maxTime, err
	}
	return o, maxTime, returnDocumentOption("findOneAndUpdate", raw, o.SetReturnDocument)
}

func findOneAndReplaceOptions(raw any) (*options.FindOneAndReplaceOptionsBuilder, time.Duration, error) {
	o := options.FindOneAndReplace()
	var maxTime time.Duration
	err := applyOptions("findOneAndReplace", raw, optionSetters{
		"sort":                     sortOpt(o.SetSort),
		"projection":               anyOpt(o.SetProjection),
		"upsert":                   boolOpt(o.SetUpsert),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                requiredMaxTimeOpt(&maxTime),
	})
	if err != nil {
		return o, maxTime, err
	}
	return o, maxTime, returnDocumentOption("findOneAndReplace", raw, o.SetReturnDocument)
}

func countOptions(raw any) (*options.CountOptionsBuilder, runOptions, error) {
	o := options.Count()
	var run runOptions
	err := applyOptions("countDocuments", raw, optionSetters{
		"limit":          countOpt(o.SetLimit),
		"skip":           countOpt(o.SetSkip),
		"collation":      collationOpt(o.SetCollation),
		"hint":           hintOpt(o.SetHint),
		"comment":        anyOpt(o.SetComment),
		"maxTimeMS":      maxTimeOpt(&run.maxTime),
		"readConcern":    readConcernOpt(&run.readConcern),
		"readPreference": readPreferenceOpt(&run.readPreference),
	})
	return o, run, err
}

func aggregateOptions(raw any) (*options.AggregateOptionsBuilder, runOptions, error) {
	o := options.Aggregate()
	var run runOptions
	var err error
	if run.explain, err = explainVerbosity(raw); err != nil {
		return o, run, err
	}
	setters := optionSetters{
		"allowDiskUse":             boolOpt(o.SetAllowDiskUse),
		"batchSize":                batchSizeOpt(o),
		"cursor":                   cursorOpt(o),
		"collation":                collationOpt(o.SetCollation),
		"hint":                     hintOpt(o.SetHint),
		"let":                      anyOpt(o.SetLet),
		"bypassDocumentValidation": boolOpt(o.SetBypassDocumentValidation),
		"comment":                  anyOpt(o.SetComment),
		"maxTimeMS":                maxTimeOpt(&run.maxTime),
		"readConcern":              readConcernOpt(&run.readConcern),
		"readPreference":           readPreferenceOpt(&run.readPreference),
	}
	if run.explain != "" {
		// As with mongosh, the server is the one to judge an explain's read concern,
		// and only newer versions refuse a bad one. See explainAggregate.
		delete(setters, "readConcern")
	}
	return o, run, applyOptions("aggregate", raw, setters)
}

// explainVerbosity reads an aggregate's explain option as mongosh does: true is the
// fullest plan and false the shortest, an empty string means no explain, and any
// other string goes to the server to judge. Anything else is an error.
func explainVerbosity(raw any) (string, error) {
	doc, ok := asBsonDoc(raw)
	if !ok {
		return "", nil
	}
	v, ok := lookupOption(doc, "explain")
	if !ok {
		return "", nil
	}
	switch x := v.(type) {
	case string:
		return x, nil
	case bool:
		if x {
			return "allPlansExecution", nil
		}
		return "queryPlanner", nil
	}
	return "", errors.New("aggregate explain must be a boolean or a verbosity")
}

// aggregateArgs splits aggregate's arguments as mongosh does: (pipeline, options),
// or every argument a stage when the first is not an array.
func aggregateArgs(args []any) (any, any) {
	switch args[0].(type) {
	case []any, bson.A:
		return convertToBson(args[0]), argAt(args, 1)
	}
	return convertToBson(args), nil
}

// explainAggregate runs an aggregate with the explain option, taking the aggregate's
// other options as they would apply to a real run. aggregateOptions has already
// checked them. target is the collection name, or 1 for a database-level pipeline.
func explainAggregate(ctx context.Context, db *mongo.Database, target, pipeline, rawOptions any, run runOptions) (bson.D, error) {
	aggregate := bson.D{
		{Key: "aggregate", Value: target},
		{Key: "pipeline", Value: pipeline},
	}
	cursor := bson.D{}
	doc, _ := asBsonDoc(rawOptions)
	for _, e := range doc {
		if e.Value == nil {
			continue
		}
		switch e.Key {
		case "hint", "collation", "let", "comment", "readConcern":
			aggregate = append(aggregate, e)
		case "allowDiskUse", "bypassDocumentValidation":
			if _, ok := e.Value.(bool); ok {
				aggregate = append(aggregate, e)
			}
		case "batchSize":
			cursor = bson.D{{Key: "batchSize", Value: e.Value}}
		case "cursor":
			if size := docField(e.Value.(bson.D), "batchSize"); size != nil {
				cursor = bson.D{{Key: "batchSize", Value: size}}
			}
		}
	}
	aggregate = append(aggregate, bson.E{Key: "cursor", Value: cursor})
	cmd := bson.D{
		{Key: "explain", Value: aggregate},
		{Key: "verbosity", Value: run.explain},
	}
	opts := options.RunCmd()
	if run.readPreference != nil {
		opts.SetReadPreference(run.readPreference)
	}
	var result bson.D
	err := db.RunCommand(ctx, cmd, opts).Decode(&result)
	return result, err
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
