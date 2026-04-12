//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Cursor_Explain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ x: 1 }).explain()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "queryPlanner")
}

func TestIntegration_Cursor_ExplainWithVerbosity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ x: 1 }).explain("executionStats")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "executionStats")
}

func TestIntegration_Cursor_Hint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }, { x: 3 }])`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.test.createIndex({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({}).hint({ x: 1 })`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 3)
}

func TestIntegration_Cursor_BatchSizeAndMaxTimeMS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }, { x: 3 }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({}).batchSize(2).maxTimeMS(5000)`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 3)
}

func TestIntegration_Cursor_Collation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ s: "a" }, { s: "A" }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ s: "a" }).collation({ locale: "en", strength: 2 })`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 2)
}

func TestIntegration_Cursor_Comment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ x: 1 }).comment("tagged-query")`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 1)
}
