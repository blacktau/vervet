//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_EJSON_Stringify_QueryResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	// Insert a document
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.ejsontest.insertOne({ name: "alice", age: 30 })`)
	require.NoError(t, err)

	// Use EJSON.stringify on a find result
	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		var docs = db.ejsontest.find({}).toArray();
		EJSON.stringify(docs[0])
	`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "alice")
	assert.Contains(t, result.RawOutput, "$oid") // _id should be in Extended JSON format
}

func TestIntegration_EJSON_Stringify_WithIndent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `EJSON.stringify({ x: 1, y: 2 }, null, 2)`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "\n")
}

func TestIntegration_EJSON_Parse_ExtendedJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	// Parse extended JSON, stringify it back, and verify roundtrip
	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		var doc = EJSON.parse('{"name": "from_ejson", "count": 42}');
		EJSON.stringify(doc)
	`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "from_ejson")
	assert.Contains(t, result.RawOutput, "42")
}

func TestIntegration_EJSON_SerializeDeserialize_Roundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	// Insert, fetch via toArray to get plain docs, serialize/deserialize roundtrip
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.ejsontest.insertOne({ label: "original" })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		var docs = db.ejsontest.find({ label: "original" }).toArray();
		var doc = docs[0];
		var serialized = EJSON.serialize(doc);
		var deserialized = EJSON.deserialize(serialized);
		deserialized.label
	`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "original")
}
