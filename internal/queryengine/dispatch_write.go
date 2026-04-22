package queryengine

import (
	"context"
	"fmt"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func dispatchInsertOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("insertOne requires a document argument")
	}

	doc := convertToBson(op.Args[0])
	res, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("insertOne failed: %w", err)
	}

	result := singleToResult(map[string]any{
		"acknowledged": true,
		"insertedId":   res.InsertedID,
	})
	result.OperationType = "insertOne"
	result.AffectedCount = 1
	return result, nil
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

	result := singleToResult(map[string]any{
		"acknowledged": true,
		"insertedIds":  res.InsertedIDs,
	})
	result.OperationType = "insertMany"
	result.AffectedCount = len(res.InsertedIDs)
	return result, nil
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

	result := updateResultToQueryResult(res)
	result.OperationType = "updateOne"
	result.AffectedCount = int(res.ModifiedCount)
	return result, nil
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

	result := updateResultToQueryResult(res)
	result.OperationType = "updateMany"
	result.AffectedCount = int(res.ModifiedCount)
	return result, nil
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

	result := singleToResult(map[string]any{
		"acknowledged": true,
		"deletedCount": res.DeletedCount,
	})
	result.OperationType = "deleteOne"
	result.AffectedCount = int(res.DeletedCount)
	return result, nil
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

	result := singleToResult(map[string]any{
		"acknowledged": true,
		"deletedCount": res.DeletedCount,
	})
	result.OperationType = "deleteMany"
	result.AffectedCount = int(res.DeletedCount)
	return result, nil
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

	result := updateResultToQueryResult(res)
	result.OperationType = "replaceOne"
	result.AffectedCount = int(res.ModifiedCount)
	return result, nil
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

	result := singleToResult(map[string]any{
		"acknowledged":  true,
		"insertedCount": res.InsertedCount,
		"matchedCount":  res.MatchedCount,
		"modifiedCount": res.ModifiedCount,
		"deletedCount":  res.DeletedCount,
		"upsertedCount": res.UpsertedCount,
	})
	result.OperationType = "bulkWrite"
	result.AffectedCount = int(res.ModifiedCount + res.InsertedCount + res.DeletedCount)
	return result, nil
}

func dispatchDrop(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	if err := coll.Drop(ctx); err != nil {
		return models.QueryResult{}, fmt.Errorf("drop failed: %w", err)
	}
	return models.QueryResult{OperationType: "drop"}, nil
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
