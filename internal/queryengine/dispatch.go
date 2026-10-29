package queryengine

import (
	"context"
	"encoding/json"
	"fmt"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// dispatch executes a captured operation against MongoDB using the Go driver.
// Handlers for each method live in dispatch_read.go, dispatch_write.go, and
// dispatch_indexes.go; this file keeps the switch plus the conversion helpers
// shared by all of them.
func dispatch(ctx context.Context, client *mongo.Client, dbName string, op CapturedOp) (models.QueryResult, error) {
	coll := client.Database(dbName).Collection(op.Collection)

	switch op.Method {
	case "find":
		return dispatchFind(ctx, coll, op)
	case "explainFind":
		return dispatchExplainFind(ctx, client, dbName, op)
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

// toBsonDoc converts an exported JS object to a bson.D, preserving key order.
// Anything that is not a document yields an empty bson.D.
func toBsonDoc(v any) bson.D {
	doc, _ := asBsonDoc(v)
	return doc
}

// asBsonDoc converts an exported JS object to a bson.D, reporting whether v
// was a document at all. Plain maps are accepted for values that did not come
// through exportValue; their order is whatever Go's map iteration yields.
func asBsonDoc(v any) (bson.D, bool) {
	switch v.(type) {
	case bson.D, bson.M, map[string]any:
	default:
		return bson.D{}, false
	}
	doc, ok := convertToBson(v).(bson.D)
	if !ok {
		// A wrapped BSON value (e.g. a regex), not a document.
		return bson.D{}, false
	}
	return doc, true
}

// convertToBson recursively converts Go values from goja into BSON-compatible types.
// Documents become bson.D (keeping the order of an incoming bson.D) and slices
// become bson.A. Maps with a __bsonValue key are
// unwrapped to their original BSON primitive (ObjectID, DateTime, etc.).
func convertToBson(v any) any {
	switch val := v.(type) {
	case bson.D:
		doc := make(bson.D, len(val))
		for i, e := range val {
			doc[i] = bson.E{Key: e.Key, Value: convertToBson(e.Value)}
		}
		return doc
	case bson.M:
		return convertToBson(map[string]any(val))
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
// through canonical Extended JSON. Canonical mode preserves all BSON type information
// (e.g. $numberInt, $numberLong, $numberDouble, $date, $regularExpression) so the
// frontend can display types correctly.
func docsToResult(docs []bson.M) models.QueryResult {
	if len(docs) == 0 {
		return models.QueryResult{Documents: []any{}}
	}

	cleaned := make([]any, 0, len(docs))
	for _, doc := range docs {
		b, err := bson.MarshalExtJSON(doc, true, false)
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
// Uses canonical Extended JSON to preserve BSON type info (e.g. insertedId as $oid).
// The Single flag tells the Goja engine to hand the unwrapped object to scripts
// so `result.insertedIds` works instead of `result[0].insertedIds`.
func singleToResult(v any) models.QueryResult {
	b, err := bson.MarshalExtJSON(v, true, false)
	if err != nil {
		// Fall back to regular JSON for non-BSON types (e.g. plain maps)
		b, err = json.Marshal(v)
		if err != nil {
			return models.QueryResult{Documents: []any{v}, Single: true}
		}
	}
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return models.QueryResult{Documents: []any{v}, Single: true}
	}
	return models.QueryResult{Documents: []any{m}, Single: true}
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
