package queryengine

import (
	"context"
	"fmt"
	"strings"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// indexSpecOptions are the options that describe an index. As in mongosh, any
// option that neither describes the index nor belongs to the command is ignored,
// and the values go to the server, so it decides what its version supports.
var indexSpecOptions = map[string]bool{
	"background": true, "unique": true, "partialFilterExpression": true, "sparse": true,
	"hidden": true, "expireAfterSeconds": true, "storageEngine": true, "collation": true,
	"weights": true, "default_language": true, "language_override": true,
	"textIndexVersion": true, "2dsphereIndexVersion": true, "bits": true, "min": true,
	"max": true, "bucketSize": true, "wildcardProjection": true,
}

func dispatchCreateIndex(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("createIndex requires a keys argument")
	}
	keys, ok := asBsonDoc(op.Args[0])
	if !ok {
		return models.QueryResult{}, fmt.Errorf("createIndex keys must be a document")
	}

	names, err := createIndexes(ctx, coll, "createIndex", []bson.D{keys}, argAt(op.Args, 1), argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}

	result := singleToResult(bson.D{
		{Key: "name", Value: names[0]},
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

	keys := make([]bson.D, 0, len(rawKeys))
	for _, rawKey := range rawKeys {
		key, ok := asBsonDoc(rawKey)
		if !ok {
			return models.QueryResult{}, fmt.Errorf("createIndexes key must be an object")
		}
		keys = append(keys, key)
	}

	names, err := createIndexes(ctx, coll, "createIndexes", keys, argAt(op.Args, 1), argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}

	result := singleToResult(bson.D{
		{Key: "indexNames", Value: names},
	})
	result.OperationType = "createIndexes"
	return result, nil
}

// createIndexes runs the server's createIndexes command, with the same options on
// every index, and returns the index names. commitQuorum is the method's third
// argument, which takes the place of a commitQuorum option.
func createIndexes(ctx context.Context, coll *mongo.Collection, method string, keys []bson.D, rawOptions, commitQuorum any) ([]any, error) {
	opts, err := indexOptions(method, rawOptions)
	if err != nil {
		return nil, err
	}

	specs := make(bson.A, len(keys))
	names := make([]any, len(keys))
	for i, key := range keys {
		spec := indexSpec(key, opts)
		specs[i] = spec
		names[i] = docField(spec, "name")
	}

	cmd := bson.D{{Key: "createIndexes", Value: coll.Name()}, {Key: "indexes", Value: specs}}
	if comment := docField(opts, "comment"); comment != nil {
		cmd = append(cmd, bson.E{Key: "comment", Value: comment})
	}
	if wc := writeConcernCommandField(docField(opts, "writeConcern")); len(wc) > 0 {
		cmd = append(cmd, bson.E{Key: "writeConcern", Value: wc})
	}
	if commitQuorum == nil {
		commitQuorum = docField(opts, "commitQuorum")
	}
	if commitQuorum != nil {
		cmd = append(cmd, bson.E{Key: "commitQuorum", Value: commitQuorum})
	}

	if err := coll.Database().RunCommand(ctx, cmd).Err(); err != nil {
		return nil, fmt.Errorf("%s failed: %w", method, err)
	}
	return names, nil
}

// indexOptions reads the options argument of createIndex and createIndexes. Unlike
// most methods, mongosh refuses options that aren't an object here. null counts as
// absent, and a BSON value such as an ObjectId has no options to apply.
func indexOptions(method string, raw any) (bson.D, error) {
	switch raw.(type) {
	case string, bool, int32, int64, float64, []any, bson.A:
		return nil, fmt.Errorf("%s options must be an object", method)
	}
	doc, _ := asBsonDoc(raw)
	return doc, nil
}

// indexSpec builds one entry of the command's indexes array. An option set to null
// or undefined still goes to the server, as it does from mongosh, except for name,
// which then falls back to the generated name. version is spelt v in the command.
func indexSpec(keys, opts bson.D) bson.D {
	name := docField(opts, "name")
	if name == nil {
		name = indexName(keys)
	}
	spec := bson.D{{Key: "key", Value: keys}, {Key: "name", Value: name}}
	for _, e := range opts {
		switch {
		case e.Key == "version":
			spec = append(spec, bson.E{Key: "v", Value: e.Value})
		case indexSpecOptions[e.Key]:
			spec = append(spec, e)
		}
	}
	return spec
}

// indexName generates the default name of an index: p_1_s_-1 for { p: 1, s: -1 }.
func indexName(keys bson.D) string {
	parts := make([]string, 0, len(keys)*2)
	for _, e := range keys {
		parts = append(parts, e.Key, fmt.Sprint(e.Value))
	}
	return strings.Join(parts, "_")
}

// writeConcernCommandField gives a writeConcern option in the form a command takes,
// without the fields mongosh ignores, which the server could refuse.
func writeConcernCommandField(v any) bson.D {
	wc := parseWriteConcern(v)
	if wc == nil {
		return nil
	}
	doc := bson.D{}
	if wc.W != nil {
		doc = append(doc, bson.E{Key: "w", Value: wc.W})
	}
	if wc.Journal != nil {
		doc = append(doc, bson.E{Key: "j", Value: *wc.Journal})
	}
	if timeout := docField(v.(bson.D), "wtimeout"); timeout != nil {
		doc = append(doc, bson.E{Key: "wtimeout", Value: timeout})
	}
	return doc
}

func dispatchDropIndex(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("dropIndex requires an index name argument")
	}

	name, ok := op.Args[0].(string)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("dropIndex argument must be a string")
	}
	// The server takes * as every index but _id, which only dropIndexes should do.
	if name == "*" {
		return models.QueryResult{}, fmt.Errorf("to drop every index with '*', use dropIndexes()")
	}

	var result bson.D
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
		var result bson.D
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
		var result bson.D
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

	var results []bson.D
	for _, key := range keys {
		var r bson.D
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
	var results []bson.D
	if err := cursor.All(ctx, &results); err != nil {
		return models.QueryResult{}, fmt.Errorf("reading listIndexes cursor: %w", err)
	}
	qr := docsToResult(results)
	qr.OperationType = "listIndexes"
	return qr, nil
}
