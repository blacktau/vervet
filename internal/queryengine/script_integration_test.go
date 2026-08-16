//go:build integration

package queryengine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These cover the shapes a real mongosh maintenance script relies on: ids that
// stringify, dates that compare, cursors that return every match, and counts
// that are numbers. Each one used to fail silently rather than loudly.

func setupScriptData(t *testing.T) (*GojaEngine, string, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db := dbName(t)
	t.Cleanup(func() { testClient.Database(db).Drop(context.Background()) })

	engine := NewGojaEngine(testClient, 10, "") // small page size: paging must not truncate scripts
	_, err := engine.ExecuteQuery(ctx, testURI, db, `
		const docs = [];
		for (let i = 0; i < 25; i++) {
			docs.push({
				CustomerId: NumberLong(String(2309800 + i)),
				IsError: i % 5 === 0,
				CreatedAt: ISODate("2026-08-10T20:00:00Z"),
			});
		}
		db.responses.insertMany(docs);
	`)
	require.NoError(t, err)
	return engine, db, ctx
}

func runScript(t *testing.T, engine *GojaEngine, ctx context.Context, db, script string) string {
	t.Helper()
	result, err := engine.ExecuteQuery(ctx, testURI, db, script)
	require.NoError(t, err)
	return result.RawOutput
}

// distinct returns a bare array of Longs, as mongosh does — not a {values: ...}
// wrapper, and not plain numbers.
func TestIntegration_Script_DistinctIsAnArrayOfLongs(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const ids = db.responses.distinct("CustomerId", { IsError: false });
		print(Array.isArray(ids) + "|" + ids.length + "|" + String(ids[0]));
	`)
	assert.Equal(t, "true|20|2309801", got)
}

// The set-membership pattern every one of these scripts is built on.
func TestIntegration_Script_LongsAreUsableAsSetKeys(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const ok = new Set(db.responses.distinct("CustomerId", { IsError: false }).map(v => String(v)));
		const all = db.responses.distinct("CustomerId").map(v => String(v));
		print(ok.size + "|" + all.filter(id => !ok.has(id)).length);
	`)
	assert.Equal(t, "20|5", got)
}

func TestIntegration_Script_CountsAreNumbers(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		print(typeof db.responses.countDocuments({}));
		print(db.responses.estimatedDocumentCount() === 0);
		print(db.responses.find({}).count());
	`)
	assert.Equal(t, "number\nfalse\n25", got)
}

// A cursor terminal in a script must see every match. Paging applies only to
// the cursor left as the script's final value.
func TestIntegration_Script_ToArrayIsNotTruncatedByPageSize(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `print(db.responses.find({}).toArray().length)`)
	assert.Equal(t, "25", got)
}

func TestIntegration_Script_DocumentFieldsAreBSONValues(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const doc = db.responses.find({}).sort({ CustomerId: 1 }).limit(1).toArray()[0];
		print(String(doc.CustomerId));
		print(doc.CreatedAt instanceof Date);
		print(doc.CreatedAt >= ISODate("2026-08-01T00:00:00Z"));
		print(doc._id.getTimestamp() instanceof Date);
		print(db.responses.find({ _id: doc._id }).toArray().length);
	`)
	assert.Equal(t, "2309800\ntrue\ntrue\ntrue\n1", got)
}

// mongosh runs scripts at top level, where this rebinding is legal.
func TestIntegration_Script_CanRebindDbFromItself(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const db = db.getSiblingDB("`+db+`");
		print(db.getCollectionNames().includes("responses"));
	`)
	assert.Equal(t, "true", got)
}

// A script that throws part way through has usually printed the results that
// explain why. mongosh shows that output; so must we.
func TestIntegration_Script_KeepsOutputPrintedBeforeAnError(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `
		print("checked 25 documents");
		throw new Error("collection is empty. Wrong database.");
	`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checked 25 documents")
	assert.Contains(t, err.Error(), "Wrong database")
}

// Values written by a script must land with the BSON types it asked for.
func TestIntegration_Script_WritesPreserveBSONTypes(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		db.tasks.insertMany([{
			_id: UUID(),
			CreatedAt: ISODate(new Date().toISOString()),
			SchemaVersion: NumberInt(1),
			CustomerId: NumberLong("2309825"),
			ContactId: null,
			Status: "Pending",
			Actions: [],
		}], { ordered: false });

		const types = db.tasks.aggregate([{ $project: {
			id: { $type: "$_id" },
			customer: { $type: "$CustomerId" },
			schema: { $type: "$SchemaVersion" },
			created: { $type: "$CreatedAt" },
		} }]).toArray()[0];
		print([types.id, types.customer, types.schema, types.created].join(","));
	`)
	assert.Equal(t, "binData,long,int,date", got)
}

func TestIntegration_Script_InsertManyResultIsCountable(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const res = db.tasks.insertMany([{ n: 1 }, { n: 2 }], { ordered: false });
		print(Object.keys(res.insertedIds).length);
	`)
	assert.Equal(t, "2", got)
}

// The whole-script shape: guards, distinct, classification, CSV output.
func TestIntegration_Script_EndToEndMaintenanceScript(t *testing.T) {
	engine, db, ctx := setupScriptData(t)
	got := runScript(t, engine, ctx, db, `
		const OUTAGE_START = ISODate("2026-08-01T00:00:00Z");
		const affected = [2309800, 2309801, 2309805];
		const asLong = affected.map(id => NumberLong(String(id)));
		const key = v => String(v);

		if (!db.getCollectionNames().includes("responses")) {
			throw new Error("responses not found in db " + db.getName());
		}
		if (db.responses.estimatedDocumentCount() === 0) {
			throw new Error("responses is empty. Wrong database.");
		}

		const recovered = new Set(
			db.responses.distinct("CustomerId", {
				CustomerId: { $in: asLong }, IsError: false, CreatedAt: { $gte: OUTAGE_START },
			}).map(key)
		);

		print("CustomerId,State");
		affected.forEach(id => print(id + "," + (recovered.has(key(id)) ? "AutoRecovered" : "NeedsQueueing")));
		console.error("total: " + affected.length);
	`)
	assert.Equal(t, strings.Join([]string{
		"CustomerId,State",
		"2309800,NeedsQueueing",
		"2309801,AutoRecovered",
		"2309805,NeedsQueueing",
		"total: 3",
	}, "\n"), got)
}
