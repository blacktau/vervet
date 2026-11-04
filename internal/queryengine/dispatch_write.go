package queryengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func dispatchInsertOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("insertOne requires a document argument")
	}
	opts, maxTime, err := insertOneOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.InsertOne(ctx, insertDocument(op.Args[0]), opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("insertOne failed: %w", err)
	}

	result := singleToResult(bson.D{
		{Key: "acknowledged", Value: res.Acknowledged},
		{Key: "insertedId", Value: res.InsertedID},
	})
	setWriteSummary(&result, "insertOne", 1, res.Acknowledged)
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
	opts, maxTime, err := insertManyOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	docs := make([]any, len(rawDocs))
	for i, d := range rawDocs {
		docs[i] = insertDocument(d)
	}

	res, err := coll.InsertMany(ctx, docs, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("insertMany failed: %w", err)
	}

	result := singleToResult(bson.D{
		{Key: "acknowledged", Value: res.Acknowledged},
		{Key: "insertedIds", Value: res.InsertedIDs},
	})
	setWriteSummary(&result, "insertMany", len(res.InsertedIDs), res.Acknowledged)
	return result, nil
}

func dispatchUpdateOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("updateOne", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("updateOne requires filter and update arguments")
	}
	opts, maxTime, err := updateOneOptions(argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.UpdateOne(ctx, filter, convertToBson(op.Args[1]), opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("updateOne failed: %w", err)
	}
	return updateResultToQueryResult(res, "updateOne"), nil
}

func dispatchUpdateMany(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("updateMany", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	if len(op.Args) < 2 {
		return models.QueryResult{}, fmt.Errorf("updateMany requires filter and update arguments")
	}
	opts, maxTime, err := updateManyOptions(argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.UpdateMany(ctx, filter, convertToBson(op.Args[1]), opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("updateMany failed: %w", err)
	}
	return updateResultToQueryResult(res, "updateMany"), nil
}

func dispatchDeleteOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("deleteOne", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	opts, maxTime, err := deleteOneOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.DeleteOne(ctx, filter, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("deleteOne failed: %w", err)
	}
	return deleteResultToQueryResult(res, "deleteOne"), nil
}

func dispatchDeleteMany(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("deleteMany", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	opts, maxTime, err := deleteManyOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.DeleteMany(ctx, filter, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("deleteMany failed: %w", err)
	}
	return deleteResultToQueryResult(res, "deleteMany"), nil
}

func deleteResultToQueryResult(res *mongo.DeleteResult, method string) models.QueryResult {
	// An unacknowledged write (w: 0) has no count to report.
	doc := bson.D{{Key: "acknowledged", Value: res.Acknowledged}}
	if res.Acknowledged {
		doc = append(doc, bson.E{Key: "deletedCount", Value: int(res.DeletedCount)})
	}
	result := singleToResult(doc)
	setWriteSummary(&result, method, int(res.DeletedCount), res.Acknowledged)
	return result
}

func dispatchReplaceOne(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	filter, err := filterDoc("replaceOne", argAt(op.Args, 0))
	if err != nil {
		return models.QueryResult{}, err
	}
	replacement, err := replacementDoc("replaceOne", argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	opts, maxTime, err := replaceOptions(argAt(op.Args, 2))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	res, err := coll.ReplaceOne(ctx, filter, replacement, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("replaceOne failed: %w", err)
	}
	return updateResultToQueryResult(res, "replaceOne"), nil
}

func dispatchBulkWrite(ctx context.Context, coll *mongo.Collection, op CapturedOp) (models.QueryResult, error) {
	if len(op.Args) < 1 {
		return models.QueryResult{}, fmt.Errorf("bulkWrite requires an operations argument")
	}

	rawOps, ok := op.Args[0].([]any)
	if !ok {
		return models.QueryResult{}, fmt.Errorf("bulkWrite argument must be an array")
	}
	opts, maxTime, err := bulkWriteOptions(argAt(op.Args, 1))
	if err != nil {
		return models.QueryResult{}, err
	}
	ctx, cancel := withMaxTime(ctx, maxTime)
	defer cancel()

	writeModels := make([]mongo.WriteModel, 0, len(rawOps))
	insertedIDs := map[int64]any{}
	for i, rawOp := range rawOps {
		opDoc, ok := asBsonDoc(rawOp)
		if !ok {
			return models.QueryResult{}, fmt.Errorf("bulkWrite operation must be an object")
		}
		model, err := toBulkWriteModel(opDoc)
		if err != nil {
			return models.QueryResult{}, err
		}
		if insert, ok := model.(*mongo.InsertOneModel); ok {
			doc, id := withDocumentID(insert.Document.(bson.D))
			insert.SetDocument(doc)
			insertedIDs[int64(i)] = id
		}
		writeModels = append(writeModels, model)
	}

	res, err := coll.BulkWrite(ctx, writeModels, opts)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("bulkWrite failed: %w", err)
	}

	result := singleToResult(bson.D{
		// Always true, as in mongosh, even with w: 0.
		{Key: "acknowledged", Value: true},
		{Key: "insertedCount", Value: int(res.InsertedCount)},
		{Key: "insertedIds", Value: idsByOperation(insertedIDs)},
		{Key: "matchedCount", Value: int(res.MatchedCount)},
		{Key: "modifiedCount", Value: int(res.ModifiedCount)},
		{Key: "deletedCount", Value: int(res.DeletedCount)},
		{Key: "upsertedCount", Value: int(res.UpsertedCount)},
		{Key: "upsertedIds", Value: idsByOperation(res.UpsertedIDs)},
	})
	affected := int(res.ModifiedCount + res.InsertedCount + res.DeletedCount + res.UpsertedCount)
	setWriteSummary(&result, "bulkWrite", affected, res.Acknowledged)
	return result, nil
}

// insertDocument converts a document to insert, giving it an _id when it has none.
func insertDocument(raw any) any {
	converted := convertToBson(raw)
	doc, ok := converted.(bson.D)
	if !ok {
		return converted
	}
	withID, _ := withDocumentID(doc)
	return withID
}

// withDocumentID returns doc with its _id, generating one first if it has none. As
// in mongosh, an _id of null or undefined counts as none. The driver would add a
// missing one itself, but would store a null, and does not report what it generates
// for a bulk write, so generating it here is the only way to return it in insertedIds.
func withDocumentID(doc bson.D) (bson.D, any) {
	i := slices.IndexFunc(doc, func(e bson.E) bool { return e.Key == "_id" })
	if i >= 0 && doc[i].Value != nil {
		return doc, doc[i].Value
	}
	id := bson.NewObjectID()
	if i < 0 {
		return append(bson.D{{Key: "_id", Value: id}}, doc...), id
	}
	doc = slices.Clone(doc)
	doc[i].Value = id
	return doc, id
}

// idsByOperation gives ids as mongosh does: a document keyed by the index of
// the operation that produced each one, in order.
func idsByOperation(ids map[int64]any) bson.D {
	doc := make(bson.D, 0, len(ids))
	for _, i := range slices.Sorted(maps.Keys(ids)) {
		doc = append(doc, bson.E{Key: strconv.FormatInt(i, 10), Value: ids[i]})
	}
	return doc
}

func dispatchDrop(ctx context.Context, coll *mongo.Collection) (models.QueryResult, error) {
	if err := coll.Drop(ctx); err != nil {
		return models.QueryResult{}, fmt.Errorf("drop failed: %w", err)
	}
	return models.QueryResult{OperationType: "drop"}, nil
}

// bulkOperationTypes lists the operation types in the order mongosh looks for them.
var bulkOperationTypes = []string{"insertOne", "replaceOne", "updateOne", "updateMany", "deleteOne", "deleteMany"}

// toBulkWriteModel converts a bulkWrite operation ({ insertOne: {...} }) to a
// mongo.WriteModel. As in mongosh, the operation type is the first of
// bulkOperationTypes the document has, wherever that key is, even when its value
// is null, and any other keys are ignored. Within the operation, the required
// fields must be there, and options behave as they do on the matching method.
func toBulkWriteModel(opDoc bson.D) (mongo.WriteModel, error) {
	opType := ""
	var value any
	for _, name := range bulkOperationTypes {
		if v, ok := lookupOption(opDoc, name); ok {
			opType, value = name, v
			break
		}
	}
	if opType == "" {
		return nil, fmt.Errorf("bulkWrite operation must be one of %v", bulkOperationTypes)
	}
	args, ok := value.(bson.D) // asBsonDoc converted nested documents too
	if !ok {
		return nil, fmt.Errorf("bulkWrite operation %s must be an object", opType)
	}
	name := "bulkWrite " + opType

	if opType == "insertOne" {
		// As in mongosh, the operation's own fields are the document when it has
		// no document field.
		raw := docField(args, "document")
		if raw == nil {
			return mongo.NewInsertOneModel().SetDocument(args), nil
		}
		document, ok := raw.(bson.D)
		if !ok {
			return nil, fmt.Errorf("%s document must be a document", name)
		}
		return mongo.NewInsertOneModel().SetDocument(document), nil
	}

	filter, err := filterDoc(name, docField(args, "filter"))
	if err != nil {
		return nil, err
	}
	update := docField(args, "update")
	if update == nil && (opType == "updateOne" || opType == "updateMany") {
		return nil, fmt.Errorf("%s requires an update", name)
	}

	switch opType {
	case "updateOne":
		m := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update)
		return m, applyOptions(name, args, optionSetters{
			"upsert":       boolOpt(m.SetUpsert),
			"arrayFilters": arrayFiltersOpt(m.SetArrayFilters),
			"collation":    collationOpt(m.SetCollation),
			"hint":         hintOpt(m.SetHint),
			"sort":         sortOpt(m.SetSort),
		})
	case "updateMany":
		m := mongo.NewUpdateManyModel().SetFilter(filter).SetUpdate(update)
		return m, applyOptions(name, args, optionSetters{
			"upsert":       boolOpt(m.SetUpsert),
			"arrayFilters": arrayFiltersOpt(m.SetArrayFilters),
			"collation":    collationOpt(m.SetCollation),
			"hint":         hintOpt(m.SetHint),
		})
	case "replaceOne":
		replacement, err := replacementDoc(name, docField(args, "replacement"))
		if err != nil {
			return nil, err
		}
		m := mongo.NewReplaceOneModel().SetFilter(filter).SetReplacement(replacement)
		return m, applyOptions(name, args, optionSetters{
			"upsert":    boolOpt(m.SetUpsert),
			"collation": collationOpt(m.SetCollation),
			"hint":      hintOpt(m.SetHint),
			"sort":      sortOpt(m.SetSort),
		})
	case "deleteOne":
		m := mongo.NewDeleteOneModel().SetFilter(filter)
		return m, applyOptions(name, args, optionSetters{
			"collation": collationOpt(m.SetCollation),
			"hint":      hintOpt(m.SetHint),
		})
	}
	// deleteMany is the only type left.
	m := mongo.NewDeleteManyModel().SetFilter(filter)
	return m, applyOptions(name, args, optionSetters{
		"collation": collationOpt(m.SetCollation),
		"hint":      hintOpt(m.SetHint),
	})
}
