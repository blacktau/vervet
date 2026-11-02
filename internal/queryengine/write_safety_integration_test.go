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

// Whatever mongosh accepts must run here too, whichever server it is against, so
// these match what mongosh does with each input.
func TestIntegration_Writes_MatchMongoshOnOddOptions(t *testing.T) {
	run, _ := setupWrites(t)
	cases := map[string]string{
		// Unknown options, and options that are not a document, are ignored.
		`print(db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { upsrt: true }).modifiedCount)`:     "1",
		`print(db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, 5).modifiedCount)`:                   "1",
		`print(db.c.deleteOne({ _id: 1 }, "x").deletedCount)`:                                      "1",
		`print(db.c.deleteMany({ _id: 1 }, []).deletedCount)`:                                      "1",
		`print(db.c.deleteOne({ _id: 1 }, { bogus: 1, session: {} }).deletedCount)`:                "1",
		`print(db.c.replaceOne({ _id: 1 }, { z: 1 }, { bogus: 1 }).modifiedCount)`:                 "1",
		`print(db.c.findOneAndDelete({ _id: 1 }, { bogus: 1 })._id)`:                               "1",
		`print(db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 }, bogus: 1 } }], 5).deletedCount)`: "1",
		`print(db.c.insertOne({ _id: 5 }, { w: 2, j: true }).acknowledged)`:                        "true",
		`print(db.c.insertMany([{ _id: 5 }], 5).acknowledged)`:                                     "true",
		`print(db.c.countDocuments({}, { bogus: 1 }))`:                                             "3",
		`print(db.c.aggregate([], { bogus: 1, writeConcern: { w: 2 } }).toArray().length)`:         "3",
		`print(db.c.deleteOne({ _id: 1 }, { hint: null, comment: undefined }).deletedCount)`:       "1",
		// A flag that is not a boolean stays at its default.
		`print(db.c.updateOne({ _id: 99 }, { $set: { z: 1 } }, { upsert: "yes" }).upsertedCount)`:                                    "0",
		`print(db.c.updateOne({ _id: 99 }, { $set: { z: 1 } }, { upsert: 1 }).upsertedCount)`:                                        "0",
		`print(db.c.bulkWrite([{ updateOne: { filter: { _id: 99 }, update: { $set: { z: 1 } }, upsert: 1 } }]).upsertedCount)`:       "0",
		`try { db.c.insertMany([{ _id: 1 }, { _id: 50 }], { ordered: "no" }) } catch (e) {} print(db.c.countDocuments({ _id: 50 }))`: "0",
		`print(db.c.aggregate([], { allowDiskUse: "x" }).toArray().length)`:                                                          "3",
		// A number that is not a number is ignored.
		`print(db.c.countDocuments({}, { limit: "1" }))`:     "3",
		`print(db.c.countDocuments({}, { maxTimeMS: "5" }))`: "3",
		// Every spelling of "return the new version".
		`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { returnNewDocument: 1 }).p)`:  "50",
		`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { returnOriginal: false }).p)`: "50",
		`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { new: true }).p)`:             "50",
		`print(db.c.findOneAndUpdate({ _id: 1 }, { $set: { p: 50 } }, { remove: true }).p)`:          "1",
		// Every form of sort.
		`print(db.c.findOneAndDelete({}, { sort: "p" })._id)`:           "1",
		`print(db.c.findOneAndDelete({}, { sort: [["p", -1]] })._id)`:   "2",
		`print(db.c.findOneAndDelete({}, { sort: ["p", -1] })._id)`:     "2",
		`print(db.c.findOneAndDelete({}, { sort: ["p", "desc"] })._id)`: "2",
		`print(db.c.findOneAndDelete({}, { sort: ["s", "p"] })._id)`:    "1",
		// Stages as separate arguments.
		`print(db.c.aggregate({ $match: { _id: 1 } }, { $limit: 1 }).toArray().length)`: "1",
		`print(db.c.aggregate({ $sort: { _id: 1 } }, { $limit: 2 }).toArray().length)`:  "2",
		// Read settings.
		`print(db.c.countDocuments({}, { readConcern: { level: "local" } }))`:                                                   "3",
		`print(db.c.aggregate([], { readConcern: { level: "local" }, readPreference: "secondaryPreferred" }).toArray().length)`: "3",
		// explain gives the plan rather than the documents.
		`print(db.c.aggregate([], { explain: true }).ok)`:           "1",
		`print(db.c.aggregate([], { explain: "queryPlanner" }).ok)`: "1",
		`print(db.c.aggregate([], { explain: false }).ok)`:          "1",
		`print(db.c.aggregate([], { explain: "" }).length)`:         "3",
		// A bulk insertOne with no document field inserts its own fields.
		`db.c.bulkWrite([{ insertOne: { doc: 1 } }]); print(db.c.countDocuments({ doc: 1 }))`: "1",
		`print(db.c.bulkWrite([{ insertOne: {} }]).insertedCount)`:                            "1",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// What mongosh refuses is refused here too, by the server where it is the one
// that decides.
func TestIntegration_Writes_InvalidOptionValuesFail(t *testing.T) {
	run, db := setupWrites(t)
	for _, script := range []string{
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { arrayFilters: 5 })`,
		`db.c.deleteOne({ _id: 1 }, { collation: 5 })`,
		`db.c.deleteOne({ _id: 1 }, { hint: 5 })`,
		`db.c.deleteOne({ _id: 1 }, { let: 5 })`,
		`db.c.findOneAndDelete({ _id: 1 }, { projection: "p" })`,
		`db.c.findOneAndUpdate({}, { $set: { z: 1 } }, { returnDocument: "new" })`,
		`db.c.countDocuments({}, { limit: -1 })`,
		`db.c.countDocuments({}, { skip: 1.5 })`,
		`db.c.countDocuments({}, { readConcern: { level: "bogus" } })`,
		`db.c.aggregate([], { readConcern: { level: "bogus" } })`,
		`db.aggregate([{ $documents: [] }], { readConcern: { level: "bogus" } })`,
		`db.c.aggregate([], { batchSize: -1 })`,
		`db.c.aggregate([], { batchSize: "x" })`,
		`db.c.aggregate([], { cursor: 5 })`,
		`db.c.insertOne({ z: 1 }, { writeConcern: { w: -1 } })`,
		`db.c.insertOne({ z: 1 }, { writeConcern: { w: {} } })`,
		`db.c.bulkWrite([{ deleteOne: 5 }])`,
		`db.c.bulkWrite([{ bogusOp: { filter: {} } }])`,
		`db.c.bulkWrite([])`,
		`db.c.bulkWrite([{ insertOne: { document: 5 } }])`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
		assertUnchanged(t, db, script)
	}
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

	// mongosh accepts all of these.
	for _, script := range []string{
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: 5 }).acknowledged)`,
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: 1.5 } }).acknowledged)`,
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { j: 1 } }).acknowledged)`,
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { fsync: true } }).acknowledged)`,
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: 1, wtimeout: 100 } }).acknowledged)`,
		`print(db.c.insertOne({ _id: 5 }, { writeConcern: { w: 1, bogus: 1 } }).acknowledged)`,
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, "true", got, script)
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
		`print(db.c.bulkWrite([{ deleteOne: { filter: { _id: 1 } } }], { writeConcern: { w: 0 } }).acknowledged)`:                          "true",
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

func TestIntegration_Writes_DbAggregateOptions(t *testing.T) {
	run, _ := setupWrites(t)
	for script, want := range map[string]string{
		`print(db.aggregate([{ $documents: [{ t: "ABC" }] }, { $match: { t: "abc" } }], { collation: { locale: "en", strength: 2 } }).length)`: "1",
		`print(db.aggregate({ $documents: [{ t: "ABC" }] }, { $limit: 1 }).length)`:                                                            "1",
		`print(db.aggregate([{ $documents: [{ t: "ABC" }] }], { bogus: 1 }).length)`:                                                           "1",
		`print(db.aggregate([{ $documents: [{ t: "ABC" }] }], { explain: true }).ok)`:                                                          "1",
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// sort on updateOne and replaceOne needs MongoDB 8. The option goes to the server
// either way, so what happens on an older one is for that server to decide.
func TestIntegration_Writes_UpdateSortOnMongo8(t *testing.T) {
	skipBeforeMajor(t, 8)
	run, _ := setupWrites(t)
	got, err := run(`db.c.updateOne({ s: "r" }, { $set: { z: 1 } }, { sort: { p: -1 } }); print(db.c.findOne({ z: 1 })._id)`)
	require.NoError(t, err)
	assert.Equal(t, "2", got)
	got, err = run(`db.c.replaceOne({ s: "r" }, { z: 1 }, { sort: "p" }); print(db.c.findOne({ z: 1 })._id)`)
	require.NoError(t, err)
	assert.Equal(t, "1", got)
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

// maxTimeMS reaches the server, so a write that runs out of time writes nothing,
// including when the time runs out after the write has started.
func TestIntegration_Writes_MaxTimeMSStopsTheWrite(t *testing.T) {
	run, db := setupWrites(t)
	slow := `{ $where: "sleep(300); return true" }`
	for _, script := range []string{
		`db.c.updateOne(` + slow + `, { $set: { z: 1 } }, { maxTimeMS: 50 })`,
		`db.c.updateMany(` + slow + `, { $set: { z: 1 } }, { maxTimeMS: 50 })`,
		`db.c.replaceOne(` + slow + `, { z: 1 }, { maxTimeMS: 50 })`,
		`db.c.deleteOne(` + slow + `, { maxTimeMS: 50 })`,
		`db.c.deleteMany(` + slow + `, { maxTimeMS: 50 })`,
		`db.c.bulkWrite([{ deleteMany: { filter: ` + slow + ` } }], { maxTimeMS: 50 })`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
		assertUnchanged(t, db, script)
	}
	// The server would still be working through the documents after a client-side
	// timeout alone, so wait to see that nothing arrives late.
	_, err := run(`db.c.updateMany({ $where: "sleep(200); return true" }, { $set: { z: 1 } }, { maxTimeMS: 100 })`)
	assert.Error(t, err)
	time.Sleep(time.Second)
	assertUnchanged(t, db, "updateMany, after waiting")

	// As with other options, values that are not numbers are ignored, and bad numbers
	// are refused before anything is written.
	got, err := run(`print(db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { maxTimeMS: "x" }).modifiedCount,
		db.c.deleteOne({ _id: 2 }, { maxTimeMS: 60000 }).deletedCount,
		db.c.insertOne({ _id: 7 }, { maxTimeMS: 60000 }).insertedId,
		db.c.insertMany([{ _id: 8 }], { maxTimeMS: 0 }).acknowledged)`)
	require.NoError(t, err)
	assert.Equal(t, "1 1 7 true", got)
	for _, script := range []string{
		`db.c.updateOne({ _id: 1 }, { $set: { z: 1 } }, { maxTimeMS: -1 })`,
		`db.c.deleteOne({ _id: 1 }, { maxTimeMS: 1.5 })`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
		assertUnchanged(t, db, script)
	}
}

// With several operation types in one entry, mongosh goes by a fixed order, not the
// order they are written in. Going by the first key instead would run the wrong
// operation, so these check what was written as well as the counts.
func TestIntegration_Writes_BulkWritePicksOperationAsMongosh(t *testing.T) {
	run, _ := setupWrites(t)
	state := `print(r.insertedCount, r.modifiedCount, r.deletedCount, db.c.countDocuments());`
	for script, want := range map[string]string{
		`{ deleteOne: { filter: { _id: 1 } }, insertOne: { document: { _id: 9 } } }`:                                                   "1 0 0 4",
		`{ deleteOne: { filter: { _id: 2 } }, replaceOne: { filter: { _id: 1 }, replacement: { r: 1 } } }`:                             "0 1 0 3",
		`{ deleteMany: { filter: {} }, updateOne: { filter: { _id: 1 }, update: { $set: { z: 1 } } } }`:                                "0 1 0 3",
		`{ updateOne: { filter: { _id: 1 }, update: { $set: { z: 1 } } }, replaceOne: { filter: { _id: 1 }, replacement: { r: 1 } } }`: "0 1 0 3",
		`{ updateMany: { filter: {}, update: { $set: { z: 1 } } }, updateOne: { filter: { _id: 1 }, update: { $set: { y: 1 } } } }`:    "0 1 0 3",
		`{ deleteMany: { filter: {} }, updateMany: { filter: {}, update: { $set: { z: 1 } } } }`:                                       "0 3 0 3",
		`{ deleteMany: { filter: {} }, deleteOne: { filter: { _id: 1 } } }`:                                                            "0 0 1 2",
		`{ bogus: {}, deleteOne: { filter: { _id: 1 } } }`:                                                                             "0 0 1 2",
		`{ deleteOne: { filter: { _id: 1 } }, bogus: {} }`:                                                                             "0 0 1 2",
	} {
		got, err := run(`var r = db.c.bulkWrite([` + script + `]); ` + state)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}

	// A type that comes first in the order decides even when its value is unusable.
	for _, script := range []string{
		`{ insertOne: 5, deleteOne: { filter: { _id: 1 } } }`,
		`{ insertOne: null, deleteOne: { filter: { _id: 1 } } }`,
		`{ deleteOne: { filter: { _id: 1 } }, updateOne: null }`,
		`{ replaceOne: { filter: { _id: 1 } }, deleteOne: { filter: { _id: 1 } } }`,
	} {
		got, err := run(`try { db.c.bulkWrite([` + script + `]); print("no error") } catch (e) { print(db.c.countDocuments()) }`)
		require.NoError(t, err, script)
		assert.Equal(t, "3", got, script)
	}
}

// Which version findOneAnd* returns depends on which options are present, not their
// order: returnDocument, then returnOriginal, then returnNewDocument, then new.
func TestIntegration_Writes_ReturnVersionPrecedence(t *testing.T) {
	run, _ := setupWrites(t)
	for options, want := range map[string]string{
		`{ returnDocument: "before", returnNewDocument: true }`: "before",
		`{ returnNewDocument: true, returnDocument: "before" }`: "before",
		`{ returnDocument: "before", returnOriginal: false }`:   "before",
		`{ returnDocument: "after", returnOriginal: true }`:     "after",
		`{ returnDocument: "before", new: true }`:               "before",
		`{ returnNewDocument: true, returnOriginal: true }`:     "before",
		`{ returnOriginal: true, returnNewDocument: true }`:     "before",
		`{ returnNewDocument: false, returnOriginal: false }`:   "after",
		`{ new: true, returnOriginal: true }`:                   "before",
		`{ new: true, returnNewDocument: false }`:               "before",
		`{ returnNewDocument: false, new: true }`:               "before",
		`{ new: false, returnNewDocument: true }`:               "after",
		`{ returnOriginal: 0 }`:                                 "after",
		`{ returnOriginal: null }`:                              "after",
		`{ returnOriginal: "x" }`:                               "before",
		`{ returnNewDocument: 1 }`:                              "after",
		`{ returnNewDocument: null }`:                           "before",
		`{ new: 1 }`:                                            "after",
		`{ new: null }`:                                         "before",
		`{}`:                                                    "before",
	} {
		for _, call := range []string{
			`db.c.findOneAndUpdate({ _id: 1 }, { $set: { z: 1 } }, ` + options + `)`,
			`db.c.findOneAndReplace({ _id: 1 }, { z: 1 }, ` + options + `)`,
		} {
			got, err := run(`print(` + call + `.z === 1 ? "after" : "before")`)
			require.NoError(t, err, call)
			assert.Equal(t, want, got, call)
		}
	}

	// A bad returnDocument throws, even beside a usable spelling.
	for _, options := range []string{
		`{ returnDocument: "new" }`, `{ returnDocument: null }`, `{ returnDocument: true }`,
		`{ returnDocument: "x", returnNewDocument: true }`, `{ returnDocument: undefined, returnOriginal: false }`,
	} {
		_, err := run(`db.c.findOneAndUpdate({ _id: 1 }, { $set: { z: 1 } }, ` + options + `)`)
		assert.Error(t, err, options)
	}

	// findOneAndDelete has no versions to choose between, so these mean nothing.
	got, err := run(`print(db.c.findOneAndDelete({ _id: 1 }, { returnDocument: "x" })._id)`)
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

// The explain option describes the query the other options would run, so they have
// to be part of what is explained.
func TestIntegration_Aggregate_ExplainKeepsOptions(t *testing.T) {
	run, _ := setupWrites(t)
	setup := `db.c.createIndex({ p: 1 }); function plan(o) { return JSON.stringify(db.c.aggregate([{ $match: { p: { $gt: 0 } } }], Object.assign({ explain: "queryPlanner" }, o))); }; `
	for options, want := range map[string]string{
		`{ hint: { p: 1 } }`:                           `"indexName":"p_1"`,
		`{ hint: "_id_" }`:                             `"indexName":"_id_"`,
		`{ collation: { locale: "en", strength: 2 } }`: `"strength":2`,
	} {
		got, err := run(setup + `print(plan(` + options + `))`)
		require.NoError(t, err, options)
		assert.Contains(t, got, want, options)
	}

	for script, want := range map[string]string{
		// These only work if the option reaches the server.
		`print(db.c.aggregate([{ $match: { $expr: { $eq: ["$_id", "$$v"] } } }], { explain: true, let: { v: 1 } }).ok)`: "1",
		`print(db.aggregate([{ $documents: [{ a: "$$v" }] }], { explain: true, let: { v: 1 } }).ok)`:                    "1",
		// An explain has no read concern, so mongosh does not look at it.
		`print(db.c.aggregate([], { explain: true, readConcern: { level: "bogus" } }).ok)`: "1",
		`print(db.c.aggregate([], { explain: true, batchSize: 1 }).ok)`:                    "1",
		`print(db.c.aggregate([], { explain: true, cursor: { batchSize: 1 } }).ok)`:        "1",
		`print(db.c.aggregate([], { explain: true, allowDiskUse: 1 }).ok)`:                 "1",
		// Which plan each form asks for.
		`print(!!db.c.aggregate([], { explain: true }).executionStats)`:  "true",
		`print(!!db.c.aggregate([], { explain: false }).executionStats)`: "false",
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}

	for _, script := range []string{
		`db.c.aggregate([{ $match: { p: 1 } }], { explain: true, hint: "nope" })`,
		`db.c.aggregate([{ $match: { p: 1 } }], { explain: true, hint: 5 })`,
		`db.c.aggregate([{ $match: { p: 1 } }], { explain: true, collation: { locale: "nope" } })`,
		`db.c.aggregate([{ $match: { p: 1 } }], { explain: true, let: 5 })`,
		`db.c.aggregate([{ $match: { $expr: { $eq: ["$_id", "$$v"] } } }], { explain: true })`,
		`db.c.aggregate([{ $match: { $where: "sleep(300); return true" } }], { explain: "executionStats", maxTimeMS: 50 })`,
		`db.c.aggregate([], { explain: true, batchSize: -1 })`,
		`db.c.aggregate([], { explain: true, readPreference: "bogus" })`,
		`db.c.aggregate([], { explain: null })`,
		`db.c.aggregate([], { explain: 1 })`,
		`db.c.aggregate([], { explain: {} })`,
		`db.c.aggregate([], { explain: "bogus" })`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
	}
}
