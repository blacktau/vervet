//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupNumberData(t *testing.T) (*GojaEngine, string, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db := dbName(t)
	t.Cleanup(func() { testClient.Database(db).Drop(context.Background()) })

	engine := NewGojaEngine(testClient, 10, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: 1, n: NumberInt(5), l: NumberLong("7") })`)
	require.NoError(t, err)
	return engine, db, ctx
}

// Write acknowledgements report counts as plain numbers, as countDocuments and
// mongosh do, so `=== 1` comparisons hold.
func TestIntegration_Numbers_WriteAckCounts(t *testing.T) {
	cases := map[string]struct{ script, want string }{
		"updateOne":  {`var r = db.c.updateOne({ _id: 1 }, { $set: { x: 1 } }); print(typeof r.modifiedCount, r.modifiedCount === 1, typeof r.matchedCount, r.matchedCount === 1)`, "number true number true"},
		"updateMany": {`var r = db.c.updateMany({}, { $set: { x: 2 } }); print(typeof r.modifiedCount, r.modifiedCount === 1, r.matchedCount === 1)`, "number true true"},
		"replaceOne": {`var r = db.c.replaceOne({ _id: 1 }, { y: 1 }); print(typeof r.modifiedCount, r.modifiedCount === 1)`, "number true"},
		"deleteOne":  {`var r = db.c.deleteOne({ _id: 1 }); print(typeof r.deletedCount, r.deletedCount === 1)`, "number true"},
		"deleteMany": {`var r = db.c.deleteMany({}); print(typeof r.deletedCount, r.deletedCount === 1)`, "number true"},
		"bulkWrite": {`var r = db.c.bulkWrite([{ insertOne: { document: { _id: 2 } } }, { updateOne: { filter: { _id: 1 }, update: { $set: { x: 3 } } } }, { deleteOne: { filter: { _id: 2 } } }]);
			print(r.insertedCount === 1, r.matchedCount === 1, r.modifiedCount === 1, r.deletedCount === 1, r.upsertedCount === 0)`, "true true true true true"},
	}
	engine, db, ctx := setupNumberData(t)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Each case starts from the same single document. The database name
			// comes from the test name, so subtests cannot have their own.
			_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.deleteMany({}); db.c.insertOne({ _id: 1 })`)
			require.NoError(t, err)
			assert.Equal(t, tc.want, runScript(t, engine, ctx, db, tc.script))
		})
	}
}

// The same stored value has the same type whichever aggregate returns it: a
// stored long is a Long, a stored int is a plain number.
func TestIntegration_Numbers_AggregatesAgreeOnStoredTypes(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	got := runScript(t, engine, ctx, db, `
		var a = db.c.aggregate([{ $match: {} }]).toArray()[0];
		var b = db.aggregate([{ $documents: [{ n: NumberInt(5), l: NumberLong("7") }] }])[0];
		print(typeof a.l, a.l === 7, String(a.l), typeof a.n, a.n === 5);
		print(typeof b.l, b.l === 7, String(b.l), typeof b.n, b.n === 5);
	`)
	assert.Equal(t, "object false 7 number true\nobject false 7 number true", got)
}

func TestIntegration_Numbers_DbAggregateHasCursorMethods(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	got := runScript(t, engine, ctx, db, `print(db.aggregate([{ $documents: [{ n: 1 }] }]).toArray().length)`)
	assert.Equal(t, "1", got)
}

// EJSON.parse with { relaxed: false } keeps a long beyond 2^53 exact.
func TestIntegration_Numbers_EJSONParseStrictKeepsLongPrecision(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	got := runScript(t, engine, ctx, db, `
		var v = EJSON.parse('{"n": {"$numberLong": "9007199254740993"}}', { relaxed: false }).n;
		print(typeof v, String(v));
	`)
	assert.Equal(t, "object 9007199254740993", got)
}

// Adding toArray must not change how the result displays when it is the
// script's final value.
func TestIntegration_Numbers_DbAggregateDisplaysAsDocuments(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	for _, script := range []string{
		`db.aggregate([{ $documents: [{ n: 1 }] }])`,
		`db.aggregate([{ $documents: [{ n: 1 }] }]).toArray()`,
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err)
		assert.Len(t, result.Documents, 1, script)
	}
}

// upsertedCount is reported, as a plain number, on every kind of update ack.
func TestIntegration_Numbers_UpdateAcksHaveUpsertedCount(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	got := runScript(t, engine, ctx, db, `
		print(db.c.updateOne({ _id: 1 }, { $set: { x: 1 } }).upsertedCount === 0);
		print(db.c.updateMany({}, { $set: { x: 2 } }).upsertedCount === 0);
		print(db.c.replaceOne({ _id: 1 }, { y: 1 }).upsertedCount === 0);
	`)
	assert.Equal(t, "true\ntrue\ntrue", got)
}
