//go:build integration

package queryengine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FetchPage_RoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 25)

	// Seed 60 docs with monotonic n
	docs := ""
	for i := 0; i < 60; i++ {
		if i > 0 {
			docs += ","
		}
		docs += fmt.Sprintf("{n:%d}", i)
	}
	_, err := engine.ExecuteQuery(ctx, testURI, db, "db.paged.insertMany(["+docs+"])")
	require.NoError(t, err)

	// Initial query: first 25 + PageContext
	res, err := engine.ExecuteQuery(ctx, testURI, db, `db.paged.find({}).sort({n:1})`)
	require.NoError(t, err)
	assert.Len(t, res.Documents, 25)
	require.NotNil(t, res.PageContext)
	assert.Equal(t, "paged", res.PageContext.Collection)

	pc := *res.PageContext

	// Page 2 (0-indexed) -> last 10
	page2, err := engine.FetchPage(ctx, db, pc, 2, 25)
	require.NoError(t, err)
	assert.Len(t, page2.Documents, 10)

	// Empty filter -> estimated count
	count, est, err := engine.CountForPage(ctx, db, pc)
	require.NoError(t, err)
	assert.Equal(t, int64(60), count)
	assert.True(t, est)

	// Filtered -> exact count
	pcF := pc
	pcF.Filter = map[string]any{"n": map[string]any{"$gte": int64(10)}}
	count, est, err = engine.CountForPage(ctx, db, pcF)
	require.NoError(t, err)
	assert.False(t, est)
	assert.Equal(t, int64(50), count)
}

func TestIntegration_FetchPage_RespectsUserLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 25)

	docs := ""
	for i := 0; i < 100; i++ {
		if i > 0 {
			docs += ","
		}
		docs += fmt.Sprintf("{n:%d}", i)
	}
	_, err := engine.ExecuteQuery(ctx, testURI, db, "db.capped.insertMany(["+docs+"])")
	require.NoError(t, err)

	// User limits to 50 — page 1 (0-indexed) should return 25, page 2 empty.
	pc := models.PageContext{Collection: "capped", Filter: map[string]any{}, UserLimit: 50, Sort: map[string]any{"n": int64(1)}}
	page1, err := engine.FetchPage(ctx, db, pc, 1, 25)
	require.NoError(t, err)
	assert.Len(t, page1.Documents, 25)

	page2, err := engine.FetchPage(ctx, db, pc, 2, 25)
	require.NoError(t, err)
	assert.Len(t, page2.Documents, 0)

	count, _, err := engine.CountForPage(ctx, db, pc)
	require.NoError(t, err)
	assert.Equal(t, int64(50), count)
}
