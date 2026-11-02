package queryengine

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func dispatchFind(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	results, err := findDocs(ctx, coll, op)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("find failed: %w", err)
	}
	result := docsToResult(results)
	result.OperationType = "find"
	return result, nil
}

// findDocs runs a find for op and reads every document it returns.
func findDocs(ctx context.Context, coll *mongo.Collection, op CapturedOp) ([]bson.D, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	// The server's limit, which also keeps the Duration below from overflowing.
	if op.MaxTimeMS > math.MaxInt32 {
		return nil, fmt.Errorf("maxTimeMS must be at most %d", math.MaxInt32)
	}
	if op.MaxTimeMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(op.MaxTimeMS)*time.Millisecond)
		defer cancel()
	}

	opts := options.Find()
	if len(op.Args) > 1 && op.Args[1] != nil {
		if proj, ok := asBsonDoc(op.Args[1]); ok {
			opts.SetProjection(proj)
		}
	}
	applyFindOptions(opts, op)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.D
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("reading cursor: %w", err)
	}
	return results, nil
}

// applyFindOptions applies the shared cursor-scoped options from op to the given FindOptions.
func applyFindOptions(opts *options.FindOptionsBuilder, op CapturedOp) {
	if op.Limit > 0 {
		opts.SetLimit(op.Limit)
	}
	if op.Skip > 0 {
		opts.SetSkip(op.Skip)
	}
	if op.Sort != nil {
		opts.SetSort(toBsonDoc(op.Sort))
	}
	if op.Hint != nil {
		if hintStr, ok := op.Hint.(string); ok {
			opts.SetHint(hintStr)
		} else if hint, ok := asBsonDoc(op.Hint); ok {
			opts.SetHint(hint)
		} else {
			opts.SetHint(op.Hint)
		}
	}
	if op.BatchSize > 0 {
		opts.SetBatchSize(op.BatchSize)
	}
	if op.Collation != nil {
		opts.SetCollation(toCollation(op.Collation))
	}
	if op.Comment != "" {
		opts.SetComment(op.Comment)
	}
}

// toCollation converts a map[string]any into a driver *options.Collation.
func toCollation(m map[string]any) *options.Collation {
	c := &options.Collation{}
	if v, ok := m["locale"].(string); ok {
		c.Locale = v
	}
	if v, ok := m["caseLevel"].(bool); ok {
		c.CaseLevel = v
	}
	if v, ok := m["caseFirst"].(string); ok {
		c.CaseFirst = v
	}
	if v, ok := m["strength"]; ok {
		c.Strength = int(coerceInt64(v))
	}
	if v, ok := m["numericOrdering"].(bool); ok {
		c.NumericOrdering = v
	}
	if v, ok := m["alternate"].(string); ok {
		c.Alternate = v
	}
	if v, ok := m["maxVariable"].(string); ok {
		c.MaxVariable = v
	}
	if v, ok := m["normalization"].(bool); ok {
		c.Normalization = v
	}
	if v, ok := m["backwards"].(bool); ok {
		c.Backwards = v
	}
	return c
}

// dispatchExplainFind runs an explain command describing the find query for a lazy cursor.
func dispatchExplainFind(ctx context.Context, client *mongo.Client, dbName string, op CapturedOp) (models.QueryResult, error) {
	verbosity := "queryPlanner"
	if len(op.Args) > 2 {
		if s, ok := op.Args[2].(string); ok && s != "" {
			verbosity = s
		}
	}

	findCmd := bson.D{{Key: "find", Value: op.Collection}}
	if len(op.Args) > 0 && op.Args[0] != nil {
		findCmd = append(findCmd, bson.E{Key: "filter", Value: toBsonDoc(op.Args[0])})
	}
	if len(op.Args) > 1 && op.Args[1] != nil {
		if proj, ok := asBsonDoc(op.Args[1]); ok {
			findCmd = append(findCmd, bson.E{Key: "projection", Value: proj})
		}
	}
	if op.Sort != nil {
		findCmd = append(findCmd, bson.E{Key: "sort", Value: toBsonDoc(op.Sort)})
	}
	if op.Limit > 0 {
		findCmd = append(findCmd, bson.E{Key: "limit", Value: op.Limit})
	}
	if op.Skip > 0 {
		findCmd = append(findCmd, bson.E{Key: "skip", Value: op.Skip})
	}
	if op.Hint != nil {
		if hint, ok := asBsonDoc(op.Hint); ok {
			findCmd = append(findCmd, bson.E{Key: "hint", Value: hint})
		} else {
			findCmd = append(findCmd, bson.E{Key: "hint", Value: op.Hint})
		}
	}
	if op.BatchSize > 0 {
		findCmd = append(findCmd, bson.E{Key: "batchSize", Value: op.BatchSize})
	}
	if op.MaxTimeMS > 0 {
		findCmd = append(findCmd, bson.E{Key: "maxTimeMS", Value: op.MaxTimeMS})
	}
	if op.Collation != nil {
		findCmd = append(findCmd, bson.E{Key: "collation", Value: toBsonDoc(op.Collation)})
	}
	if op.Comment != "" {
		findCmd = append(findCmd, bson.E{Key: "comment", Value: op.Comment})
	}

	cmd := bson.D{
		{Key: "explain", Value: findCmd},
		{Key: "verbosity", Value: verbosity},
	}

	var result bson.D
	if err := client.Database(dbName).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return models.QueryResult{}, fmt.Errorf("explain failed: %w", err)
	}

	qr := singleToResult(result)
	qr.OperationType = "explain"
	return qr, nil
}

// dispatchFindOne runs findOne(filter, projection, options) as a find limited
// to one document, so it takes the same options a find cursor does.
func dispatchFindOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	findOp := CapturedOp{Collection: op.Collection, Method: "find", Limit: 1}
	if len(op.Args) > 0 {
		findOp.Args = append(findOp.Args, op.Args[0])
	}
	if len(op.Args) > 1 {
		findOp.Args = append(findOp.Args, op.Args[1])
	}
	if len(op.Args) > 2 {
		if err := applyFindOneOptions(&findOp, op.Args[2]); err != nil {
			return models.QueryResult{}, err
		}
	}

	docs, err := findDocs(ctx, coll, findOp)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOne failed: %w", err)
	}
	result := docsToResult(docs)
	result.OperationType = "findOne"
	result.Single = len(docs) > 0
	return result, nil
}

// applyFindOneOptions copies findOne's options document onto a find operation.
// A projection given here applies when the projection argument is absent.
// Options that cannot change the document returned are ignored. Any other
// option it does not support is an error rather than silently dropped, since
// ignoring one such as min or readConcern could change what is returned.
func applyFindOneOptions(op *CapturedOp, raw any) error {
	if raw == nil {
		return nil
	}
	opts, ok := asMap(raw)
	if !ok {
		return fmt.Errorf("findOne options must be an object")
	}
	// Applied in the order the script wrote them, so the first bad option
	// is the one reported. A map has no order, so its keys are sorted.
	var keys []string
	if doc, ok := raw.(bson.D); ok {
		for _, e := range doc {
			keys = append(keys, e.Key)
		}
	} else {
		keys = slices.Sorted(maps.Keys(opts))
	}
	for _, key := range keys {
		if err := applyFindOneOption(op, key, opts[key]); err != nil {
			return err
		}
	}
	return nil
}

// applyFindOneOption applies one findOne option to a find operation.
func applyFindOneOption(op *CapturedOp, key string, v any) error {
	var err error
	switch key {
	case "projection":
		if len(op.Args) < 2 || op.Args[1] == nil {
			for len(op.Args) < 2 {
				op.Args = append(op.Args, nil)
			}
			op.Args[1] = v
		}
	case "sort":
		op.Sort = v
	case "hint":
		op.Hint = v
	case "skip":
		op.Skip, err = countOption(key, v)
	case "maxTimeMS":
		op.MaxTimeMS, err = countOption(key, v)
	case "collation":
		collation, ok := asMap(v)
		if !ok {
			return fmt.Errorf("findOne collation must be an object")
		}
		op.Collation = collation
	case "comment":
		comment, ok := v.(string)
		if !ok {
			return fmt.Errorf("findOne comment must be a string")
		}
		op.Comment = comment
	case "limit", "batchSize", "noCursorTimeout":
		// findOne reads one document, so these change nothing.
	default:
		return fmt.Errorf("findOne option %q is not supported", key)
	}
	return err
}

// countOption reads a findOne option that takes a non-negative whole number.
func countOption(name string, v any) (int64, error) {
	n, err := wholeNumber(v)
	if err != nil {
		return 0, fmt.Errorf("findOne %s %w", name, err)
	}
	return n, nil
}

func dispatchCountDocuments(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	opts, maxTime, err := countOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	count, err := coll.CountDocuments(ctx, filter, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("countDocuments failed: %w", err)
	}

	result := singleToResult(bson.D{
		{Key: "count", Value: count},
	})
	// A Go int, not int64: a count is a plain number, not a stored long.
	result.JSValue = int(count)
	result.OperationType = "countDocuments"
	return result, nil
}

func dispatchAggregate(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("aggregate requires a pipeline argument")
	}

	pipeline := convertToBson(op.Args[0])

	opts, maxTime, err := aggregateOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	cursor, err := coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("aggregate failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.D
	if err := cursor.All(ctx, &results); err != nil {
		return models.QueryResult{}, fmt.Errorf("reading aggregate cursor: %w", err)
	}

	result := docsToResult(results)
	result.OperationType = "aggregate"
	return result, nil
}

func dispatchDistinct(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("distinct requires a field argument")
	}

	field, ok := op.Args[0].(string)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("distinct field must be a string")
	}

	filter := bson.D{}
	if len(op.Args) > 1 && op.Args[1] != nil {
		filter = toBsonDoc(op.Args[1])
	}

	var results []any
	if err := coll.Distinct(ctx, field, filter).Decode(&results); err != nil {
		return models.QueryResult{}, fmt.Errorf("distinct failed: %w", err)
	}

	result := singleToResult(bson.D{
		{Key: "values", Value: results},
	})
	// Scripts get the bare array mongosh returns, converted like document
	// fields (Long, ObjectId, ...) rather than as plain reflected Go values.
	result.JSValue = resultValues(results)
	result.OperationType = "distinct"
	return result, nil
}

// findAndModifyResult is what a findOneAnd* method returns: the document on its
// own, as findOne gives it, or no documents when none matched.
func findAndModifyResult(method string, doc bson.D, err error) (models.QueryResult, error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.QueryResult{Documents: []any{}, OperationType: method}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("%s failed: %w", method, err)
	}
	r := singleToResult(doc)
	r.OperationType = method
	return r, nil
}

func dispatchFindOneAndDelete(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("findOneAndDelete", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	opts, maxTime, err := findOneAndDeleteOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	var doc bson.D
	err = coll.FindOneAndDelete(ctx, filter, opts).Decode(&doc)
	return findAndModifyResult("findOneAndDelete", doc, err)
}

func dispatchFindOneAndReplace(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("findOneAndReplace", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	replacement, err := replacementDoc("findOneAndReplace", argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	opts, maxTime, err := findOneAndReplaceOptions(argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	var doc bson.D
	err = coll.FindOneAndReplace(ctx, filter, replacement, opts).Decode(&doc)
	return findAndModifyResult("findOneAndReplace", doc, err)
}

func dispatchFindOneAndUpdate(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("findOneAndUpdate", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("findOneAndUpdate requires filter and update arguments")
	}
	opts, maxTime, err := findOneAndUpdateOptions(argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	var doc bson.D
	err = coll.FindOneAndUpdate(ctx, filter, convertToBson(op.Args[1]), opts).Decode(&doc)
	return findAndModifyResult("findOneAndUpdate", doc, err)
}

func dispatchEstimatedDocumentCount(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("estimatedDocumentCount failed: %w", err)
	}

	result := singleToResult(bson.D{
		{Key: "count", Value: count},
	})
	result.JSValue = int(count)
	result.OperationType = "estimatedDocumentCount"
	return result, nil
}
