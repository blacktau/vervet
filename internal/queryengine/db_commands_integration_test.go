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

	engine := NewGojaEngine(testClient)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ ping: 1 })`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "ok")
}

func TestIntegration_RunCommand_CollStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

	// Create a collection first
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertOne({ x: 1 })`)
	require.NoError(t, err)

	// Run collStats
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ collStats: "test" })`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "ns")
}

func TestIntegration_AdminCommand_ListDatabases(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.adminCommand({ listDatabases: 1 })`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "databases")
}

func TestIntegration_RunCommand_InvalidCommand_Errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ notARealCommand: 1 })`)
	assert.Error(t, err)
}
