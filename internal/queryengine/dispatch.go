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
	case "distinct":
		return dispatchDistinct(ctx, coll, op)
	case "findOneAndDelete":
		return dispatchFindOneAndDelete(ctx, coll, op)
	case "findOneAndReplace":
		return dispatchFindOneAndReplace(ctx, coll, op)
	case "findOneAndUpdate":
		return dispatchFindOneAndUpdate(ctx, coll, op)
	case "estimatedDocumentCount":
		return dispatchEstimatedDocumentCount(ctx, coll)
	case "bulkWrite":
		return dispatchBulkWrite(ctx, coll, op)
	case "drop":
		return dispatchDrop(ctx, coll)
	case "createIndex":
		return dispatchCreateIndex(ctx, coll, op)
	case "createIndexes":
		return dispatchCreateIndexes(ctx, coll, op)
	case "dropIndex":
		return dispatchDropIndex(ctx, coll, op)
	case "dropIndexes":
		return dispatchDropIndexes(ctx, coll, op)
	case "listIndexes":
		return dispatchListIndexes(ctx, coll)
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
	if op.Limit > 0 {
		opts.SetLimit(op.Limit)
	}
	if op.Skip > 0 {
		opts.SetSkip(op.Skip)
	}
	if op.Sort != nil {
		opts.SetSort(toBsonDoc(op.Sort))
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

	return singleToResult(map[string]any{
		"values": results,
	}), nil
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
		return models.QueryResult{Documents: []any{}}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndDelete failed: %w", err)
	}

	return docsToResult([]bson.M{result}), nil
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
		return models.QueryResult{Documents: []any{}}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndReplace failed: %w", err)
	}

	return docsToResult([]bson.M{result}), nil
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
		return models.QueryResult{Documents: []any{}}, nil
	}
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("findOneAndUpdate failed: %w", err)
	}

	return docsToResult([]bson.M{result}), nil
}

func dispatchEstimatedDocumentCount(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("estimatedDocumentCount failed: %w", err)
	}

	return singleToResult(map[string]any{
		"count": count,
	}), nil
}

func dispatchBulkWrite(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("bulkWrite requires an operations argument")
	}

	rawOps, ok := op.Args[0].([]any)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("bulkWrite argument must be an array")
	}

	writeModels := make([]mongo.WriteModel, 0, len(rawOps))
	for _, rawOp := range rawOps {
		opMap, ok := rawOp.(map[string]any)
		if !ok {
			return models.QueryResult{}, fmt.Errorf("bulkWrite operation must be an object")
		}
		model, err := toBulkWriteModel(opMap)
		if err != nil {
			return models.QueryResult{}, err
		}
		writeModels = append(writeModels, model)
	}

	res, err := coll.BulkWrite(ctx, writeModels)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("bulkWrite failed: %w", err)
	}

	return singleToResult(map[string]any{
		"acknowledged":  true,
		"insertedCount": res.InsertedCount,
		"matchedCount":  res.MatchedCount,
		"modifiedCount": res.ModifiedCount,
		"deletedCount":  res.DeletedCount,
		"upsertedCount": res.UpsertedCount,
	}), nil
}

func dispatchDrop(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	if err := coll.Drop(ctx); err != nil {
		return models.QueryResult{}, fmt.Errorf("drop failed: %w", err)
	}
	return models.QueryResult{}, nil
}

func dispatchCreateIndex(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("createIndex requires a keys argument")
	}

	keys := toBsonDoc(op.Args[0])
	model := mongo.IndexModel{
		Keys: keys,
	}

	name, err := coll.Indexes().CreateOne(ctx, model)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("createIndex failed: %w", err)
	}

	return singleToResult(map[string]any{
		"name": name,
	}), nil
}

func dispatchCreateIndexes(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("createIndexes requires a keys argument")
	}

	rawKeys, ok := op.Args[0].([]any)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("createIndexes argument must be an array of key objects")
	}

	keys := make([]mongo.IndexModel, 0, len(rawKeys))
	for _, rawKey := range rawKeys {
		key, ok := rawKey.(map[string]any)
		if !ok {
			return models.QueryResult{}, fmt.Errorf("createIndexes key must be an object")
		}
		keys = append(keys, mongo.IndexModel{Keys: toBsonDoc(key)})
	}

	names, err := coll.Indexes().CreateMany(ctx, keys)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("createIndexes failed: %w", err)
	}

	return singleToResult(map[string]any{
		"indexNames": names,
	}), nil
}

func dispatchDropIndex(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("dropIndex requires an index name argument")
	}

	name, ok := op.Args[0].(string)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("dropIndex argument must be a string")
	}

	res, err := coll.Indexes().DropOne(ctx, name)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("dropIndex failed: %w", err)
	}

	var result bson.M
	if err := bson.Unmarshal(res, &result); err != nil {
		return models.QueryResult{}, fmt.Errorf("dropIndex failed to parse response: %w", err)
	}

	return singleToResult(result), nil
}

func dispatchDropIndexes(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		// dropIndexes supports deleting all indexes with no arguments
		res, err := coll.Indexes().DropAll(ctx)
		if err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed: %w", err)
		}

		var result bson.M
		if err := bson.Unmarshal(res, &result); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed to parse response: %w", err)
		}

		return singleToResult(result), nil
	}

	name, ok := op.Args[0].(string)
	if ok {
		res, err := coll.Indexes().DropOne(ctx, name)
		if err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed: %w", err)
		}

		var result bson.M
		if err := bson.Unmarshal(res, &result); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed to parse response: %w", err)
		}

		return singleToResult(result), nil
	}

	rawKeys, ok := op.Args[0].([]any)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("dropIndexes argument must be an array, string or empty")
	}

	keys := make([]string, 0, len(rawKeys))
	for _, rawKey := range rawKeys {
		key, ok := rawKey.(string)
		if !ok {
			return models.QueryResult{}, fmt.Errorf("dropIndexes array elements must be strings")
		}
		keys = append(keys, key)
	}

	var results []bson.M
	for _, key := range keys {
		res, err := coll.Indexes().DropOne(ctx, key)
		if err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed on '%s': %w", key, err)
		}
		var r bson.M
		if err := bson.Unmarshal(res, &r); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed to parse response for '%s': %w", key, err)
		}
		results = append(results, r)
	}

	return docsToResult(results), nil
}

func dispatchListIndexes(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("listIndexes failed: %w", err)
	}
	defer cursor.Close(ctx)
	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return models.QueryResult{}, fmt.Errorf("reading listIndexes cursor: %w", err)
	}
	return docsToResult(results), nil
}

// toBulkWriteModel converts a map[string]any (from goja) to a mongo.WriteModel.
func toBulkWriteModel(opMap map[string]any) (mongo.WriteModel, error) {
	for opType, v := range opMap {
		args, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("bulkWrite operation %s must be an object", opType)
		}
		switch opType {
		case "insertOne":
			return mongo.NewInsertOneModel().
				SetDocument(convertToBson(args["document"])), nil
		case "updateOne":
			return mongo.NewUpdateOneModel().
				SetFilter(toBsonDoc(args["filter"])).
				SetUpdate(convertToBson(args["update"])), nil
		case "updateMany":
			return mongo.NewUpdateManyModel().
				SetFilter(toBsonDoc(args["filter"])).
				SetUpdate(convertToBson(args["update"])), nil
		case "deleteOne":
			return mongo.NewDeleteOneModel().
				SetFilter(toBsonDoc(args["filter"])), nil
		case "deleteMany":
			return mongo.NewDeleteManyModel().
				SetFilter(toBsonDoc(args["filter"])), nil
		case "replaceOne":
			return mongo.NewReplaceOneModel().
				SetFilter(toBsonDoc(args["filter"])).
				SetReplacement(convertToBson(args["replacement"])), nil
		default:
			return nil, fmt.Errorf("unknown bulkWrite operation type: %s", opType)
		}
	}
	return nil, fmt.Errorf("bulkWrite operation must have a type")
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
// Maps become bson.D and slices become bson.A. Maps with a __bsonValue key are
// unwrapped to their original BSON primitive (ObjectID, DateTime, etc.).
func convertToBson(v any) any {
	switch val := v.(type) {
	case map[string]any:
		// Check for wrapped BSON values (from registerBSONTypes)
		if bsonVal, ok := val["__bsonValue"]; ok {
			if w, ok := bsonVal.(*bsonWrapper); ok {
				return w.Value
			}
			return bsonVal
		}
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
// through relaxed Extended JSON. This preserves BSON type information (e.g. $oid,
// $date, $numberLong) so the frontend can display types correctly.
func docsToResult(docs []bson.M) models.QueryResult {
	if len(docs) == 0 {
		return models.QueryResult{Documents: []any{}}
	}

	cleaned := make([]any, 0, len(docs))
	for _, doc := range docs {
		b, err := bson.MarshalExtJSON(doc, false, false)
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
// Uses relaxed Extended JSON to preserve BSON type info (e.g. insertedId as $oid).
func singleToResult(v any) models.QueryResult {
	b, err := bson.MarshalExtJSON(v, false, false)
	if err != nil {
		// Fall back to regular JSON for non-BSON types (e.g. plain maps)
		b, err = json.Marshal(v)
		if err != nil {
			return models.QueryResult{Documents: []any{v}}
		}
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