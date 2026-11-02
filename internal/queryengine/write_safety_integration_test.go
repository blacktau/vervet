//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// setupWrites starts each script from the same three documents.
func setupWrites(t *testing.T) (func(script string) (string, error), string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db := dbName(t)
	t.Cleanup(func() { testClient.Database(db).Drop(context.Background()) })

	engine := NewGojaEngine(testClient, 0, "")
	return func(script string) (string, error) {
		_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.drop(); db.c.insertMany([
			{ _id: 1, s: "r", p: 1, a: [{ k: 1 }, { k: 2 }] },
			{ _id: 2, s: "r", p: 9, a: [{ k: 1 }] },
			{ _id: 3, s: "r", p: 5, a: [] },
		])`)
		require.NoError(t, err)
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		return printedText(result), err
	}, db
}

// A write with no usable filter must fail and change nothing. Each of these
// used to act on the whole collection, or on an arbitrary document.
func TestIntegration_Writes_BadFiltersAreRejected(t *testing.T) {
	run, db := setupWrites(t)
	cases := map[string]string{
		`db.c.deleteMany()`:                                     "deleteMany requires a filter document",
		`db.c.deleteMany(null)`:                                 "deleteMany requires a filter document",
		`db.c.deleteMany(undefined)`:                            "deleteMany requires a filter document",
		`db.c.deleteMany("x")`:                                  "deleteMany filter must be a document",
		`db.c.deleteMany(ObjectId())`:                           "deleteMany filter must be a document",
		`db.c.deleteMany([])`:                                   "deleteMany filter must be a document",
		`db.c.deleteMany(/x/)`:                                  "deleteMany filter must be a document",
		`db.c.deleteOne()`:                                      "deleteOne requires a filter document",
		`db.c.deleteOne(5)`:                                     "deleteOne filter must be a document",
		`db.c.updateOne(null, { $set: { z: 1 } })`:              "updateOne requires a filter document",
		`db.c.updateMany(ObjectId(), { $set: { z: 1 } })`:       "updateMany filter must be a document",
		`db.c.replaceOne(5, { z: 1 })`:                          "replaceOne filter must be a document",
		`db.c.replaceOne({ _id: 1 }, 5)`:                        "replaceOne replacement must be a document",
		`db.c.findOneAndDelete()`:                               "findOneAndDelete requires a filter document",
		`db.c.findOneAndDelete(null)`:                           "findOneAndDelete requires a filter document",
		`db.c.findOneAndUpdate(ObjectId(), { $set: { z: 1 } })`: "findOneAndUpdate filter must be a document",
		`db.c.findOneAndReplace({ _id: 1 }, 5)`:                 "findOneAndReplace replacement must be a document",
		`db.c.findOneAndReplace({ _id: 1 }, null)`:              "findOneAndReplace replacement must be a document",

		`db.c.bulkWrite([{ deleteMany: { filtr: { x: 1 } } }])`:                         "bulkWrite deleteMany requires a filter document",
		`db.c.bulkWrite([{ deleteOne: {} }])`:                                           "bulkWrite deleteOne requires a filter document",
		`db.c.bulkWrite([{ updateMany: { update: { $set: { z: 1 } } } }])`:              "bulkWrite updateMany requires a filter document",
		`db.c.bulkWrite([{ updateOne: { update: { $set: { z: 1 } }, upsert: true } }])`: "bulkWrite updateOne requires a filter document",
		`db.c.bulkWrite([{ updateOne: { filter: { _id: 1 } } }])`:                       "bulkWrite updateOne requires an update",
		`db.c.bulkWrite([{ replaceOne: { replacement: { z: 1 } } }])`:                   "bulkWrite replaceOne requires a filter document",
		`db.c.bulkWrite([{ replaceOne: { filter: { _id: 1 } } }])`:                      "bulkWrite replaceOne replacement must be a document",
		`db.c.bulkWrite([{ insertOne: {} }])`:                                           "bulkWrite insertOne requires a document",
		// Valid operations before a bad one must not run either.
		`db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }, { deleteMany: {} }])`: "bulkWrite deleteMany requires a filter document",
	}
	for script, want := range cases {
		_, err := run(script)
		require.Error(t, err, script)
		assert.Contains(t, err.Error(), want, script)
		assertUnchanged(t, db, script)
	}

	// An explicit {} still means every document.
	got, err := run(`print(db.c.deleteMany({}).deletedCount)`)
	require.NoError(t, err)
	assert.Equal(t, "3", got)
}

func assertUnchanged(t *testing.T, db, script string) {
	t.Helper()
	coll := testClient.Database(db).Collection("c")
	n, err := coll.CountDocuments(context.Background(), map[string]any{"s": "r", "z": map[string]any{"$exists": false}})
	require.NoError(t, err)
	assert.Equal(t, int64(3), n, script)
}

// An option that is not applied must not be dropped silently.
func TestIntegration_Writes_UnknownOptionsAreRejected(t *testing.T) {
	run, db := setupWrites(t)
	cases := map[string]string{
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { upsrt: true })`:                                  `updateOne option "upsrt" is not supported`,
		`db.c.replaceOne({ _id: 1 }, { z: 1 }, { bogus: 1 })`:                                              `replaceOne option "bogus" is not supported`,
		`db.c.deleteOne({ _id: 1 }, { bogus: 1 })`:                                                         `deleteOne option "bogus" is not supported`,
		`db.c.deleteMany({}, { bogus: 1 })`:                                                                `deleteMany option "bogus" is not supported`,
		`db.c.findOneAndDelete({}, { bogus: 1 })`:                                                          `findOneAndDelete option "bogus" is not supported`,
		`db.c.findOneAndUpdate({}, { $set: { z: 1 } }, { bogus: 1 })`:                                      `findOneAndUpdate option "bogus" is not supported`,
		`db.c.findOneAndReplace({}, { z: 1 }, { bogus: 1 })`:                                               `findOneAndReplace option "bogus" is not supported`,
		`db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }], { bogus: 1 })`:                            `bulkWrite option "bogus" is not supported`,
		`db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 }, bogus: 1 } }])`:                                `bulkWrite deleteOne option "bogus" is not supported`,
		`db.c.bulkWrite([{ updateOne: { filter: { _id: 1 }, update: { $set: { z: 1 } }, upsrt: true } }])`: `bulkWrite updateOne option "upsrt" is not supported`,
		`db.c.insertOne({ z: 1 }, { bogus: 1 })`:                                                           `insertOne option "bogus" is not supported`,
		`db.c.insertMany([{ z: 1 }], { bogus: 1 })`:                                                        `insertMany option "bogus" is not supported`,
		`db.c.countDocuments({}, { bogus: 1 })`:                                                            `countDocuments option "bogus" is not supported`,
		`db.c.aggregate([], { bogus: 1 })`:                                                                 `aggregate option "bogus" is not supported`,
		`db.aggregate([{ $documents: [] }], { bogus: 1 })`:                                                 `aggregate option "bogus" is not supported`,
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { upsert: "yes" })`:                                `updateOne upsert must be a boolean`,
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, 5)`:                                                `updateOne options must be an object`,
		`db.c.findOneAndUpdate({}, { $set: { z: 1 } }, { returnDocument: "new" })`:                         `findOneAndUpdate returnDocument must be "before" or "after"`,
	}
	for script, want := range cases {
		_, err := run(script)
		require.Error(t, err, script)
		assert.Contains(t, err.Error(), want, script)
		assertUnchanged(t, db, script)
	}

	// Null and undefined options count as absent.
	got, err := run(`print(db.c.deleteOne({ _id: 1 }, { hint: null, comment: undefined }).deletedCount)`)
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

func TestIntegration_Writes_OptionsAreApplied(t *testing.T) {
	run, _ := setupWrites(t)
	cases := map[string]struct{ script, want string }{
		"updateOne upsert": {`var r = db.c.updateOne({ _id: 99 }, { $set: { x: 1 } }, { upsert: true });
			print(r.insertedId, r.matchedCount, r.modifiedCount, r.upsertedCount, db.c.countDocuments({ _id: 99 }))`, "99 0 0 1 1"},
		"updateMany upsert": {`var r = db.c.updateMany({ _id: 99 }, { $set: { x: 1 } }, { upsert: true });
			print(r.insertedId, r.upsertedCount, db.c.countDocuments({ _id: 99 }))`, "99 1 1"},
		"replaceOne upsert": {`var r = db.c.replaceOne({ _id: 99 }, { x: 1 }, { upsert: true });
			print(r.insertedId, r.upsertedCount, db.c.countDocuments({ _id: 99 }))`, "99 1 1"},
		"no upsert has null insertedId": {`print(db.c.updateOne({ _id: 1 }, { $set: { x: 1 } }).insertedId)`, "null"},
		"arrayFilters": {`db.c.updateOne({ _id: 1 }, { $set: { "a.$[e].k": 7 } }, { arrayFilters: [{ "e.k": 2 }] });
			print(db.c.findOne({ _id: 1 }).a.map(e => e.k).join())`, "1,7"},
		"updateMany arrayFilters": {`db.c.updateMany({}, { $set: { "a.$[e].k": 7 } }, { arrayFilters: [{ "e.k": 1 }] });
			print(db.c.findOne({ _id: 2 }).a[0].k)`, "7"},
		"update collation": {`db.c.insertOne({ _id: 10, t: "ABC" });
			print(db.c.updateOne({ t: "abc" }, { $set: { x: 1 } }, { collation: { locale: "en", strength: 2 } }).modifiedCount)`, "1"},
		"delete collation": {`db.c.insertOne({ _id: 10, t: "ABC" });
			print(db.c.deleteMany({ t: "abc" }, { collation: { locale: "en", strength: 2 } }).deletedCount)`, "1"},
		"delete hint is sent": {`try { db.c.deleteOne({ _id: 1 }, { hint: "nope" }); print("no error") } catch (e) { print("rejected") }`, "rejected"},
		"insertMany unordered": {`try { db.c.insertMany([{ _id: 1 }, { _id: 50 }], { ordered: false }) } catch (e) {}
			print(db.c.countDocuments({ _id: 50 }))`, "1"},
		"insertMany ordered stops": {`try { db.c.insertMany([{ _id: 1 }, { _id: 50 }]) } catch (e) {}
			print(db.c.countDocuments({ _id: 50 }))`, "0"},
		"bulkWrite unordered": {`try { db.c.bulkWrite([{ insertOne: { document: { _id: 1 } } }, { insertOne: { document: { _id: 50 } } }], { ordered: false }) } catch (e) {}
			print(db.c.countDocuments({ _id: 50 }))`, "1"},
		"bulkWrite upsert": {`var r = db.c.bulkWrite([
				{ updateOne: { filter: { _id: 60 }, update: { $set: { x: 1 } }, upsert: true } },
				{ updateMany: { filter: { _id: 61 }, update: { $set: { x: 1 } }, upsert: true } },
				{ replaceOne: { filter: { _id: 62 }, replacement: { x: 1 }, upsert: true } },
			]);
			print(r.upsertedCount, r.upsertedIds[0], r.upsertedIds[1], r.upsertedIds[2], db.c.countDocuments({ x: 1 }))`, "3 60 61 62 3"},
		"bulkWrite arrayFilters": {`db.c.bulkWrite([{ updateOne: { filter: { _id: 1 }, update: { $set: { "a.$[e].k": 7 } }, arrayFilters: [{ "e.k": 2 }] } }]);
			print(db.c.findOne({ _id: 1 }).a[1].k)`, "7"},
		"bulkWrite insertedIds": {`db.c.deleteMany({});
			var r = db.c.bulkWrite([{ deleteOne: { filter: { _id: 0 } } }, { insertOne: { document: { _id: 7 } } }, { insertOne: { document: { n: 1 } } }]);
			var id = r.insertedIds[2];
			print(r.insertedCount, r.insertedIds[1], Object.keys(r.insertedIds).join(), Object.keys(r.upsertedIds).length, db.c.countDocuments({ _id: id }))`, "2 7 1,2 0 1"},
		"bulkWrite keeps field order": {`print(Object.keys(db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }])).join())`,
			"acknowledged,insertedCount,insertedIds,matchedCount,modifiedCount,deletedCount,upsertedCount,upsertedIds"},
		"findOneAndDelete sort": {`print(db.c.findOneAndDelete({ s: "r" }, { sort: { p: -1 } })._id, db.c.countDocuments({ _id: 2 }))`, "2 0"},
		"findOneAndUpdate sort and after": {`var d = db.c.findOneAndUpdate({ s: "r" }, { $set: { z: 1 } }, { sort: { p: -1 }, returnDocument: "after" });
			print(d._id, d.z)`, "2 1"},
		"findOneAndUpdate before by default": {`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }).p)`, "1"},
		"findOneAndUpdate returnNewDocument": {`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { returnNewDocument: true }).p)`, "50"},
		"findOneAndUpdate projection":        {`print(Object.keys(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { projection: { p: 1, _id: 0 } })).join())`, "p"},
		"findOneAndUpdate upsert":            {`print(db.c.findOneAndUpdate({ _id: 70 }, { $set: { z: 1 } }, { upsert: true, returnDocument: "after" }).z)`, "1"},
		"findOneAndUpdate arrayFilters":      {`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { "a.$[e].k": 7 } }, { arrayFilters: [{ "e.k": 2 }], returnDocument: "after" }).a[1].k)`, "7"},
		"findOneAndReplace": {`var d = db.c.findOneAndReplace({ s: "r" }, { z: 1 }, { sort: { p: -1 }, returnDocument: "after", projection: { _id: 1 } });
			print(d._id, db.c.countDocuments({ z: 1 }))`, "2 1"},
		"findOneAnd no match is null": {`print(db.c.findOneAndDelete({ _id: 500 }), db.c.findOneAndUpdate({ _id: 500 }, { $set: { z: 1 } }), db.c.findOneAndReplace({ _id: 500 }, { z: 1 }))`, "null null null"},
		"count options":               {`print(db.c.countDocuments({}, { limit: 1 }), db.c.countDocuments({}, { skip: 2 }), db.c.countDocuments({}, undefined))`, "1 1 3"},
		"aggregate options": {`print(db.c.aggregate([{ $match: { t: "abc" } }], { allowDiskUse: true, cursor: { batchSize: 1 }, maxTimeMS: 5000 }).toArray().length);
			db.c.insertOne({ _id: 10, t: "ABC" });
			print(db.c.aggregate([{ $match: { t: "abc" } }], { collation: { locale: "en", strength: 2 } }).toArray().length)`, "0\n1"},
		"db.aggregate options":               {`print(db.aggregate([{ $documents: [{ t: "ABC" }] }, { $match: { t: "abc" } }], { collation: { locale: "en", strength: 2 } }).length)`, "1"},
		"a findOne result works as a filter": {`print(db.c.deleteOne(db.c.findOne({ _id: 1 })).deletedCount)`, "1"},
	}
	for name, tc := range cases {
		got, err := run(tc.script)
		require.NoError(t, err, name)
		assert.Equal(t, tc.want, got, name)
	}
}

// mongosh sends the writeConcern on every write method. The test server is a
// standalone, which refuses w: 2 before writing, so that error proves it arrived.
func TestIntegration_Writes_WriteConcernIsApplied(t *testing.T) {
	run, db := setupWrites(t)
	for _, script := range []string{
		`db.c.insertOne({ z: 1 }, { writeConcern: { w: 2 } })`,
		`db.c.insertMany([{ z: 1 }], { writeConcern: { w: 2 } })`,
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { writeConcern: { w: 2 } })`,
		`db.c.updateMany({}, { $set: { z: 1 } }, { writeConcern: { w: 2 } })`,
		`db.c.replaceOne({ _id: 1 }, { z: 1 }, { writeConcern: { w: 2 } })`,
		`db.c.deleteOne({ _id: 1 }, { writeConcern: { w: 2 } })`,
		`db.c.deleteMany({}, { writeConcern: { w: 2 } })`,
		`db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }], { writeConcern: { w: 2 } })`,
		`db.c.findOneAndDelete({ _id: 1 }, { writeConcern: { w: 2 } })`,
		`db.c.findOneAndUpdate({ _id: 1 }, { $set: { z: 1 } }, { writeConcern: { w: 2 } })`,
		`db.c.findOneAndReplace({ _id: 1 }, { z: 1 }, { writeConcern: { w: 2 } })`,
	} {
		_, err := run(script)
		require.Error(t, err, script)
		assert.Contains(t, err.Error(), "cannot use 'w' > 1 when a host is not replicated", script)
		assertUnchanged(t, db, script)
	}

	cases := map[string]string{
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: "majority" } }).acknowledged)`:                              "true",
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: 1, j: true } }).acknowledged)`:                              "true",
		`print(db.c.updateOne({ _id: 1 }, { $set: { x: 1 } }, { upsert: true, writeConcern: { j: true } }).modifiedCount)`: "1",
		`print(db.c.deleteOne({ _id: 1 }, { writeConcern: null }).deletedCount)`:                                           "1",
		`print(db.c.findOneAndDelete({ _id: 1 }, { sort: { p: 1 }, writeConcern: { w: 1 } })._id)`:                         "1",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}

	for script, want := range map[string]string{
		`db.c.insertOne({ z: 1 }, { writeConcern: 5 })`:                      "insertOne writeConcern must be an object",
		`db.c.insertOne({ z: 1 }, { writeConcern: { w: 1.5 } })`:             "insertOne writeConcern w must be a whole number",
		`db.c.insertOne({ z: 1 }, { writeConcern: { j: 1 } })`:               "insertOne writeConcern j must be a boolean",
		`db.c.insertOne({ z: 1 }, { writeConcern: { w: 1, wtimeout: 10 } })`: `insertOne writeConcern option "wtimeout" is not supported`,
		`db.c.insertOne({ z: 1 }, { writeConcern: { bogus: 1 } })`:           `insertOne writeConcern option "bogus" is not supported`,
		`db.c.aggregate([], { writeConcern: { w: 1 } })`:                     `aggregate option "writeConcern" is not supported`,
		`db.c.countDocuments({}, { writeConcern: { w: 1 } })`:                `countDocuments option "writeConcern" is not supported`,
	} {
		_, err := run(script)
		require.Error(t, err, script)
		assert.Contains(t, err.Error(), want, script)
		assertUnchanged(t, db, script)
	}
}

// With w: 0 the result says the write was not acknowledged, and the write still
// happens.
func TestIntegration_Writes_UnacknowledgedWrites(t *testing.T) {
	run, db := setupWrites(t)
	cases := map[string]string{
		`var r = db.c.updateOne({ _id: 2 }, { $set: { z: 1 } }, { writeConcern: { w: 0 } }); print(Object.keys(r).join(), r.acknowledged)`: "acknowledged,insertedId,upsertedCount false",
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: 0 } }).acknowledged)`:                                                       "false",
		`print(db.c.insertMany([{ _id: 5 }], { writeConcern: { w: 0 } }).acknowledged)`:                                                    "false",
		`print(db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }], { writeConcern: { w: 0 } }).acknowledged)`:                          "false",
		`print(db.c.findOneAndDelete({ _id: 1 }, { writeConcern: { w: 0 } }))`:                                                             "null",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}

	got, err := run(`var r = db.c.deleteOne({ _id: 1 }, { writeConcern: { w: 0 } }); print(Object.keys(r).join(), r.acknowledged)`)
	require.NoError(t, err)
	assert.Equal(t, "acknowledged false", got)
	// The server may not have applied it yet.
	assert.Eventually(t, func() bool {
		n, err := testClient.Database(db).Collection("c").CountDocuments(context.Background(), bson.D{{Key: "_id", Value: 1}})
		return err == nil && n == 0
	}, 5*time.Second, 10*time.Millisecond)
}

// AffectedCount counts upserts as well as modified documents.
func TestIntegration_Writes_AffectedCountIncludesUpserts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(context.Background())

	upsert := bson.D{{Key: "upsert", Value: true}}
	set := bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 1}}}}
	filter := func(id int) bson.D { return bson.D{{Key: "_id", Value: id}} }
	for _, tc := range []struct {
		method string
		args   []any
		want   int
	}{
		{"updateOne", []any{filter(1), set, upsert}, 1},
		{"updateMany", []any{filter(2), set, upsert}, 1},
		{"replaceOne", []any{filter(3), bson.D{{Key: "x", Value: 1}}, upsert}, 1},
		{"bulkWrite", []any{[]any{
			bson.D{{Key: "updateOne", Value: bson.D{{Key: "filter", Value: filter(4)}, {Key: "update", Value: set}, {Key: "upsert", Value: true}}}},
			bson.D{{Key: "updateOne", Value: bson.D{{Key: "filter", Value: filter(5)}, {Key: "update", Value: set}, {Key: "upsert", Value: true}}}},
		}}, 2},
	} {
		result, err := dispatch(ctx, testClient, db, CapturedOp{Collection: "c", Method: tc.method, Args: tc.args})
		require.NoError(t, err, tc.method)
		assert.Equal(t, tc.want, result.AffectedCount, tc.method)
	}
}

// A findOneAnd* result shows as the one document it returned.
func TestIntegration_Writes_FindOneAndDisplaysOneDocument(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(context.Background())

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: 1 })`)
	require.NoError(t, err)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.findOneAndUpdate({ _id: 1 }, { $set: { z: 1 } }, { returnDocument: "after" })`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 1)
	assert.Contains(t, resultText(result), `"z"`)
}
