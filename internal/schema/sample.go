package schema

import (
	"context"
	"errors"
	"fmt"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultSampleSize = 1000

// Sample collects up to size docs from the collection, accretes stats,
// and returns a CollectionSchema. Uses $sample; falls back to Find on error.
func Sample(ctx context.Context, client *mongo.Client, dbName, collName string, size int) (models.CollectionSchema, error) {
	if client == nil {
		return models.CollectionSchema{}, errors.New("schema.Sample: nil client")
	}
	if size <= 0 {
		size = defaultSampleSize
	}
	coll := client.Database(dbName).Collection(collName)

	total, _ := coll.EstimatedDocumentCount(ctx)

	schema, err := sampleViaAggregate(ctx, coll, size)
	if err != nil {
		schema, err = sampleViaFind(ctx, coll, size)
		if err != nil {
			return models.CollectionSchema{}, err
		}
	}
	schema.TotalCount = total
	return schema, nil
}

func sampleViaAggregate(ctx context.Context, coll *mongo.Collection, size int) (models.CollectionSchema, error) {
	pipeline := mongo.Pipeline{{{Key: "$sample", Value: bson.D{{Key: "size", Value: size}}}}}
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return models.CollectionSchema{}, fmt.Errorf("$sample failed: %w", err)
	}
	defer cursor.Close(ctx)
	return drainCursor(ctx, cursor)
}

func sampleViaFind(ctx context.Context, coll *mongo.Collection, size int) (models.CollectionSchema, error) {
	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetLimit(int64(size)))
	if err != nil {
		return models.CollectionSchema{}, fmt.Errorf("Find fallback failed: %w", err)
	}
	defer cursor.Close(ctx)
	return drainCursor(ctx, cursor)
}

func drainCursor(ctx context.Context, cursor *mongo.Cursor) (models.CollectionSchema, error) {
	acc := newAccumulator()
	sampled := 0
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		acc.add(doc)
		sampled++
	}
	if err := cursor.Err(); err != nil {
		return models.CollectionSchema{}, err
	}
	return acc.build(sampled), nil
}
