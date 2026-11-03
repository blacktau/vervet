//go:build integration

package queryengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Without a usable query, findAndModify must fail and change nothing. Each of these
// used to change or delete the first document.
func TestIntegration_FindAndModify_BadSpecsChangeNothing(t *testing.T) {
	run, db := setupWrites(t)
	for _, script := range []string{
		`db.c.findAndModify({ remove: true })`,
		`db.c.findAndModify({ query: undefined, remove: true })`,
		`db.c.findAndModify({ query: null, remove: true })`,
		`db.c.findAndModify({ query: "x", remove: true })`,
		`db.c.findAndModify({ query: 5, remove: true })`,
		`db.c.findAndModify({ query: true, remove: true })`,
		`db.c.findAndModify({ query: [], remove: true })`,
		`db.c.findAndModify({ query: /x/, remove: true })`,
		`db.c.findAndModify({ query: ObjectId(), remove: true })`,
		`db.c.findAndModify({ update: { $set: { z: 1 } } })`,
		`db.c.findAndModify({ query: "x", update: { $set: { z: 1 } } })`,
		`db.c.findAndModify({ query: null, update: { $set: { z: 1 } } })`,
		`db.c.findAndModify({ query: ObjectId(), update: { $set: { z: 1 } } })`,
		// Valid queries, but something else in the spec is wrong.
		`db.c.findAndModify()`,
		`db.c.findAndModify(5)`,
		`db.c.findAndModify({})`,
		`db.c.findAndModify({ query: {} })`,
		`db.c.findAndModify({ query: {}, remove: false })`,
		`db.c.findAndModify({ query: {}, update: null })`,
		`db.c.findAndModify({ query: {}, update: 5 })`,
		`db.c.findAndModify({ query: {}, fields: { p: 1 }, projection: { s: 1 }, remove: true })`,
		`db.c.findAndModify({ query: {}, fields: 0, projection: { s: 1 }, remove: true })`,
		`db.c.findAndModify({ query: {}, fields: 5, remove: true })`,
		`db.c.findAndModify({ query: {}, sort: 5, remove: true })`,
		`db.c.findAndModify({ query: {}, hint: 5, remove: true })`,
		`db.c.findAndModify({ query: {}, hint: "nope", remove: true })`,
		`db.c.findAndModify({ query: {}, maxTimeMS: "x", remove: true })`,
		`db.c.findAndModify({ query: {}, maxTimeMS: -1, remove: true })`,
		`db.c.findAndModify({ query: {}, collation: 5, remove: true })`,
		`db.c.findAndModify({ query: {}, arrayFilters: 5, update: { $set: { z: 1 } } })`,
		`db.c.findAndModify({ query: {}, returnDocument: "bad", update: { $set: { z: 1 } } })`,
		// The findOneAnd* methods refuse this too.
		`db.c.findOneAndDelete({}, { maxTimeMS: "x" })`,
		`db.c.findOneAndUpdate({}, { $set: { z: 1 } }, { maxTimeMS: "x" })`,
		`db.c.findOneAndReplace({}, { z: 1 }, { maxTimeMS: "x" })`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
		assertUnchanged(t, db, script)
	}
}

func TestIntegration_FindAndModify_MatchesMongosh(t *testing.T) {
	run, _ := setupWrites(t)
	all := `db.c.find().sort({ _id: 1 }).toArray().map(d => d._id).join()`
	cases := map[string]string{
		// A truthy remove deletes; a falsy one leaves the update to run.
		`print(db.c.findAndModify({ query: { _id: 2 }, remove: 1 })._id, ` + all + `)`:                            "2 1,3",
		`print(db.c.findAndModify({ query: { _id: 2 }, remove: "y" })._id, ` + all + `)`:                          "2 1,3",
		`db.c.findAndModify({ query: { _id: 2 }, remove: 0, update: { $set: { m: 1 } } }); print(` + all + `)`:    "1,2,3",
		`db.c.findAndModify({ query: { _id: 2 }, remove: true, update: { $set: { m: 1 } } }); print(` + all + `)`: "1,3",
		`print(db.c.findAndModify({ query: {}, sort: { p: -1 }, remove: true })._id)`:                             "2",
		`print(db.c.findAndModify({ query: { _id: 99 }, remove: true }))`:                                         "null",
		`print(db.c.findAndModify({ query: { _id: 99 }, update: { $set: { m: 1 } } }))`:                           "null",
		// Which version of the document comes back.
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } } }).m)`:                                      "undefined",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } }, new: true }).m)`:                           "1",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } }, new: 1 }).m)`:                              "1",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } }, new: 0 }).m)`:                              "undefined",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } }, returnDocument: "after" }).m)`:             "1",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: { $set: { m: 1 } }, new: true, returnDocument: "before" }).m)`: "undefined",
		// A replacement document, and a pipeline.
		`print(Object.keys(db.c.findAndModify({ query: { _id: 2 }, update: { m: 1 }, new: true })).join())`:   "_id,m",
		`print(Object.keys(db.c.findAndModify({ query: { _id: 2 }, update: {}, new: true })).join())`:         "_id",
		`print(db.c.findAndModify({ query: { _id: 2 }, update: [{ $set: { m: 1 } }], new: true }).m)`:         "1",
		`print(db.c.findAndModify({ query: { _id: 99 }, update: { m: 1 }, upsert: true, new: true }).m)`:      "1",
		`print(db.c.findAndModify({ query: { _id: 99 }, update: { $set: { m: 1 } }, upsert: 1, new: true }))`: "null",
		// Options that used to be dropped.
		`print(Object.keys(db.c.findAndModify({ query: { _id: 2 }, fields: { p: 1 }, remove: true })).join())`:                                  "_id,p",
		`print(Object.keys(db.c.findAndModify({ query: { _id: 2 }, projection: { p: 1 }, update: { $set: { m: 1 } } })).join())`:                "_id,p",
		`print(db.c.findAndModify({ query: { s: "R" }, remove: true }))`:                                                                        "null",
		`print(db.c.findAndModify({ query: { s: "R" }, collation: { locale: "en", strength: 2 }, sort: { _id: 1 }, remove: true })._id)`:        "1",
		`print(db.c.findAndModify({ query: { _id: 1 }, arrayFilters: [{ "e.k": 2 }], update: { $set: { "a.$[e].z": 1 } }, new: true }).a[1].z)`: "1",
		`print(db.c.findAndModify({ query: { $expr: { $eq: ["$p", "$$v"] } }, let: { v: 9 }, remove: true })._id)`:                              "2",
		`print(db.c.findAndModify({ query: { _id: 2 }, hint: "_id_", remove: true })._id)`:                                                      "2",
		`print(db.c.findAndModify({ query: { _id: 2 }, maxTimeMS: 60000, writeConcern: { w: 1 }, comment: "c", remove: true })._id)`:            "2",
		`print(db.c.findAndModify({ query: { _id: 2 }, bogus: 1, remove: true })._id)`:                                                          "2",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// indexField is a script printing a field of the only index besides _id_.
func indexField(field string) string {
	return ` print(JSON.stringify(db.c.getIndexes()[1]["` + field + `"]))`
}

func TestIntegration_CreateIndex_AppliesOptions(t *testing.T) {
	run, _ := setupWrites(t)
	cases := map[string]string{
		`db.c.createIndex({ p: 1 }, { unique: true });` + indexField("unique"):                                                "true",
		`db.c.createIndex({ p: 1 }, { unique: 1 });` + indexField("unique"):                                                   "true",
		`db.c.createIndex({ p: 1 }, { name: "nm" });` + indexField("name"):                                                    `"nm"`,
		`db.c.createIndex({ p: 1 }, { name: null });` + indexField("name"):                                                    `"p_1"`,
		`db.c.createIndex({ p: 1, s: -1 });` + indexField("name"):                                                             `"p_1_s_-1"`,
		`db.c.createIndex({ p: 1 }, { sparse: true });` + indexField("sparse"):                                                "true",
		`db.c.createIndex({ p: 1 }, { expireAfterSeconds: 60 });` + indexField("expireAfterSeconds"):                          "60",
		`db.c.createIndex({ p: 1 }, { partialFilterExpression: { p: { $gt: 1 } } });` + indexField("partialFilterExpression"): `{"p":{"$gt":1}}`,
		`db.c.createIndex({ p: 1 }, { hidden: true });` + indexField("hidden"):                                                "true",
		`db.c.createIndex({ p: 1 }, { collation: { locale: "en" } }); print(db.c.getIndexes()[1].collation.locale)`:           "en",
		`db.c.createIndex({ s: "text" }, { weights: { s: 5 } }); print(db.c.getIndexes()[1].weights.s)`:                       "5",
		`db.c.createIndex({ p: 1 }, { version: 1 });` + indexField("v"):                                                       "1",
		// Ignored, as they are in mongosh.
		`db.c.createIndex({ p: 1 }, { v: 1 });` + indexField("v"):                                                 "2",
		`db.c.createIndex({ p: 1 }, { bogus: 1, key: { z: 1 }, ns: "x.y", maxTimeMS: -1 });` + indexField("name"): `"p_1"`,
		`db.c.createIndex({ p: 1 }, null);` + indexField("name"):                                                  `"p_1"`,
		`db.c.createIndex({ p: 1 }, ObjectId());` + indexField("name"):                                            `"p_1"`,
		`db.c.createIndex({ p: 1 }, {}, null);` + indexField("name"):                                              `"p_1"`,
		`db.c.createIndex({ p: 1 }, { writeConcern: { w: 1, bogus: 1 }, comment: "c" });` + indexField("name"):    `"p_1"`,
		// Every index gets the same options.
		`db.c.createIndexes([{ p: 1 }, { s: 1 }], { sparse: true }); print(db.c.getIndexes().filter(i => i.sparse).length)`: "2",
		`db.c.createIndexes([{ p: 1 }], { name: "nm" });` + indexField("name"):                                              `"nm"`,
		`db.c.createIndexes([{ p: 1 }], { unique: true });` + indexField("unique"):                                          "true",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}

// A unique index must be unique: this is the failure that lost constraints.
func TestIntegration_CreateIndex_UniqueIsEnforced(t *testing.T) {
	run, _ := setupWrites(t)
	_, err := run(`db.c.createIndex({ p: 1 }, { unique: true }); db.c.insertOne({ p: 100 }); db.c.insertOne({ p: 100 })`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E11000")
}

// What the server refuses must reach the script, with no index left behind.
func TestIntegration_CreateIndex_Errors(t *testing.T) {
	run, db := setupWrites(t)
	for _, script := range []string{
		// Over existing duplicates, and values only the server can judge.
		`db.c.createIndex({ s: 1 }, { unique: true })`,
		`db.c.createIndex({ p: 1 }, { unique: "x" })`,
		`db.c.createIndex({ p: 1 }, { unique: null })`,
		`db.c.createIndex({ p: 1 }, { unique: undefined })`,
		`db.c.createIndex({ p: 1 }, { name: 5 })`,
		`db.c.createIndex({ p: 1 }, { name: "" })`,
		`db.c.createIndex({ p: 1 }, { expireAfterSeconds: "x" })`,
		`db.c.createIndex({ p: 1 }, { collation: 5 })`,
		`db.c.createIndex({ p: 1 }, { collation: { locale: 5 } })`,
		`db.c.createIndex({ p: 1 }, { writeConcern: { w: 5 } })`,
		`db.c.createIndexes([{ p: 1 }, { s: 1 }], { name: "same" })`,
		// Options that aren't an object.
		`db.c.createIndex({ p: 1 }, 5)`,
		`db.c.createIndex({ p: 1 }, "x")`,
		`db.c.createIndex({ p: 1 }, true)`,
		`db.c.createIndex({ p: 1 }, 0)`,
		`db.c.createIndex({ p: 1 }, [])`,
		`db.c.createIndexes([{ p: 1 }], 5)`,
		`db.c.createIndexes([{ p: 1 }], [])`,
		// Keys.
		`db.c.createIndex()`,
		`db.c.createIndex(null)`,
		`db.c.createIndex("p")`,
		`db.c.createIndex({})`,
		`db.c.createIndex({ p: "bad" })`,
		`db.c.createIndexes({ p: 1 })`,
		`db.c.createIndexes([])`,
		`db.c.createIndexes([{}])`,
		`db.c.createIndexes([5])`,
	} {
		_, err := run(script)
		assert.Error(t, err, script)
		reply, err := testClient.Database(db).RunCommand(context.Background(), map[string]any{"listIndexes": "c"}).Raw()
		require.NoError(t, err, script)
		first, err := reply.LookupErr("cursor", "firstBatch")
		require.NoError(t, err, script)
		indexes, err := first.Array().Values()
		require.NoError(t, err, script)
		assert.Len(t, indexes, 1, script)
	}
}

// A test server is a standalone, which refuses any commitQuorum.
func TestIntegration_CreateIndex_SendsCommitQuorum(t *testing.T) {
	run, _ := setupWrites(t)
	for _, script := range []string{
		`db.c.createIndex({ p: 1 }, {}, "majority")`,
		`db.c.createIndex({ p: 1 }, {}, 1)`,
		`db.c.createIndex({ p: 1 }, { commitQuorum: "majority" })`,
		`db.c.createIndexes([{ p: 1 }], {}, "majority")`,
	} {
		_, err := run(script)
		require.Error(t, err, script)
		assert.Contains(t, err.Error(), "commitQuorum", script)
	}
}

func TestIntegration_DropIndex_RefusesStar(t *testing.T) {
	run, _ := setupWrites(t)
	setup := `db.c.createIndex({ p: 1 }); db.c.createIndex({ s: 1 }); `

	_, err := run(setup + `db.c.dropIndex("*")`)
	require.Error(t, err)

	got, err := run(setup + `try { db.c.dropIndex("*") } catch (e) {} print(db.c.getIndexes().length)`)
	require.NoError(t, err)
	assert.Equal(t, "3", got)

	got, err = run(setup + `db.c.dropIndex("p_1"); print(db.c.getIndexes().length)`)
	require.NoError(t, err)
	assert.Equal(t, "2", got)

	got, err = run(setup + `db.c.dropIndexes("*"); print(db.c.getIndexes().length)`)
	require.NoError(t, err)
	assert.Equal(t, "1", got)
}

// An _id of null or undefined means no _id, so one is generated. Other falsy values
// are real ids.
func TestIntegration_Inserts_NullIDIsGenerated(t *testing.T) {
	run, _ := setupWrites(t)
	stored := `db.c.findOne({ a: 7 })._id`
	isGenerated := func(id string) string { return `print(typeof ` + id + `.toHexString)` }
	cases := map[string]string{
		`var id = db.c.insertOne({ _id: null, a: 7 }).insertedId; print(String(id) === String(` + stored + `)); ` + isGenerated(`id`): "true\nfunction",
		`db.c.insertOne({ _id: undefined, a: 7 }); ` + isGenerated(stored):                                                            "function",
		`db.c.insertOne({ a: 7, _id: null }); ` + isGenerated(stored):                                                                 "function",
		`db.c.insertMany([{ _id: null, a: 7 }]); ` + isGenerated(stored):                                                              "function",
		`db.c.insertMany([{ _id: null }, { _id: null }]); print(db.c.countDocuments({}), db.c.countDocuments({ _id: null }))`:         "5 0",
		`db.c.bulkWrite([{ insertOne: { document: { _id: null, a: 7 } } }]); ` + isGenerated(stored):                                  "function",
		`db.c.bulkWrite([{ insertOne: { _id: null, a: 7 } }]); ` + isGenerated(stored):                                                "function",
		isGenerated(`db.c.bulkWrite([{ insertOne: { document: { _id: undefined } } }]).insertedIds["0"]`):                             "function",
		`print(db.c.insertOne({ _id: 0 }).insertedId)`:                                                                                "0",
		`print(db.c.insertOne({ _id: false }).insertedId)`:                                                                            "false",
		`print(JSON.stringify(db.c.insertOne({ _id: "" }).insertedId))`:                                                               `""`,
		// Only the document's own _id.
		`db.c.insertOne({ _id: 7, n: { _id: null } }); print(db.c.findOne({ _id: 7 }).n._id)`: "null",
	}
	for script, want := range cases {
		got, err := run(script)
		require.NoError(t, err, script)
		assert.Equal(t, want, got, script)
	}
}
