//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vervet/internal/models"

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

// A cursor read during the script and then returned is still shown paged.
func TestIntegration_CursorReadInScriptStaysPaged(t *testing.T) {
	ctx, db, _ := keyOrderEngine(t)
	engine := NewGojaEngine(testClient, 5, "")

	_, err := engine.ExecuteQuery(ctx, testURI, db, `for (var i = 0; i < 12; i++) { db.big.insertOne({ n: i }); }`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `const c = db.big.find({}, { _id: 0 }).sort({ n: 1 }); c.hasNext(); c`)
	require.NoError(t, err)
	require.NotNil(t, result.PageContext, "the returned cursor must keep its pager")
	assert.Len(t, result.Documents, 5)

	page2, err := engine.FetchPage(ctx, db, *result.PageContext, 1, 5)
	require.NoError(t, err)
	assert.Equal(t, `[{"n":{"$numberLong":"5"}},{"n":{"$numberLong":"6"}},{"n":{"$numberLong":"7"}},{"n":{"$numberLong":"8"}},{"n":{"$numberLong":"9"}}]`, resultText(page2))

	// After next() has consumed documents, the rest are shown a page at a
	// time, continuing after the consumed ones.
	result, err = engine.ExecuteQuery(ctx, testURI, db, `const d = db.big.find({}, { _id: 0 }).sort({ n: 1 }); d.next(); d.next(); d`)
	require.NoError(t, err)
	require.NotNil(t, result.PageContext)
	assert.Equal(t, `[{"n":{"$numberLong":"2"}},{"n":{"$numberLong":"3"}},{"n":{"$numberLong":"4"}},{"n":{"$numberLong":"5"}},{"n":{"$numberLong":"6"}}]`, resultText(result))
	page2, err = engine.FetchPage(ctx, db, *result.PageContext, 1, 5)
	require.NoError(t, err)
	assert.Equal(t, `[{"n":{"$numberLong":"7"}},{"n":{"$numberLong":"8"}},{"n":{"$numberLong":"9"}},{"n":{"$numberLong":"10"}},{"n":{"$numberLong":"11"}}]`, resultText(page2))
}

// Cursor methods continue from where next() left off, as in mongosh, and a
// script's printed output is not replaced by the cursor it returns.
func TestIntegration_CursorMethodsContinueAfterNext(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `for (var i = 0; i < 4; i++) { db.cm.insertOne({ n: i }); }`)
	require.NoError(t, err)

	for _, tc := range []struct{ script, want string }{
		{`var c = db.cm.find().sort({ n: 1 }); c.next(); c.toArray().map(function (d) { return d.n; }).join()`, "1,2,3"},
		{`var c = db.cm.find().sort({ n: 1 }); c.next(); var seen = []; c.forEach(function (d) { seen.push(d.n); }); seen.join()`, "1,2,3"},
		{`var c = db.cm.find().sort({ n: 1 }); c.next(); c.map(function (d) { return d.n; }).join()`, "1,2,3"},
		{`var c = db.cm.find().sort({ n: 1 }); c.toArray(); c.hasNext()`, "false"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, resultText(result), tc.script)
	}

	result, err := engine.ExecuteQuery(ctx, testURI, db,
		`var c = db.cm.find().sort({ n: 1 }); while (c.hasNext()) { print(c.next().n); } c`)
	require.NoError(t, err)
	assert.Equal(t, "0\n1\n2\n3", result.RawOutput)

	result, err = engine.ExecuteQuery(ctx, testURI, db, `var f = db.cm.findOne({ n: 2 }); f.hasNext(); f`)
	require.NoError(t, err)
	assert.Equal(t, "findOne", result.OperationType)
}

// Values from EJSON.parse and command results keep their BSON types, so they
// work in queries and writes.
func TestIntegration_ParsedAndCommandValuesKeepBSONTypes(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	oid := bson.NewObjectID()
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.bt.insertOne(EJSON.parse('{"_id": {"$oid": "`+oid.Hex()+`"}, "at": {"$date": "2026-01-02T03:04:05Z"}, "n": {"$numberLong": "7"}}'))`)
	require.NoError(t, err)
	var stored bson.D
	require.NoError(t, testClient.Database(db).Collection("bt").FindOne(ctx, bson.D{}).Decode(&stored))
	assert.Equal(t, oid, docField(stored, "_id"))
	assert.IsType(t, bson.DateTime(0), docField(stored, "at"))
	assert.Equal(t, int64(7), docField(stored, "n"))

	for _, tc := range []struct{ script, want string }{
		{`db.runCommand(EJSON.parse('{"find": "bt", "filter": {"_id": {"$oid": "` + oid.Hex() + `"}}}')).cursor.firstBatch.length`, "1"},
		{`db.runCommand({ find: "bt" }).cursor.firstBatch[0]._id.toHexString()`, oid.Hex()},
		{`db.runCommand({ find: "bt" }).cursor.firstBatch[0].at instanceof Date`, "true"},
		{`db.bt.findAndModify({ query: {}, update: { $set: { x: 1 } }, new: true })._id.toHexString()`, oid.Hex()},
		{`typeof (db.stats().storageSize / 1024)`, "number"},
		{`var r = db.runCommand({ find: "bt" }); String(db.bt.deleteOne({ _id: r.cursor.firstBatch[0]._id }).deletedCount)`, "1"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, strings.Trim(resultText(result), `"`), tc.script)
	}
}

// Command results display BSON values as Extended JSON, not Go structs (#304).
func TestIntegration_CommandResultsDisplayBSONTypes(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	result, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.runCommand({ update: "u", updates: [{ q: { r: 1 }, u: { $setOnInsert: { _id: UUID("ce9313d1-4caa-45b6-969d-1161e00ad853") } }, upsert: true }] })`)
	require.NoError(t, err)
	text := resultText(result)
	assert.Contains(t, text, `"_id":{"$binary":{"base64":"zpMT0UyqRbaWnRFh4ArYUw==","subType":"04"}}`)
	assert.NotContains(t, text, "Subtype")
}

// Write and count results list their fields in the same order on every run.
func TestIntegration_KeyOrder_WriteResults(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	want := map[string]string{
		`db.w.insertOne({ _id: 1 })`:                        `[{"acknowledged":true,"insertedId":{"$numberLong":"1"}}]`,
		`db.w.updateOne({ _id: 1 }, { $set: { a: 1 } })`:    `[{"acknowledged":true,"matchedCount":{"$numberLong":"1"},"modifiedCount":{"$numberLong":"1"}}]`,
		`Object.keys(db.w.insertMany([{ _id: 2 }])).join()`: `acknowledged,insertedIds`,
	}
	for i := 0; i < 10; i++ {
		_, err := engine.ExecuteQuery(ctx, testURI, db, `db.w.deleteMany({})`)
		require.NoError(t, err)
		for _, q := range []string{
			`db.w.insertOne({ _id: 1 })`,
			`db.w.updateOne({ _id: 1 }, { $set: { a: 1 } })`,
			`Object.keys(db.w.insertMany([{ _id: 2 }])).join()`,
		} {
			result, err := engine.ExecuteQuery(ctx, testURI, db, q)
			require.NoError(t, err)
			require.Equal(t, want[q], resultText(result), q)
		}
	}
}

// The collation reaches later pages with its types intact, so numeric
// ordering gives one consistent order across pages.
func TestIntegration_FetchPage_CollationSurvivesFrontend(t *testing.T) {
	ctx, db, _ := keyOrderEngine(t)
	engine := NewGojaEngine(testClient, 3, "")

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`["10", "9", "2", "33", "1", "100"].forEach(function (s) { db.coll.insertOne({ s: s }); })`)
	require.NoError(t, err)

	res, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.coll.find({}, { _id: 0 }).sort({ s: 1 }).collation({ locale: "en", numericOrdering: true, strength: 2 })`)
	require.NoError(t, err)
	require.NotNil(t, res.PageContext)
	wire, err := json.Marshal(res.PageContext)
	require.NoError(t, err)
	var pc models.PageContext
	require.NoError(t, json.Unmarshal(wire, &pc))

	page2, err := engine.FetchPage(ctx, db, pc, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, `[{"s":"10"},{"s":"33"},{"s":"100"}]`, resultText(page2))
}

// Paging a cursor the script already read from: a used-up limit gives an empty
// result with no pager, and the count leaves out skipped and consumed
// documents.
func TestIntegration_ReadCursorPagingLimitAndCount(t *testing.T) {
	ctx, db, _ := keyOrderEngine(t)
	engine := NewGojaEngine(testClient, 5, "")

	_, err := engine.ExecuteQuery(ctx, testURI, db, `for (var i = 0; i < 30; i++) { db.lc.insertOne({ n: i }); }`)
	require.NoError(t, err)

	for _, script := range []string{
		`const c = db.lc.find().limit(3); c.next(); c.next(); c.next(); c`,
		`const c = db.lc.find(); c.toArray(); c`,
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err, script)
		assert.Nil(t, result.PageContext, script)
		assert.Empty(t, result.Documents, script)
	}

	for script, want := range map[string]int64{
		`const c = db.lc.find(); c.next(); c`:                     29,
		`db.lc.find().skip(2)`:                                    28,
		`const c = db.lc.find().limit(10); c.next(); c.next(); c`: 8,
		`db.lc.find({ n: { $gte: 10 } }).skip(5)`:                 15,
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err, script)
		require.NotNil(t, result.PageContext, script)
		count, _, err := engine.CountForPage(ctx, db, *result.PageContext)
		require.NoError(t, err, script)
		assert.Equal(t, want, count, script)
	}
}

func TestIntegration_EJSONParseKeepsNumbersPlain(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `EJSON.parse('{"n": 5000000000}').n + 1`)
	require.NoError(t, err)
	assert.Equal(t, "5000000001", resultText(result))
}

// An unread cursor is shown as its documents even when the script printed.
func TestIntegration_UnreadCursorShownDespitePrints(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.up.insertOne({ n: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `var c = db.up.find({}, { _id: 0 }); if (c.hasNext()) { print("found"); } c`)
	require.NoError(t, err)
	assert.Equal(t, `[{"n":{"$numberLong":"1"}}]`, resultText(result))
}

// On an empty result, a script's printed message is not replaced by the
// empty cursor, whichever way the script read it.
func TestIntegration_PrintsShownForEmptyReadCursor(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	for _, script := range []string{
		`var c = db.none.find(); c.forEach(function (d) {}); print("none found"); c`,
		`var c = db.none.find(); while (c.hasNext()) { c.next(); } print("none found"); c`,
		`var c = db.none.find(); c.toArray(); print("none found"); c`,
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err, script)
		assert.Equal(t, "none found", result.RawOutput, script)
	}
}

// More print/cursor combinations: printed output wins once the script has
// read from the cursor, or when its read found nothing.
func TestIntegration_PrintsAndCursorCombinations(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.some.insertMany([{ n: 1 }, { n: 2 }])`)
	require.NoError(t, err)

	for _, tc := range []struct{ script, want string }{
		{`var c = db.none.find(); c.map(function (d) { return d; }); print("none found"); c`, "none found"},
		{`var c = db.none.find(); if (!c.hasNext()) { print("none"); } c`, "none"},
		{`var c = db.some.find().sort({ n: 1 }); c.forEach(function (d) { print(d.n); }); print("done"); c`, "1\n2\ndone"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, result.RawOutput, tc.script)
	}
}

// A forEach/map callback that throws leaves the documents it never reached
// readable, as in mongosh.
func TestIntegration_CallbackErrorLeavesRestReadable(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.cb.insertMany([{ n: 1 }, { n: 2 }, { n: 3 }])`)
	require.NoError(t, err)

	for _, method := range []string{"forEach", "map"} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, `var c = db.cb.find().sort({ n: 1 });
			try { c.`+method+`(function (d) { if (String(d.n) === "1") { throw new Error("stop"); } }); } catch (e) {}
			c.toArray().map(function (d) { return d.n; }).join()`)
		require.NoError(t, err, method)
		assert.Equal(t, "2,3", resultText(result), method)
	}
}
