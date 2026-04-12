//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestIntegration_CollectionStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.stats()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "ns")
}

func TestIntegration_CollectionIsCapped_False(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.isCapped()`)
	require.NoError(t, err)
	assert.Equal(t, "false", result.RawOutput)
}

func TestIntegration_CollectionIsCapped_True(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	err := testClient.Database(db).CreateCollection(ctx, "capped",
		options.CreateCollection().SetCapped(true).SetSizeInBytes(4096))
	require.NoError(t, err)

	engine := NewGojaEngine(testClient)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.capped.isCapped()`)
	require.NoError(t, err)
	assert.Equal(t, "true", result.RawOutput)
}

func TestIntegration_CollectionSizeHelpers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }])`)
	require.NoError(t, err)

	for _, method := range []string{"dataSize", "storageSize", "totalSize", "totalIndexSize"} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.`+method+`()`)
		require.NoError(t, err, method)
		assert.NotEmpty(t, result.RawOutput, method)
	}
}

func TestIntegration_CollectionGetIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.getIndexes()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "_id_")
}

func TestIntegration_CollectionCount(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([{ x: 1 }, { x: 2 }, { x: 3 }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.count({})`)
	require.NoError(t, err)
	assert.Equal(t, "3", result.RawOutput)
}

func TestIntegration_CollectionRename(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.oldname.insertOne({ x: 1 })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.oldname.renameCollection("newname")`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.newname.count({})`)
	require.NoError(t, err)
	assert.Equal(t, "1", result.RawOutput)
}

func TestIntegration_CollectionValidate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.validate()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "valid")
}

func TestIntegration_CollectionFindAndModify_Update(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1, name: "old" })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		db.test.findAndModify({
			query: { x: 1 },
			update: { $set: { name: "new" } },
			new: true
		})
	`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "new")
}

func TestIntegration_CollectionFindAndModify_Remove(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.test.findAndModify({ query: { x: 1 }, remove: true })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.count({})`)
	require.NoError(t, err)
	assert.Equal(t, "0", result.RawOutput)
}
