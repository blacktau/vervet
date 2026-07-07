package queryengine

import (
	"context"
	"fmt"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

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

	result := singleToResult(map[string]any{
		"name": name,
	})
	result.OperationType = "createIndex"
	return result, nil
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

	result := singleToResult(map[string]any{
		"indexNames": names,
	})
	result.OperationType = "createIndexes"
	return result, nil
}

func dispatchDropIndex(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("dropIndex requires an index name argument")
	}

	name, ok := op.Args[0].(string)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("dropIndex argument must be a string")
	}

	var result bson.M
	cmd := bson.D{{Key: "dropIndexes", Value: coll.Name()}, {Key: "index", Value: name}}
	if err := coll.Database().RunCommand(ctx, cmd).Decode(&result); err != nil {
		return models.QueryResult{}, fmt.Errorf("dropIndex failed: %w", err)
	}

	qr := singleToResult(result)
	qr.OperationType = "dropIndex"
	return qr, nil
}

func dispatchDropIndexes(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		// dropIndexes supports deleting all indexes with no arguments
		var result bson.M
		cmd := bson.D{{Key: "dropIndexes", Value: coll.Name()}, {Key: "index", Value: "*"}}
		if err := coll.Database().RunCommand(ctx, cmd).Decode(&result); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed: %w", err)
		}

		qr := singleToResult(result)
		qr.OperationType = "dropIndexes"
		return qr, nil
	}

	name, ok := op.Args[0].(string)
	if ok {
		var result bson.M
		cmd := bson.D{{Key: "dropIndexes", Value: coll.Name()}, {Key: "index", Value: name}}
		if err := coll.Database().RunCommand(ctx, cmd).Decode(&result); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed: %w", err)
		}

		qr := singleToResult(result)
		qr.OperationType = "dropIndexes"
		return qr, nil
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
		var r bson.M
		cmd := bson.D{{Key: "dropIndexes", Value: coll.Name()}, {Key: "index", Value: key}}
		if err := coll.Database().RunCommand(ctx, cmd).Decode(&r); err != nil {
			return models.QueryResult{}, fmt.Errorf("dropIndexes failed on '%s': %w", key, err)
		}
		results = append(results, r)
	}

	qr := docsToResult(results)
	qr.OperationType = "dropIndexes"
	return qr, nil
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
	qr := docsToResult(results)
	qr.OperationType = "listIndexes"
	return qr, nil
}
