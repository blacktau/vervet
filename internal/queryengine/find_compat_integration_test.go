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
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"vervet/internal/models"
)

// setupFinds runs scripts against three documents and an index on p. The expected
// values below are what mongosh gave for the same scripts.
func setupFinds(t *testing.T) func(script string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db := dbName(t)
	t.Cleanup(func() { testClient.Database(db).Drop(context.Background()) })

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `
		db.c.insertMany([{ _id: 1, s: "a", p: 1 }, { _id: 2, s: "b", p: 9 }, { _id: 3, s: "A", p: 5 }]);
		db.c.createIndex({ p: 1 });
		db.createCollection("cap", { capped: true, size: 100000 });
		db.cap.insertOne({ _id: 1 });
	`)
	require.NoError(t, err)
	return func(script string) (string, error) {
		result, err := engine.ExecuteQuery(ctx, testURI, db, `
			function ids(c) { return c.toArray().map(d => d._id).join() }
			`+script)
		return printedText(result), err
	}
}

func TestIntegration_Find_MatchesMongosh(t *testing.T) {
	run := setupFinds(t)
	for script, want := range map[string]string{
		// Filters: absent, or an ObjectId meaning that _id.
		`print(ids(db.c.find()))`:           "1,2,3",
		`print(ids(db.c.find(undefined)))`:  "1,2,3",
		`print(ids(db.c.find(ObjectId())))`: "",
		`print(db.c.findOne(ObjectId()))`:   "null",
		`var id = db.c.insertOne({ n: 7 }).insertedId; var n = db.c.findOne(id).n; db.c.deleteOne({ _id: id }); print(n)`: "7",
		// Options on find and findOne.
		`print(ids(db.c.find({}, {}, { sort: { p: -1 } })))`:                                                                 "2,3,1",
		`print(ids(db.c.find({}, {}, { sort: "p" })))`:                                                                       "1,3,2",
		`print(ids(db.c.find({}, {}, { sort: ["p", -1] })))`:                                                                 "2,3,1",
		`print(db.c.findOne({}, {}, { sort: [["p", -1]] })._id)`:                                                             "2",
		`print(ids(db.c.find({}, {}, { sort: null })))`:                                                                      "1,2,3",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, limit: 1 })))`:                                                      "1",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, limit: -1 })))`:                                                     "1",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, limit: -2.5 })))`:                                                   "1,2",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, limit: 1.5 })))`:                                                    "1",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, limit: "1" })))`:                                                    "1,2,3",
		`print(ids(db.c.find({}, {}, { sort: { _id: 1 }, skip: 2 })))`:                                                       "3",
		`print(db.c.findOne({}, {}, { sort: { _id: 1 }, skip: 1.5 })._id)`:                                                   "2",
		`print(db.c.findOne({}, {}, { sort: { _id: 1 }, skip: "1" })._id)`:                                                   "1",
		`print(db.c.find({ _id: 1 }, undefined, { projection: { p: 1, _id: 0 } }).toArray()[0].p)`:                           "1",
		`print(ids(db.c.find({ s: "B" }, {}, { collation: { locale: "en", strength: 2 } })))`:                                "2",
		`print(db.c.findOne({ _id: 1 }, {}, { collation: 5 })._id)`:                                                          "1",
		`print(ids(db.c.find({}, {}, { min: { p: 5 }, max: { p: 100 }, hint: { p: 1 } })))`:                                  "3,2",
		`print(db.c.findOne({}, {}, { min: { p: 5 }, max: { p: 100 }, hint: { p: 1 } })._id)`:                                "3",
		`print(ids(db.c.find({ $expr: { $eq: ["$_id", "$$x"] } }, {}, { let: { x: 2 } })))`:                                  "2",
		`print(JSON.stringify(db.c.findOne({ _id: 1 }, {}, { returnKey: true })))`:                                           `{"_id":1}`,
		`print(Object.keys(db.c.findOne({ _id: 1 }, {}, { showRecordId: true })).join())`:                                    "_id,s,p,$recordId",
		`print(Object.keys(db.c.findOne({ _id: 1 }, {}, { showRecordId: 1 })).join())`:                                       "_id,s,p",
		`print(db.c.findOne({ _id: 1 }, {}, { readConcern: { level: "local" }, readPreference: "secondaryPreferred" })._id)`: "1",
		`print(ids(db.c.find({ _id: 1 }, {}, { readConcern: { level: "local" } })))`:                                         "1",
		// Options it ignores.
		`print(db.c.findOne({ _id: 1 }, {}, { bogus: 1, writeConcern: { w: 2 }, comment: { a: 1 }, allowDiskUse: true, maxTimeMS: "5" })._id)`: "1",
		`print(db.c.findOne({ _id: 1 }, {}, []). _id)`:                        "1",
		`print(ids(db.c.find({ _id: 1 }, {}, { bogus: 1, batchSize: "x" })))`: "1",
		// explain gives the plan, even when false.
		`print(db.c.findOne({ _id: 1 }, {}, { explain: true }).ok)`:        "1",
		`print(db.c.findOne({ _id: 1 }, {}, { explain: false }).ok)`:       "1",
		`print(db.c.find({ _id: 1 }, {}, { explain: "queryPlanner" }).ok)`: "1",
		// Tailable reads what is there.
		`var c = db.cap.find({}, {}, { tailable: true }); print(c.next()._id); c.close()`: "1",
		`var c = db.cap.find().tailable(); print(c.next()._id); c.close()`:                "1",
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

func TestIntegration_Find_RefusesWhatMongoshRefuses(t *testing.T) {
	run := setupFinds(t)
	for _, script := range []string{
		`db.c.find(null)`, `db.c.find("x")`, `db.c.find(5)`, `db.c.find([])`, `db.c.find(true)`,
		`db.c.find(new Date())`, `db.c.find(/x/)`, `db.c.find(UUID())`, `db.c.find(NumberLong("1"))`,
		`db.c.findOne(null)`, `db.c.findOne("x")`, `db.c.findOne(5)`, `db.c.findOne([])`,
		`db.c.find({}, "p")`, `db.c.find({}, 5)`, `db.c.findOne({}, "p")`, `db.c.findOne({}, 5)`,
		`db.c.find({}, undefined, { projection: 5 })`,
		`db.c.find({}, {}, 5)`, `db.c.find({}, {}, "x")`, `db.c.findOne({}, {}, 5)`, `db.c.findOne({}, {}, "x")`,
		`db.c.find({}, {}, { sort: 5 })`, `db.c.find({}, {}, { sort: true })`,
		`db.c.find({}, {}, { skip: -1 })`,
		`db.c.findOne({}, {}, { hint: 5 })`, `db.c.findOne({}, {}, { hint: "nope" })`,
		`db.c.find({}, {}, { hint: "nope" }).toArray()`,
		`db.c.findOne({}, {}, { maxTimeMS: -1 })`, `db.c.find({}, {}, { maxTimeMS: 1.5 })`,
		`db.c.findOne({}, {}, { readConcern: { level: "bogus" } })`,
		`db.c.find({}, {}, { readConcern: { level: "bogus" } })`,
		`db.c.findOne({}, {}, { readPreference: "bogus" })`,
		`db.c.findOne({}, {}, { min: { p: 5 } })`, `db.c.find({}, {}, { min: 5 })`,
		`db.c.findOne({}, {}, { let: 5 })`,
		`db.c.findOne({ _id: 1 }, {}, { tailable: true })`,
		`db.c.find({ _id: 1 }, {}, { tailable: true }).toArray()`,
		// Cursor methods.
		`db.c.find().limit()`, `db.c.find().limit(null)`, `db.c.find().limit("1")`,
		`db.c.find().limit(NumberInt(1))`, `db.c.find().limit(NumberLong("1"))`,
		`db.c.find().skip()`, `db.c.find().skip("1")`, `db.c.find().skip(-1)`,
		`db.c.find().sort(5)`, `db.c.find().hint(5)`, `db.c.find().hint("nope").toArray()`,
		`db.c.find().maxTimeMS("5")`, `db.c.find().maxTimeMS(-1)`, `db.c.find().batchSize("x")`,
		`db.c.find().hint({ p: 1 }).min(5)`, `db.c.find().projection(5)`,
		`db.c.find().readConcern("bogus")`, `db.c.find().readConcern({ level: "local" })`,
		`db.c.find().readPref("bogus")`, `db.c.find().allowDiskUse()`,
		`db.c.find().addOption(0)`, `db.c.find().addOption(4)`, `db.c.find().addOption(16)`, `db.c.find().addOption(32)`,
		`db.c.find({ _id: 1 }).addOption(2).toArray()`, `db.c.find({ _id: 1 }).tailable().toArray()`,
		`var c = db.c.find(); c.next(); c.limit(1)`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
	}
}

func TestIntegration_Cursor_MatchesMongosh(t *testing.T) {
	run := setupFinds(t)
	for script, want := range map[string]string{
		`print(ids(db.c.find().sort("p")))`:                                                     "1,3,2",
		`print(ids(db.c.find().sort("p", -1)))`:                                                 "1,3,2", // the second argument does nothing
		`print(ids(db.c.find().sort([["p", -1]])))`:                                             "2,3,1",
		`print(ids(db.c.find().sort()))`:                                                        "1,2,3",
		`print(ids(db.c.find().sort(null)))`:                                                    "1,2,3",
		`print(ids(db.c.find().sort({ _id: 1 }).limit(-1)))`:                                    "1",
		`print(ids(db.c.find().sort({ _id: 1 }).limit(0)))`:                                     "1,2,3",
		`print(ids(db.c.find().sort({ _id: 1 }).limit(1.5)))`:                                   "1",
		`print(ids(db.c.find().sort({ _id: 1 }).skip(1.5)))`:                                    "2,3",
		`print(ids(db.c.find().hint({ p: 1 })))`:                                                "1,3,2",
		`print(ids(db.c.find().hint("p_1")))`:                                                   "1,3,2",
		`print(ids(db.c.find().hint()))`:                                                        "1,2,3",
		`print(ids(db.c.find({ s: "B" }).collation({ locale: "en", strength: 2 })))`:            "2",
		`print(ids(db.c.find({ _id: 1 }).collation(5)))`:                                        "1",
		`print(ids(db.c.find({ _id: 1 }).collation()))`:                                         "1",
		`print(ids(db.c.find({ _id: 1 }).comment(5)))`:                                          "1",
		`print(ids(db.c.find({ _id: 1 }).comment({ a: 1 })))`:                                   "1",
		`print(ids(db.c.find({ _id: 1 }).comment()))`:                                           "1",
		`print(ids(db.c.find({ _id: 1 }).batchSize(-1)))`:                                       "1",
		`print(ids(db.c.find().hint({ p: 1 }).min({ p: 5 }).max({ p: 100 })))`:                  "3,2",
		`print(JSON.stringify(db.c.find({ _id: 1 }).returnKey(true).toArray()))`:                `[{"_id":1}]`,
		`print(Object.keys(db.c.find({ _id: 1 }).returnKey().toArray()[0]).join())`:             "_id,s,p",
		`print(Object.keys(db.c.find({ _id: 1 }).showRecordId().toArray()[0]).join())`:          "_id,s,p,$recordId",
		`print(Object.keys(db.c.find({ _id: 1 }).showRecordId(false).toArray()[0]).join())`:     "_id,s,p,$recordId",
		`print(JSON.stringify(db.c.find({ _id: 1 }).projection({ p: 1, _id: 0 }).toArray()))`:   `[{"p":1}]`,
		`print(Object.keys(db.c.find({ _id: 1 }).projection().toArray()[0]).join())`:            "_id,s,p",
		`print(ids(db.c.find({ _id: 1 }).readConcern("local").readPref("secondaryPreferred")))`: "1",
		`print(ids(db.c.find({ _id: 1 }).sort({ p: 1 }).allowDiskUse()))`:                       "1",
		`print(ids(db.c.find({ _id: 1 }).sort({ p: 1 }).allowDiskUse(false)))`:                  "1",
		`print(ids(db.c.find({ _id: 1 }).allowPartialResults().noCursorTimeout().pretty()))`:    "1",
		`print(ids(db.c.find({ _id: 1 }).addOption(8).addOption(64).addOption(128)))`:           "1",
		// Counting.
		`print(db.c.find().count(), db.c.find({ _id: 1 }).count())`:                                           "3 1",
		`print(db.c.find().limit(1).count(), db.c.find().skip(1).count())`:                                    "1 2",
		`print(db.c.find().limit(1).size(), db.c.find().skip(1).size(), db.c.find().skip(1).limit(1).size())`: "1 2 1",
		`print(db.c.find().itcount())`:                                                                        "3",
		`var c = db.c.find(); c.next(); print(c.itcount(), c.itcount())`:                                      "2 0",
		// Reading.
		`var c = db.c.find({ _id: 1 }); print(c.next()._id, c.next(), c.tryNext())`:            "1 null null",
		`print(db.c.find().sort({ _id: 1 }).tryNext()._id)`:                                    "1",
		`var c = db.c.find().sort({ _id: 1 }); c.next(); print(ids(c), c.toArray().length)`:    "2,3 0",
		`var n = 0; db.c.find().forEach(d => { n++; return false }); print(n)`:                 "1",
		`var n = 0; for (const d of db.c.find()) { n++ } print(n)`:                             "3",
		`print(db.c.find().sort({ _id: 1 }).map(d => d._id).map(x => x + 1).toArray().join())`: "2,3,4",
		`print(db.c.find().sort({ _id: 1 }).map(d => d._id).next())`:                           "1",
		// State.
		`var c = db.c.find(); print(c.isExhausted(), c.isClosed(), typeof c.objsLeftInBatch())`:                 "false false number",
		`var c = db.c.find(); c.next(); print(c.isExhausted(), c.objsLeftInBatch())`:                            "false 2",
		`var c = db.c.find({ _id: 1 }); c.toArray(); print(c.isExhausted(), c.isClosed())`:                      "true true",
		`var c = db.c.find(); c.close(); print(c.isClosed(), c.isExhausted(), c.hasNext(), c.toArray().length)`: "true true false 0",
		`print(db.c.find({ _id: 1 }).explain().ok)`:                                                             "1",
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// The editor suggests these, from completionData.ts. Keep the two in step: a method
// removed here would otherwise go on being suggested and fail when run.
func TestIntegration_SuggestedCursorMethodsExist(t *testing.T) {
	run := setupFinds(t)
	cursorMethods := []string{"limit", "skip", "sort", "toArray", "count", "forEach", "pretty", "explain", "hint", "batchSize", "maxTimeMS", "collation", "comment", "map", "hasNext", "next", "tryNext", "projection", "min", "max", "returnKey", "showRecordId", "readConcern", "readPref", "allowDiskUse", "allowPartialResults", "noCursorTimeout", "tailable", "addOption", "size", "itcount", "objsLeftInBatch", "isExhausted", "isClosed", "close"}
	arrayCursorMethods := []string{"close", "forEach", "hasNext", "isClosed", "isExhausted", "itcount", "map", "next", "objsLeftInBatch", "pretty", "toArray", "tryNext"}
	for expr, methods := range map[string][]string{
		`db.c.find()`:                                       cursorMethods,
		`db.c.aggregate([])`:                                arrayCursorMethods,
		`db.c.listIndexes()`:                                arrayCursorMethods,
		`db.getCollectionInfos()`:                           arrayCursorMethods,
		`db.c.find().map(function (d) { return d })`:        arrayCursorMethods,
		`db.c.aggregate([]).map(function (d) { return d })`: arrayCursorMethods,
		`db.c.aggregate([]).pretty()`:                       arrayCursorMethods,
	} {
		for _, method := range methods {
			script := `print(typeof (` + expr + `).` + method + `)`
			got, err := run(script)
			require.NoError(t, err, script)
			assert.Equal(t, "function", got, script)
		}
	}
}

// aggregate reads eagerly here, so its result is an array that also works as the
// cursor mongosh returns.
func TestIntegration_AggregateCursorMethods(t *testing.T) {
	run := setupFinds(t)
	agg := `db.c.aggregate([{ $sort: { _id: 1 } }])`
	for script, want := range map[string]string{
		`var c = ` + agg + `; print(c.hasNext(), c.next()._id, c.tryNext()._id, c.objsLeftInBatch())`:  "true 1 2 1",
		`var c = ` + agg + `; c.next(); print(c.toArray().map(d => d._id).join(), c.toArray().length)`: "2,3 0",
		`var c = ` + agg + `; c.next(); print(c.itcount(), c.itcount())`:                               "2 0",
		`var c = ` + agg + `; c.toArray(); print(c.next(), c.isExhausted(), c.hasNext())`:              "null true false",
		`var c = ` + agg + `; c.close(); print(c.isClosed(), c.hasNext(), c.next())`:                   "true false null",
		`var n = 0; ` + agg + `.forEach(d => { n++; return false }); print(n)`:                         "1",
		`print(` + agg + `.pretty().toArray().length)`:                                                 "3",
		// It is still usable as the array it always was.
		`var c = ` + agg + `; print(c.length, c[1]._id, c.map(d => d._id).join(), Object.keys(c).join())`: "3 2 1,2,3 0,1,2",
	} {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// The settings a query runs with must apply to every page and to its count, or
// later pages would show different documents from the first.
func TestIntegration_Paging_KeepsFindSettings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(context.Background())

	engine := NewGojaEngine(testClient, 2, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `
		var docs = [];
		for (var i = 0; i < 7; i++) { docs.push({ _id: i, p: i, t: i % 2 ? "X" : "x" }); }
		db.c.insertMany(docs);
		db.c.createIndex({ p: 1 });
	`)
	require.NoError(t, err)

	page := func(script string, n int64) (models.QueryResult, models.PageContext) {
		first, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err, script)
		require.NotNil(t, first.PageContext, script)
		// The PageContext goes to the frontend and back as JSON.
		data, err := json.Marshal(first.PageContext)
		require.NoError(t, err, script)
		var pc models.PageContext
		require.NoError(t, json.Unmarshal(data, &pc), script)
		next, err := engine.FetchPage(ctx, pc, n, 2)
		require.NoError(t, err, script)
		return next, pc
	}

	for _, tc := range []struct {
		name, script string
		want         int
		count        int64
	}{
		{"options argument", `db.c.find({}, {}, { sort: { _id: -1 }, skip: 1, limit: 4 })`, 2, 4},
		{"min and max", `db.c.find().hint({ p: 1 }).min({ p: 2 }).max({ p: 6 })`, 2, 4},
		{"let", `db.c.find({ $expr: { $gte: ["$p", "$$x"] } }, {}, { let: { x: 4 } })`, 1, 3},
		{"collation", `db.c.find({ t: "x" }, {}, { collation: { locale: "en", strength: 2 } })`, 2, 7},
		{"object comment", `db.c.find({}, {}, { comment: { a: 1 } })`, 2, 7},
		{"read settings", `db.c.find({}, {}, { readConcern: { level: "local" }, readPreference: "primary" })`, 2, 7},
	} {
		next, pc := page(tc.script, 1)
		assert.Len(t, next.Documents, tc.want, tc.name)
		count, _, err := engine.CountForPage(ctx, pc)
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.count, count, tc.name)
	}

	// Only documents 0 and 3 have q, so a sparse index on it holds just those two. A
	// script's createIndex can't make one yet (#331).
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.h.insertMany([
		{ _id: 0, q: 1, t: "x" }, { _id: 1, t: "x" }, { _id: 2, t: "X" }, { _id: 3, q: 2, t: "X" }, { _id: 4, t: "y" },
	])`)
	require.NoError(t, err)
	_, err = testClient.Database(db).Collection("h").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "q", Value: 1}}, Options: options.Index().SetSparse(true).SetName("sparse_q"),
	})
	require.NoError(t, err)
	for _, tc := range []struct {
		script string
		want   int
		count  int64
	}{
		{`db.h.find().hint({ q: 1 })`, 0, 2},
		{`db.h.find().hint("sparse_q")`, 0, 2},
		{`db.h.find({}, {}, { hint: { q: 1 } })`, 0, 2},
		{`db.h.find({ t: "x" }).hint({ q: 1 })`, 0, 1},
		{`db.h.find({ t: "x" }).hint({ q: 1 }).collation({ locale: "en", strength: 2 })`, 0, 2},
		{`db.h.find().hint({ q: 1 }).skip(1)`, 0, 1},
		{`db.h.find().hint({ q: 1 }).limit(1)`, 0, 1},
		{`db.h.find().hint({ _id: 1 })`, 2, 5},
		{`db.h.find().hint({ $natural: 1 })`, 2, 5},
		{`db.h.find().hint({ $natural: -1 })`, 2, 5},
	} {
		next, pc := page(tc.script, 1)
		assert.Len(t, next.Documents, tc.want, tc.script)
		count, estimated, err := engine.CountForPage(ctx, pc)
		require.NoError(t, err, tc.script)
		assert.False(t, estimated, tc.script)
		assert.Equal(t, tc.count, count, tc.script)
	}

	next, _ := page(`db.c.find().returnKey(true).sort({ _id: 1 })`, 1)
	require.Len(t, next.Documents, 2)
	assert.JSONEq(t, `[{"_id":{"$numberInt":"2"}},{"_id":{"$numberInt":"3"}}]`, resultText(next))
}

// createCollection passes its options to the server rather than dropping them.
func TestIntegration_CreateCollectionOptions(t *testing.T) {
	run := setupFinds(t)
	got, err := run(`print(JSON.stringify(db.createCollection("made", { capped: true, size: 4096 })));
		print(db.made.isCapped(), db.getCollectionInfos({ name: "made" })[0].options.capped);
		print(db.createCollection("plain").ok, db.getCollectionInfos({ name: "plain" })[0].options.capped)`)
	require.NoError(t, err)
	assert.Equal(t, "{\"ok\":1}\ntrue true\n1 undefined", got)

	_, err = run(`db.createCollection("bad", { capped: "x", size: "y" })`)
	assert.Error(t, err)
}

// A mapped or closed cursor left as the final value still displays.
func TestIntegration_Find_FinalValues(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(context.Background())
	engine := NewGojaEngine(testClient, 25, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertMany([{ _id: 1 }, { _id: 2 }])`)
	require.NoError(t, err)

	for script, want := range map[string]int{
		`db.c.find()`:                          2,
		`db.c.find().map(d => ({ x: d._id }))`: 2,
		`db.c.aggregate([{ $match: {} }])`:     2,
		`var c = db.c.find(); c.close(); c`:    0,
		`db.c.findOne({ _id: 1 })`:             1,
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err, script)
		assert.Len(t, result.Documents, want, script)
	}
}
