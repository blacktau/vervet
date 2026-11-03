package queryengine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
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
	result.Namespace = findNamespace(coll.Database().Name(), coll.Name(), argAt(op.Args, 1), op.findExtras)
	return result, nil
}

// findDocs runs a find for op and reads every document it returns.
func findDocs(ctx context.Context, coll *mongo.Collection, op CapturedOp) ([]bson.D, error) {
	filter, err := queryFilter(argAt(op.Args, 0))
	if err != nil {
		return nil, err
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

	run, err := op.findExtras.run()
	if err != nil {
		return nil, err
	}
	cursor, err := run.collection(coll).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.D
	if op.Tailable {
		return readTailable(ctx, cursor)
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("reading cursor: %w", err)
	}
	return results, nil
}

// readTailable reads what a tailable cursor has now. It stays open for documents
// yet to come, so reading it to its end would never finish.
func readTailable(ctx context.Context, cursor *mongo.Cursor) ([]bson.D, error) {
	var results []bson.D
	for cursor.TryNext(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("reading cursor: %w", err)
		}
		results = append(results, doc)
	}
	if err := cursor.Err(); err != nil {
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
	if op.Comment != nil && op.Comment != "" {
		opts.SetComment(op.Comment)
	}
	op.findExtras.apply(opts)
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
		filter, err := queryFilter(op.Args[0])
		if err != nil {
			return models.QueryResult{}, err
		}
		findCmd = append(findCmd, bson.E{Key: "filter", Value: filter})
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
	if op.Comment != nil && op.Comment != "" {
		findCmd = append(findCmd, bson.E{Key: "comment", Value: op.Comment})
	}
	findCmd = op.findExtras.explainFields(findCmd)

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
func dispatchFindOne(ctx context.Context, client *mongo.Client, dbName string, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	findOp, err := parseFindArgs(op.Collection, "findOne", op.Args)
	if err != nil {
		return models.QueryResult{}, err
	}
	if findOp.Explain != "" {
		return explainFindOp(ctx, client, dbName, findOp)
	}
	// findOne reads one document, so these change nothing.
	findOp.Limit = 1
	findOp.BatchSize = 0

	docs, err := findDocs(ctx, coll, findOp)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOne failed: %w", err)
	}
	result := docsToResult(docs)
	result.OperationType = "findOne"
	result.Single = len(docs) > 0
	result.Namespace = findNamespace(dbName, op.Collection, argAt(findOp.Args, 1), findOp.findExtras)
	return result, nil
}

// explainFindOp runs find's explain option, at the verbosity it asked for.
func explainFindOp(ctx context.Context, client *mongo.Client, dbName string, op CapturedOp) (models.QueryResult, error) {
	verbosity := op.Explain
	op.Method = "explainFind"
	op.Explain = ""
	op.Args = []any{argAt(op.Args, 0), argAt(op.Args, 1), verbosity}
	return dispatchExplainFind(ctx, client, dbName, op)
}

func dispatchCountDocuments(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	opts, run, err := countOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, run.maxTime)
	defer cancel()

	count, err := run.collection(coll).CountDocuments(ctx, filter, opts)
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

// findNamespace gives the namespace of documents read by a find. They are stored
// documents, but a projection or returnKey leaves out fields, and showRecordId
// adds one that isn't stored. It gives none when the _id shown might not be the
// stored one, since edits and deletes find their document by _id.
func findNamespace(database, collection string, projection any, extras findExtras) *models.ResultNamespace {
	doc, _ := asBsonDoc(projection)
	if !projectionKeepsID(doc) {
		return nil
	}
	return &models.ResultNamespace{
		Database:       database,
		Collection:     collection,
		WholeDocuments: len(doc) == 0 && !extras.ReturnKey && !extras.ShowRecordID,
	}
}

// projectionKeepsID reports whether a projection shows _id as stored or not at
// all. Anything else computes it: { _id: "$customerId" } shows another field's
// value as the _id.
func projectionKeepsID(projection bson.D) bool {
	for _, field := range projection {
		if strings.HasPrefix(field.Key, "_id.") {
			return false
		}
		if field.Key != "_id" {
			continue
		}
		switch field.Value.(type) {
		case bool, int32, int64, float64:
		default:
			return false
		}
	}
	return true
}

// wholeDocumentStages are the aggregation stages that only choose and order
// documents, leaving each one as it is stored.
var wholeDocumentStages = map[string]bool{
	"$match": true,
	"$sort":  true,
	"$limit": true,
	"$skip":  true,
}

// keepsDocumentsWhole reports whether a pipeline's output is stored documents.
// Any other stage might reshape them ($project, $addFields) or make new ones
// ($group), and an edit or delete could then change an unrelated document.
func keepsDocumentsWhole(pipeline any) bool {
	stages, ok := pipeline.(bson.A)
	if !ok {
		return false
	}
	for _, stage := range stages {
		doc, ok := stage.(bson.D)
		if !ok || len(doc) != 1 || !wholeDocumentStages[doc[0].Key] {
			return false
		}
	}
	return true
}

func dispatchAggregate(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("aggregate requires a pipeline argument")
	}

	pipeline, rawOptions := aggregateArgs(op.Args)
	opts, run, err := aggregateOptions(rawOptions)
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, run.maxTime)
	defer cancel()

	if run.explain != "" {
		plan, err := explainAggregate(ctx, coll.Database(), coll.Name(), pipeline, rawOptions, run)
		if err != nil {
			return models.QueryResult{}, fmt.Errorf("aggregate failed: %w", err)
		}
		result := singleToResult(plan)
		result.OperationType = "explain"
		return result, nil
	}

	cursor, err := run.collection(coll).Aggregate(ctx, pipeline, opts)
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
	if keepsDocumentsWhole(pipeline) {
		result.Namespace = &models.ResultNamespace{
			Database:       coll.Database().Name(),
			Collection:     coll.Name(),
			WholeDocuments: true,
		}
	}
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
