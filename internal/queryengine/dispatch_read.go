package queryengine

import (
	"context"
	"fmt"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func dispatchFind(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	opts := options.Find()
	if len(op.Args) > 1 && op.Args[1] != nil {
		if projMap, ok := op.Args[1].(map[string]any); ok {
			opts.SetProjection(toBsonDoc(projMap))
		}
	}
	applyFindOptions(opts, op)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return models.QueryResult{}, fmt.Errorf("reading cursor: %w", err)
	}

	result := docsToResult(results)
	result.OperationType = "find"
	return result, nil
}

// applyFindOptions applies the shared cursor-scoped options from op to the given FindOptions.
func applyFindOptions(opts *options.FindOptions, op CapturedOp) {
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
		} else if hintMap, ok := op.Hint.(map[string]any); ok {
			opts.SetHint(toBsonDoc(hintMap))
		} else {
			opts.SetHint(op.Hint)
		}
	}
	if op.MaxTimeMS > 0 {
		opts.SetMaxTime(time.Duration(op.MaxTimeMS) * time.Millisecond)
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
		switch n := v.(type) {
		case int64:
			c.Strength = int(n)
		case int:
			c.Strength = n
		case float64:
			c.Strength = int(n)
		}
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

// dispatchExplainFind runs an explain command describing the find/findOne query for a lazy cursor.
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
		if projMap, ok := op.Args[1].(map[string]any); ok {
			findCmd = append(findCmd, bson.E{Key: "projection", Value: toBsonDoc(projMap)})
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
		if hintMap, ok := op.Hint.(map[string]any); ok {
			findCmd = append(findCmd, bson.E{Key: "hint", Value: toBsonDoc(hintMap)})
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

	var result bson.M
	if err := client.Database(dbName).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return models.QueryResult{}, fmt.Errorf("explain failed: %w", err)
	}

	qr := singleToResult(result)
	qr.OperationType = "explain"
	return qr, nil
}

func dispatchFindOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	var result bson.M
	err := coll.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return models.QueryResult{Documents: []any{}, OperationType: "findOne"}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOne failed: %w", err)
	}

	result2 := docsToResult([]bson.M{result})
	result2.OperationType = "findOne"
	return result2, nil
}

func dispatchCountDocuments(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("countDocuments failed: %w", err)
	}

	result := singleToResult(map[string]any{
		"count": count,
	})
	result.OperationType = "countDocuments"
	return result, nil
}

func dispatchAggregate(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("aggregate requires a pipeline argument")
	}

	pipeline := convertToBson(op.Args[0])

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("aggregate failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
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

	results, err := coll.Distinct(ctx, field, filter)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("distinct failed: %w", err)
	}

	result := singleToResult(map[string]any{
		"values": results,
	})
	result.OperationType = "distinct"
	return result, nil
}

func dispatchFindOneAndDelete(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("findOneAndDelete requires a filter argument")
	}

	filter := bson.D{}
	if op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	var result bson.M
	err := coll.FindOneAndDelete(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return models.QueryResult{Documents: []any{}, OperationType: "findOneAndDelete"}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndDelete failed: %w", err)
	}

	r := docsToResult([]bson.M{result})
	r.OperationType = "findOneAndDelete"
	return r, nil
}

func dispatchFindOneAndReplace(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("findOneAndReplace requires filter and replacement arguments")
	}

	filter := bson.D{}
	if op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	replacement := bson.D{}
	if op.Args[1] != nil {
		replacement = toBsonDoc(op.Args[1])
	}

	var result bson.M
	err := coll.FindOneAndReplace(ctx, filter, replacement).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return models.QueryResult{Documents: []any{}, OperationType: "findOneAndReplace"}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndReplace failed: %w", err)
	}

	r := docsToResult([]bson.M{result})
	r.OperationType = "findOneAndReplace"
	return r, nil
}

func dispatchFindOneAndUpdate(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("findOneAndUpdate requires filter and update arguments")
	}

	filter := toBsonDoc(op.Args[0])
	update := convertToBson(op.Args[1])

	var result bson.M
	err := coll.FindOneAndUpdate(ctx, filter, update).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return models.QueryResult{Documents: []any{}, OperationType: "findOneAndUpdate"}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndUpdate failed: %w", err)
	}

	r := docsToResult([]bson.M{result})
	r.OperationType = "findOneAndUpdate"
	return r, nil
}

func dispatchEstimatedDocumentCount(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("estimatedDocumentCount failed: %w", err)
	}

	result := singleToResult(map[string]any{
		"count": count,
	})
	result.OperationType = "estimatedDocumentCount"
	return result, nil
}
