package queryengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

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
	case bson.A:
		return convertToBson([]any(val))
	case []any:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = convertToBson(item)
		}
		return arr
	case int64:
		return jsIntegerToBson(val)
	default:
		return v
	}
}

// jsIntegerToBson types a whole JS number the way js-bson (and so mongosh)
// does: int32 when it fits, double otherwise. goja exports every whole number
// as int64, which would otherwise store a plain 5 as a long, so a script that
// reads a NumberInt and writes it back would change its type. NumberLong values,
// and server longs too large for a JS number (see jsCommand), never get here:
// they are unwrapped from __bsonValue above.
func jsIntegerToBson(n int64) any {
	if n >= math.MinInt32 && n <= math.MaxInt32 {
		return int32(n)
	}
	return float64(n)
}

// resultDoc is a document read from MongoDB. It stays in driver form (bson.D
// with native BSON values) everywhere in Go, and becomes canonical Extended
// JSON only when Wails marshals the result for the frontend. Canonical mode
// preserves all BSON type information (e.g. $numberInt, $numberLong, $date) so
// the frontend can display types correctly, and bson.D keeps field order.
// Scripts convert it straight to JS values (see jsValue).
type resultDoc bson.D

func (d resultDoc) MarshalJSON() ([]byte, error) {
	b, err := bson.MarshalExtJSON(bson.D(d), true, false)
	if err != nil {
		// Fall back to regular JSON for values Extended JSON cannot encode.
		return json.Marshal(bson.D(d))
	}
	return b, nil
}

// resultValues is an array of values read from MongoDB (e.g. distinct), which
// scripts convert the same way as a resultDoc's fields.
type resultValues bson.A

// docsToResult wraps documents read from MongoDB in a QueryResult.
func docsToResult(docs []bson.D) models.QueryResult {
	out := make([]any, len(docs))
	for i, doc := range docs {
		out[i] = resultDoc(doc)
	}
	return models.QueryResult{Documents: out}
}

// singleToResult wraps a single document as the sole document in a
// QueryResult. The Single flag tells the Goja engine to hand the unwrapped
// object to scripts so `result.insertedIds` works instead of
// `result[0].insertedIds`. It takes a bson.D so the fields keep a fixed order.
func singleToResult(doc bson.D) models.QueryResult {
	return models.QueryResult{Documents: []any{resultDoc(doc)}, Single: true}
}

// updateResultToQueryResult converts a mongo UpdateResult to a QueryResult.
func updateResultToQueryResult(res *mongo.UpdateResult) models.QueryResult {
	result := bson.D{
		{Key: "acknowledged", Value: true},
		{Key: "matchedCount", Value: res.MatchedCount},
		{Key: "modifiedCount", Value: res.ModifiedCount},
	}
	if res.UpsertedID != nil {
		result = append(result, bson.E{Key: "upsertedId", Value: res.UpsertedID})
	}
	return singleToResult(result)
}
