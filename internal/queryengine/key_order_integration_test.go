//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
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
