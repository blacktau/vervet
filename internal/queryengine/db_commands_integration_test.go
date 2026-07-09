//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_RunCommand_Ping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ ping: 1 })`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "ok")
}

// TestIntegration_RunCommand_PropertyAccess guards the v2 normalizer fix:
// bson.M gained methods in mongo-driver v2, so goja stopped reflecting command
// results as JS objects. Accessing a property of the result then returns
// undefined. normalizeForJS restores plain-object reflection; this asserts the
// property is actually reachable from JS end-to-end.
func TestIntegration_RunCommand_PropertyAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ ping: 1 }).ok`)
	require.NoError(t, err)
	assert.Equal(t, "1", resultText(result), "command result property must be reachable from JS, not undefined")
}

func TestIntegration_RunCommand_CollStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)

	// Create a collection first
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	// Run collStats
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ collStats: "test" })`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "ns")
}

func TestIntegration_AdminCommand_ListDatabases(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.adminCommand({ listDatabases: 1 })`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "databases")
}

func TestIntegration_RunCommand_InvalidCommand_Errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ notARealCommand: 1 })`)
	assert.Error(t, err)
}

// --- Core database methods ---

func TestIntegration_GetCollectionNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)

	// Create two collections
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.alpha.insertOne({ x: 1 })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.beta.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getCollectionNames()`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "alpha")
	assert.Contains(t, resultText(result), "beta")
}

func TestIntegration_GetCollectionInfos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.infocoll.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getCollectionInfos()`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "infocoll")
}

func TestIntegration_CreateCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.createCollection("newcoll")`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "ok")

	// Verify collection exists
	names, err := testClient.Database(db).ListCollectionNames(ctx, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, names, "newcoll")
}

func TestIntegration_CreateView(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.src.insertOne({ x: 1, y: 2 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.createView("myview", "src", [{ $project: { x: 1 } }])`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "ok")

	infos, err := testClient.Database(db).ListCollectionNames(ctx, map[string]any{"name": "myview"})
	require.NoError(t, err)
	assert.Contains(t, infos, "myview")
}

func TestIntegration_DropDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)

	engine := NewGojaEngine(testClient, 0)
	// Create something so the db exists
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.temp.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.dropDatabase()`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "ok")
}

func TestIntegration_Stats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.statscoll.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.stats()`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "collections")
}

func TestIntegration_Version(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.version()`)
	require.NoError(t, err)
	// Should return a version string like "7.0.x"
	assert.NotEmpty(t, resultText(result))
	assert.Contains(t, resultText(result), ".")
}

func TestIntegration_GetSiblingDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	siblingDB := db + "_sibling"
	defer testClient.Database(db).Drop(ctx)
	defer testClient.Database(siblingDB).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)

	// Insert into sibling database via getSiblingDB
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.getSiblingDB("`+siblingDB+`").crossdb.insertOne({ from: "sibling" })`)
	require.NoError(t, err)

	// Verify document exists in sibling database
	var doc map[string]any
	err = testClient.Database(siblingDB).Collection("crossdb").FindOne(ctx, map[string]any{}).Decode(&doc)
	require.NoError(t, err)
	assert.Equal(t, "sibling", doc["from"])
}

func TestIntegration_Aggregate_DbLevel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0)

	// db.aggregate with $listLocalSessions or similar db-level pipeline
	// Use $currentOp-style approach: create data then use $documents (MongoDB 5.1+)
	// Simpler: just test that it runs and returns something
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.aggregate([{ $listLocalSessions: {} }])`)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
