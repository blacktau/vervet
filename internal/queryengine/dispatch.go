package queryengine

import (
	"context"
	"encoding/json"
	"fmt"
	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// dispatch executes a captured operation against MongoDB using the Go driver.
func dispatch(ctx context.Context, client *mongo.Client, dbName string, op CapturedOp) (models.QueryResult, error) {
	coll := client.Database(dbName).Collection(op.Collection)

	switch op.Method {
	case "find":
		return dispatchFind(ctx, coll, op)
	case "findOne":
		return dispatchFindOne(ctx, coll, op)
	case "insertOne":
		return dispatchInsertOne(ctx, coll, op)
	case "insertMany":
		return dispatchInsertMany(ctx, coll, op)
	case "updateOne":
		return dispatchUpdateOne(ctx, coll, op)
	case "updateMany":
		return dispatchUpdateMany(ctx, coll, op)
	case "deleteOne":
		return dispatchDeleteOne(ctx, coll, op)
	case "deleteMany":
		return dispatchDeleteMany(ctx, coll, op)
	case "replaceOne":
		return dispatchReplaceOne(ctx, coll, op)
	case "countDocuments":
		return dispatchCountDocuments(ctx, coll, op)
	case "aggregate":
		return dispatchAggregate(ctx, coll, op)
	default:
		return models.QueryResult{}, fmt.Errorf("unsupported operation '%s'. Switch to mongosh engine in settings for full shell compatibility", op.Method)
	}
}

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

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return models.QueryResult{}, fmt.Errorf("reading cursor: %w", err)
	}

	return docsToResult(results), nil
}

func dispatchFindOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	var result bson.M
	err := coll.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return models.QueryResult{Documents: []any{}}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOne failed: %w", err)
	}

	return docsToResult([]bson.M{result}), nil
}

func dispatchInsertOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("insertOne requires a document argument")
	}

	doc := convertToBson(op.Args[0])
	res, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("insertOne failed: %w", err)
	}

	return singleToResult(map[string]any{
		"acknowledged": true,
		"insertedId":   res.InsertedID,
	}), nil
}

func dispatchInsertMany(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("insertMany requires a documents argument")
	}

	rawDocs, ok := op.Args[0].([]any)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("insertMany argument must be an array")
	}

	docs := make([]any, len(rawDocs))
	for i, d := range rawDocs {
		docs[i] = convertToBson(d)
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("insertMany failed: %w", err)
	}

	return singleToResult(map[string]any{
		"acknowledged": true,
		"insertedIds":  res.InsertedIDs,
	}), nil
}

func dispatchUpdateOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("updateOne requires filter and update arguments")
	}

	filter := toBsonDoc(op.Args[0])
	update := convertToBson(op.Args[1])

	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("updateOne failed: %w", err)
	}

	return updateResultToQueryResult(res), nil
}

func dispatchUpdateMany(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("updateMany requires filter and update arguments")
	}

	filter := toBsonDoc(op.Args[0])
	update := convertToBson(op.Args[1])

	res, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("updateMany failed: %w", err)
	}

	return updateResultToQueryResult(res), nil
}

func dispatchDeleteOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	res, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("deleteOne failed: %w", err)
	}

	return singleToResult(map[string]any{
		"acknowledged": true,
		"deletedCount": res.DeletedCount,
	}), nil
}

func dispatchDeleteMany(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter := bson.D{}
	if len(op.Args) > 0 && op.Args[0] != nil {
		filter = toBsonDoc(op.Args[0])
	}

	res, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("deleteMany failed: %w", err)
	}

	return singleToResult(map[string]any{
		"acknowledged": true,
		"deletedCount": res.DeletedCount,
	}), nil
}

func dispatchReplaceOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("replaceOne requires filter and replacement arguments")
	}

	filter := toBsonDoc(op.Args[0])
	replacement := convertToBson(op.Args[1])

	res, err := coll.ReplaceOne(ctx, filter, replacement)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("replaceOne failed: %w", err)
	}

	return updateResultToQueryResult(res), nil
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

	return singleToResult(map[string]any{
		"count": count,
	}), nil
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

	return docsToResult(results), nil
}

// toBsonDoc converts a map[string]any (from goja) to a bson.D, preserving key order.
func toBsonDoc(v any) bson.D {
	m, ok := v.(map[string]any)
	if !ok {
		return bson.D{}
	}

	doc := make(bson.D, 0, len(m))
	for k, val := range m {
		doc = append(doc, bson.E{Key: k, Value: convertToBson(val)})
	}
	return doc
}

// convertToBson recursively converts Go values from goja into BSON-compatible types.
// Maps become bson.D and slices become bson.A.
func convertToBson(v any) any {
	switch val := v.(type) {
	case map[string]any:
		doc := make(bson.D, 0, len(val))
		for k, item := range val {
			doc = append(doc, bson.E{Key: k, Value: convertToBson(item)})
		}
		return doc
	case []any:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = convertToBson(item)
		}
		return arr
	default:
		return v
	}
}

// docsToResult converts bson.M documents into a clean QueryResult by round-tripping
// through JSON to eliminate BSON-specific types (e.g. primitive.ObjectID).
func docsToResult(docs []bson.M) models.QueryResult {
	if len(docs) == 0 {
		return models.QueryResult{Documents: []any{}}
	}

	cleaned := make([]any, 0, len(docs))
	for _, doc := range docs {
		b, err := json.Marshal(doc)
		if err != nil {
			cleaned = append(cleaned, doc)
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			cleaned = append(cleaned, doc)
			continue
		}
		cleaned = append(cleaned, m)
	}

	return models.QueryResult{Documents: cleaned}
}

// singleToResult wraps a single value as the sole document in a QueryResult.
func singleToResult(v any) models.QueryResult {
	b, err := json.Marshal(v)
	if err != nil {
		return models.QueryResult{Documents: []any{v}}
	}
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return models.QueryResult{Documents: []any{v}}
	}
	return models.QueryResult{Documents: []any{m}}
}

// updateResultToQueryResult converts a mongo UpdateResult to a QueryResult.
func updateResultToQueryResult(res *mongo.UpdateResult) models.QueryResult {
	result := map[string]any{
		"acknowledged":  true,
		"matchedCount":  res.MatchedCount,
		"modifiedCount": res.ModifiedCount,
	}
	if res.UpsertedID != nil {
		result["upsertedId"] = res.UpsertedID
	}
	return singleToResult(result)
}
