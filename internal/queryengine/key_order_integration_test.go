//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Key order is significant to MongoDB in more places than command names. These
// tests guard against script objects losing their key order on the way to the
// driver (issue #301). Each repeats the operation because a randomly ordered
// map can produce the right order by chance.

func keyOrderEngine(t *testing.T) (context.Context, string, *GojaEngine) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db := dbName(t)
	t.Cleanup(func() { _ = testClient.Database(db).Drop(context.Background()) })
	return ctx, db, NewGojaEngine(testClient, 0, "")
}

func TestIntegration_KeyOrder_CompoundIndexKeys(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	for i := 0; i < 10; i++ {
		_, err := engine.ExecuteQuery(ctx, testURI, db, `db.idx.createIndex({ c: 1, a: -1, b: 1 })`)
		require.NoError(t, err)
		_, err = engine.ExecuteQuery(ctx, testURI, db, `db.idx.createIndexes([{ z: 1, y: 1, x: 1 }])`)
		require.NoError(t, err)
	}

	specs, err := testClient.Database(db).Collection("idx").Indexes().ListSpecifications(ctx)
	require.NoError(t, err)
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		names = append(names, s.Name)
	}
	assert.ElementsMatch(t, []string{"_id_", "c_1_a_-1_b_1", "z_1_y_1_x_1"}, names)
}

func TestIntegration_KeyOrder_CompoundSort(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.s.insertMany([
		{ n: 1, g: "b", v: 1 }, { n: 2, g: "a", v: 2 },
		{ n: 3, g: "b", v: 3 }, { n: 4, g: "a", v: 4 }
	])`)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		result, err := engine.ExecuteQuery(ctx, testURI, db,
			`db.s.find({}, { _id: 0, n: 1 }).sort({ g: 1, v: -1 }).toArray().map(function (d) { return d.n; }).join(",")`)
		require.NoError(t, err)
		require.Equal(t, "4,2,3,1", resultText(result))

		result, err = engine.ExecuteQuery(ctx, testURI, db,
			`db.s.aggregate([{ $sort: { g: 1, v: -1 } }]).toArray().map(function (d) { return d.n; }).join(",")`)
		require.NoError(t, err)
		require.Equal(t, "4,2,3,1", resultText(result))
	}
}

func TestIntegration_KeyOrder_InsertedFieldOrder(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.f.insertOne({ _id: 1, zebra: 1, apple: 2, mango: { y: 1, x: 2, w: 3 } })`)
	require.NoError(t, err)

	var raw bson.Raw
	require.NoError(t, testClient.Database(db).Collection("f").FindOne(ctx, bson.D{}).Decode(&raw))
	var doc bson.D
	require.NoError(t, bson.Unmarshal(raw, &doc))
	assert.Equal(t, []string{"_id", "zebra", "apple", "mango"}, docKeys(doc))
	assert.Equal(t, []string{"y", "x", "w"}, docKeys(doc[3].Value.(bson.D)))

	// An exact embedded-document match only succeeds with the same field order.
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.f.countDocuments({ mango: { y: 1, x: 2, w: 3 } })`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "1")
}

func TestIntegration_KeyOrder_EJSONParseCommand(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	for i := 0; i < 10; i++ {
		result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand(EJSON.parse(
			'{"update": "e", "updates": [{"q": {"k": 1}, "u": {"$set": {"k": 1}}, "upsert": true}], "ordered": false}'
		)).ok`)
		require.NoError(t, err)
		require.Equal(t, "1", resultText(result))
	}
}

// Documents a script reads and writes back keep their field order, so a copy
// is identical and exact embedded-document matches built from it still hit.
func TestIntegration_KeyOrder_ReadModifyWrite(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.src.insertOne({ _id: 1, zebra: 1, apple: 2, mango: { y: 1, x: 2, w: 3 }, list: [{ q: 1, p: 2 }] })`)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, err = engine.ExecuteQuery(ctx, testURI, db, `
			db.dst.deleteMany({});
			db.src.find().forEach(function (d) { db.dst.insertOne(d); });
			var one = db.src.find({ _id: 1 }).toArray()[0];
			db.dst.replaceOne({ _id: 1 }, one);
			var agg = db.src.aggregate([{ $match: {} }]).toArray()[0];
			db.dst.insertOne(Object.assign({ copy: "agg" }, agg, { _id: 2 }));`)
		require.NoError(t, err)

		for _, id := range []int32{1, 2} {
			var raw bson.Raw
			require.NoError(t, testClient.Database(db).Collection("dst").FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&raw))
			var doc bson.D
			require.NoError(t, bson.Unmarshal(raw, &doc))
			want := []string{"_id", "zebra", "apple", "mango", "list"}
			if id == 2 {
				// The server always moves _id to the front.
				want = []string{"_id", "copy", "zebra", "apple", "mango", "list"}
			}
			require.Equal(t, want, docKeys(doc))
			mango := docField(doc, "mango").(bson.D)
			require.Equal(t, []string{"y", "x", "w"}, docKeys(mango))
		}

		result, err := engine.ExecuteQuery(ctx, testURI, db,
			`var m = db.src.find().toArray()[0].mango; db.dst.countDocuments({ mango: m })`)
		require.NoError(t, err)
		require.Equal(t, "2", resultText(result))
	}
}

// Results shown in the UI keep the document's field order, for both query
// results and plain values a script returns.
func TestIntegration_KeyOrder_DisplayedResults(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.d.insertOne({ _id: 1, zebra: 1, apple: 2, mango: 3 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.d.find()`)
	require.NoError(t, err)
	data, err := json.Marshal(result.Documents)
	require.NoError(t, err)
	assert.Equal(t, `[{"_id":{"$numberLong":"1"},"zebra":{"$numberLong":"1"},"apple":{"$numberLong":"2"},"mango":{"$numberLong":"3"}}]`, string(data))

	result, err = engine.ExecuteQuery(ctx, testURI, db, `({ zebra: 1, apple: "a", mango: { y: 1, x: 2 } })`)
	require.NoError(t, err)
	data, err = json.Marshal(result.Documents)
	require.NoError(t, err)
	assert.Equal(t, `[{"zebra":1,"apple":"a","mango":{"y":1,"x":2}}]`, string(data))
}

func TestIntegration_KeyOrder_BulkWriteFirstKeyIsOperation(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	for i := 0; i < 10; i++ {
		_, err := engine.ExecuteQuery(ctx, testURI, db,
			`db.bw.bulkWrite([{ insertOne: { document: { n: 1 } }, comment: "ignored" }])`)
		require.NoError(t, err)
	}
	count, err := testClient.Database(db).Collection("bw").CountDocuments(ctx, bson.D{})
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

// Empty arrays must stay arrays in displayed results (the edit dialog saves
// the displayed document back) and in documents a script copies.
func TestIntegration_EmptyArraysSurviveResults(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.ea.insertOne({ _id: 1, tags: [], meta: {}, nested: { list: [] } })`)
	require.NoError(t, err)

	for _, q := range []string{`db.ea.find()`, `db.ea.aggregate([{ $match: {} }])`} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, q)
		require.NoError(t, err)
		assert.Equal(t, `[{"_id":{"$numberLong":"1"},"tags":[],"meta":{},"nested":{"list":[]}}]`, resultText(result), q)
	}

	_, err = engine.ExecuteQuery(ctx, testURI, db,
		`db.ea.find().forEach(function (d) { d._id = 2; db.ea.insertOne(d); })`)
	require.NoError(t, err)
	var copied bson.D
	require.NoError(t, testClient.Database(db).Collection("ea").FindOne(ctx, bson.D{{Key: "_id", Value: int64(2)}}).Decode(&copied))
	assert.Equal(t, bson.A{}, docField(copied, "tags"))
	assert.Equal(t, bson.D{{Key: "list", Value: bson.A{}}}, docField(copied, "nested"))
}

// An unpaged cursor never builds a PageContext, so a non-document filter
// falls back to an empty filter as it always did.
func TestIntegration_NonDocumentFilter(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.nd.insertMany([{ n: 1 }, { n: 2 }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.nd.find(1).toArray().length`)
	require.NoError(t, err)
	assert.Equal(t, "2", resultText(result))

	paged := NewGojaEngine(testClient, 25, "")
	result, err = paged.ExecuteQuery(ctx, testURI, db, `db.nd.find(1)`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 2)
}

func TestIntegration_KeyOrder_FindAndModifyReadBack(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.fm.insertOne({ _id: 1, zebra: 1, apple: 2, mango: { y: 1, x: 2 } })`)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, err = engine.ExecuteQuery(ctx, testURI, db, `
			var d = db.fm.findAndModify({ query: { _id: 1 }, update: { $inc: { n: 1 } }, new: true });
			db.fm.replaceOne({ _id: 1 }, d);`)
		require.NoError(t, err)

		var doc bson.D
		require.NoError(t, testClient.Database(db).Collection("fm").FindOne(ctx, bson.D{}).Decode(&doc))
		require.Equal(t, []string{"_id", "zebra", "apple", "mango", "n"}, docKeys(doc))
		require.Equal(t, []string{"y", "x"}, docKeys(docField(doc, "mango").(bson.D)))
	}
}

// getCollectionInfos returns each collection's fields in server order. The
// driver already decodes nested documents (such as a validator) as ordered
// bson.D, so only the top level of each info document was at risk.
func TestIntegration_KeyOrder_CollectionInfo(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ create: "val",
		validator: { $jsonSchema: { properties: { z: { bsonType: "string" }, a: { bsonType: "string" } } } } })`)
	require.NoError(t, err)

	cursor, err := testClient.Database(db).ListCollections(ctx, bson.D{{Key: "name", Value: "val"}})
	require.NoError(t, err)
	var server []bson.D
	require.NoError(t, cursor.All(ctx, &server))
	require.Len(t, server, 1)
	want := strings.Join(docKeys(server[0]), ",")

	for i := 0; i < 10; i++ {
		result, err := engine.ExecuteQuery(ctx, testURI, db,
			`Object.keys(db.getCollectionInfos({ name: "val" })[0]).join(",")`)
		require.NoError(t, err)
		require.Equal(t, want, resultText(result))
	}
}

func TestIntegration_KeyOrder_ExplainMatchesServer(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.ex.insertOne({ n: 1 })`)
	require.NoError(t, err)

	var server bson.D
	require.NoError(t, testClient.Database(db).RunCommand(ctx, bson.D{
		{Key: "explain", Value: bson.D{{Key: "find", Value: "ex"}, {Key: "filter", Value: bson.D{}}}},
		{Key: "verbosity", Value: "queryPlanner"},
	}).Decode(&server))

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.ex.find({}).explain()`)
	require.NoError(t, err)
	require.Len(t, result.Documents, 1)
	// explain() hands its result to the script, so the final value comes back
	// as the ordered document the script saw.
	fields, ok := result.Documents[0].(bson.D)
	require.True(t, ok, "got %T", result.Documents[0])
	assert.Equal(t, docKeys(server), docKeys(fields))
}
