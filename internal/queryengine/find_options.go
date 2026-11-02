package queryengine

import (
	"errors"
	"fmt"
	"math"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// findExtras are the find settings beyond the ones paging has always carried in
// its own fields. They are shared by CapturedOp and lazyCursor, and a PageContext
// keeps them in one encoded field (see encodeFindExtras).
type findExtras struct {
	Min, Max, Let       any
	ReturnKey           bool
	ShowRecordID        bool
	AllowDiskUse        *bool
	AllowPartialResults bool
	NoCursorTimeout     bool
	Tailable            bool
	ReadConcern         string
	ReadPreference      string
}

// run gives the read settings that belong to the collection, not the find.
func (x findExtras) run() (runOptions, error) {
	var run runOptions
	if x.ReadConcern != "" {
		run.readConcern = &readconcern.ReadConcern{Level: x.ReadConcern}
	}
	if x.ReadPreference != "" {
		pref, err := readPreference(x.ReadPreference)
		if err != nil {
			return run, err
		}
		run.readPreference = pref
	}
	return run, nil
}

func (x findExtras) apply(opts *options.FindOptionsBuilder) {
	if x.Min != nil {
		opts.SetMin(x.Min)
	}
	if x.Max != nil {
		opts.SetMax(x.Max)
	}
	if x.Let != nil {
		opts.SetLet(x.Let)
	}
	if x.ReturnKey {
		opts.SetReturnKey(true)
	}
	if x.ShowRecordID {
		opts.SetShowRecordID(true)
	}
	if x.AllowDiskUse != nil {
		opts.SetAllowDiskUse(*x.AllowDiskUse)
	}
	if x.AllowPartialResults {
		opts.SetAllowPartialResults(true)
	}
	if x.NoCursorTimeout {
		opts.SetNoCursorTimeout(true)
	}
	if x.Tailable {
		opts.SetCursorType(options.Tailable)
	}
}

// explainFields adds the settings to a find command being explained.
func (x findExtras) explainFields(cmd bson.D) bson.D {
	add := func(key string, v any) {
		cmd = append(cmd, bson.E{Key: key, Value: v})
	}
	if x.Min != nil {
		add("min", x.Min)
	}
	if x.Max != nil {
		add("max", x.Max)
	}
	if x.Let != nil {
		add("let", x.Let)
	}
	if x.ReturnKey {
		add("returnKey", true)
	}
	if x.ShowRecordID {
		add("showRecordId", true)
	}
	if x.AllowDiskUse != nil {
		add("allowDiskUse", *x.AllowDiskUse)
	}
	return cmd
}

// narrowsMatches reports whether the settings change which documents match, so a
// count cannot be taken from the filter alone.
func (x findExtras) narrowsMatches() bool {
	return x.Min != nil || x.Max != nil || x.Let != nil
}

// queryFilter reads the filter of a find. As in mongosh it is a document or an
// ObjectId, which means that _id; anything else is an error. Absent means every
// document.
func queryFilter(v any) (bson.D, error) {
	if v == nil {
		return bson.D{}, nil
	}
	if doc, ok := asBsonDoc(v); ok {
		return doc, nil
	}
	if id, ok := convertToBson(v).(bson.ObjectID); ok {
		return bson.D{{Key: "_id", Value: id}}, nil
	}
	return nil, errors.New("query filter must be a plain object or ObjectId")
}

// rejectNullFilter throws for a null filter, which mongosh refuses, unlike an
// absent or undefined one. Exported arguments can no longer tell them apart.
func rejectNullFilter(ec *execContext, call goja.FunctionCall) {
	if len(call.Arguments) > 0 && goja.IsNull(call.Arguments[0]) {
		panic(ec.rt.NewGoError(errors.New("expected field filter to be of type object")))
	}
}

// parseFindArgs turns the arguments of find or findOne (filter, projection,
// options) into an operation.
func parseFindArgs(collection, method string, args []any) (CapturedOp, error) {
	op := CapturedOp{Collection: collection, Method: "find"}
	filter, err := queryFilter(argAt(args, 0))
	if err != nil {
		return op, err
	}
	projection := argAt(args, 1)
	if projection != nil {
		if _, ok := asBsonDoc(projection); !ok {
			return op, errors.New("expected field projection to be of type object")
		}
	}
	op.Args = []any{filter, projection}

	raw := argAt(args, 2)
	switch raw.(type) {
	case string, bool, int32, int64, float64:
		return op, fmt.Errorf("%s options must be an object", method)
	}
	return op, applyOptions(method, raw, findOptionSetters(&op))
}

// findOptionSetters applies each find option as mongosh does. Most options that
// have the wrong type are ignored, which mongosh does for them too.
func findOptionSetters(op *CapturedOp) optionSetters {
	flag := func(target *bool) func(any) error {
		return func(v any) error {
			if b, ok := v.(bool); ok {
				*target = b
			}
			return nil
		}
	}
	document := func(name string, target *any) func(any) error {
		return func(v any) error {
			if _, ok := v.(bson.D); !ok {
				return fmt.Errorf("expected field %s to be of type object", name)
			}
			*target = v
			return nil
		}
	}
	return optionSetters{
		"projection": func(v any) error {
			if op.Args[1] != nil {
				return nil
			}
			return document("projection", &op.Args[1])(v)
		},
		"sort": func(v any) error {
			sort, err := normalizeSort(v)
			if err != nil {
				return err
			}
			op.Sort = sort
			return nil
		},
		"hint": func(v any) error {
			hint, err := checkHint(v)
			if err != nil {
				return err
			}
			op.Hint = hint
			return nil
		},
		"limit": func(v any) error {
			if n, ok := jsNumber(v); ok {
				op.Limit = cursorLimit(n)
			}
			return nil
		},
		"skip": func(v any) error {
			n, ok := jsNumber(v)
			if !ok {
				return nil
			}
			skip, err := cursorSkip(n)
			op.Skip = skip
			return err
		},
		"maxTimeMS": func(v any) error {
			n, ok := jsNumber(v)
			if !ok {
				return nil
			}
			ms, err := cursorMaxTime(n)
			op.MaxTimeMS = ms
			return err
		},
		"batchSize": func(v any) error {
			if n, ok := jsNumber(v); ok {
				op.BatchSize = cursorBatchSize(n)
			}
			return nil
		},
		"collation": func(v any) error {
			if m, ok := asMap(v); ok {
				op.Collation = m
			}
			return nil
		},
		"comment": func(v any) error {
			op.Comment = v
			return nil
		},
		"min": document("min", &op.Min),
		"max": document("max", &op.Max),
		"let": func(v any) error {
			if _, ok := v.(bson.D); !ok {
				return errors.New("must be an object")
			}
			op.Let = v
			return nil
		},
		"returnKey":           flag(&op.ReturnKey),
		"showRecordId":        flag(&op.ShowRecordID),
		"allowPartialResults": flag(&op.AllowPartialResults),
		"noCursorTimeout":     flag(&op.NoCursorTimeout),
		"tailable":            flag(&op.Tailable),
		"allowDiskUse": func(v any) error {
			if b, ok := v.(bool); ok {
				op.AllowDiskUse = &b
			}
			return nil
		},
		"readConcern": func(v any) error {
			level, ok, err := readConcernLevel(v)
			if ok {
				op.ReadConcern = level
			}
			return err
		},
		"readPreference": func(v any) error {
			name, ok := v.(string)
			if doc, isDoc := v.(bson.D); isDoc {
				name, ok = docField(doc, "mode").(string)
			}
			if !ok {
				return nil
			}
			if _, err := readPreference(name); err != nil {
				return err
			}
			op.ReadPreference = name
			return nil
		},
		// mongosh explains whenever the option is present, even when it is false.
		"explain": func(v any) error {
			switch x := v.(type) {
			case string:
				op.Explain = x
			case bool:
				op.Explain = "queryPlanner"
				if x {
					op.Explain = "allPlansExecution"
				}
			default:
				op.Explain = "allPlansExecution"
			}
			return nil
		},
	}
}

// jsNumber reads a plain JS number. NumberInt and NumberLong are objects in
// mongosh, so its number checks refuse them, and so does this.
func jsNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, !math.IsNaN(x)
	}
	return 0, false
}

// cursorLimit is a limit as mongosh takes it: truncated, and a negative one means
// the same number of documents.
func cursorLimit(n float64) int64 {
	return int64(math.Abs(math.Trunc(n)))
}

func cursorSkip(n float64) (int64, error) {
	if n < 0 {
		return 0, errors.New("must be >= 0")
	}
	// Converting a float64 beyond int64, or Infinity, gives MinInt64.
	if n >= math.MaxInt64 {
		return 0, errors.New("is too large")
	}
	return int64(math.Trunc(n)), nil
}

func cursorMaxTime(n float64) (int64, error) {
	if n != math.Trunc(n) {
		return 0, errors.New("has a non-integral value")
	}
	if n < 0 || n > math.MaxInt32 {
		return 0, fmt.Errorf("is out of range [0, %d]", math.MaxInt32)
	}
	return int64(n), nil
}

// cursorBatchSize is a batch size as mongosh takes it, where a negative one has
// no effect on what is read.
func cursorBatchSize(n float64) int32 {
	if n <= 0 || n > math.MaxInt32 {
		return 0
	}
	return int32(n)
}

func checkHint(v any) (any, error) {
	switch v.(type) {
	case string, bson.D:
		return v, nil
	}
	return nil, errors.New("hint must be a string or an object")
}

// readConcernLevel reads { level } or a bare level. The second return is false
// when v holds no level to apply.
func readConcernLevel(v any) (string, bool, error) {
	level, ok := v.(string)
	if doc, isDoc := v.(bson.D); isDoc {
		level, ok = docField(doc, "level").(string)
	}
	if !ok {
		return "", false, nil
	}
	switch level {
	case "local", "majority", "linearizable", "available", "snapshot":
		return level, true, nil
	}
	return "", false, errors.New("level must be either 'local', 'majority', 'linearizable', 'available', or 'snapshot'")
}

func readPreference(name string) (*readpref.ReadPref, error) {
	mode, err := readpref.ModeFromString(name)
	if err != nil {
		return nil, fmt.Errorf("invalid read preference mode %q", name)
	}
	return readpref.New(mode)
}

// encodeFindExtras renders the settings for a PageContext, so later pages run the
// same query. A comment that is not a string goes here too: the PageContext's own
// comment field holds only strings. No settings encodes as "".
func encodeFindExtras(x findExtras, comment any) (string, error) {
	var doc bson.D
	add := func(key string, v any) {
		doc = append(doc, bson.E{Key: key, Value: convertToBson(v)})
	}
	if x.Min != nil {
		add("min", x.Min)
	}
	if x.Max != nil {
		add("max", x.Max)
	}
	if x.Let != nil {
		add("let", x.Let)
	}
	if x.ReturnKey {
		add("returnKey", true)
	}
	if x.ShowRecordID {
		add("showRecordId", true)
	}
	if x.AllowDiskUse != nil {
		add("allowDiskUse", *x.AllowDiskUse)
	}
	if x.AllowPartialResults {
		add("allowPartialResults", true)
	}
	if x.NoCursorTimeout {
		add("noCursorTimeout", true)
	}
	if x.Tailable {
		add("tailable", true)
	}
	if x.ReadConcern != "" {
		add("readConcern", x.ReadConcern)
	}
	if x.ReadPreference != "" {
		add("readPreference", x.ReadPreference)
	}
	if _, isString := comment.(string); comment != nil && !isString {
		add("comment", comment)
	}
	if len(doc) == 0 {
		return "", nil
	}
	data, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodeFindExtras is the inverse of encodeFindExtras.
func decodeFindExtras(s string) (findExtras, any, error) {
	var x findExtras
	doc, err := unmarshalPageJSON(s)
	if err != nil || doc == nil {
		return x, nil, err
	}
	x.Min = docField(doc, "min")
	x.Max = docField(doc, "max")
	x.Let = docField(doc, "let")
	x.ReturnKey, _ = docField(doc, "returnKey").(bool)
	x.ShowRecordID, _ = docField(doc, "showRecordId").(bool)
	if allow, ok := docField(doc, "allowDiskUse").(bool); ok {
		x.AllowDiskUse = &allow
	}
	x.AllowPartialResults, _ = docField(doc, "allowPartialResults").(bool)
	x.NoCursorTimeout, _ = docField(doc, "noCursorTimeout").(bool)
	x.Tailable, _ = docField(doc, "tailable").(bool)
	x.ReadConcern, _ = docField(doc, "readConcern").(string)
	x.ReadPreference, _ = docField(doc, "readPreference").(string)
	return x, docField(doc, "comment"), nil
}
