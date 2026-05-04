package queryengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLazyCursor_DefaultState(t *testing.T) {
	c := &lazyCursor{collection: "users"}
	assert.Equal(t, "users", c.collection)
	assert.False(t, c.resolved)
	assert.Nil(t, c.results)
	assert.Equal(t, int64(0), c.limit)
	assert.Equal(t, int64(0), c.skip)
}

func TestLazyCursor_SetLimit(t *testing.T) {
	c := &lazyCursor{}
	c.setLimit(10)
	assert.Equal(t, int64(10), c.limit)
}

func TestLazyCursor_SetSkip(t *testing.T) {
	c := &lazyCursor{}
	c.setSkip(5)
	assert.Equal(t, int64(5), c.skip)
}

func TestLazyCursor_SetSort(t *testing.T) {
	c := &lazyCursor{}
	sort := map[string]any{"name": int64(1)}
	c.setSort(sort)
	assert.Equal(t, sort, c.sort)
}

func TestLazyCursor_SetLimitAfterResolve_Errors(t *testing.T) {
	c := &lazyCursor{resolved: true}
	err := c.setLimit(10)
	assert.Error(t, err)
}

func TestLazyCursor_SetSkipAfterResolve_Errors(t *testing.T) {
	c := &lazyCursor{resolved: true}
	err := c.setSkip(5)
	assert.Error(t, err)
}

func TestLazyCursor_SetSortAfterResolve_Errors(t *testing.T) {
	c := &lazyCursor{resolved: true}
	err := c.setSort(map[string]any{"name": int64(1)})
	assert.Error(t, err)
}

func TestLazyCursor_BuildPageContext_Find(t *testing.T) {
	c := &lazyCursor{
		collection: "users",
		filter:     map[string]any{"status": "active"},
		projection: map[string]any{"name": int64(1)},
		sort:       map[string]any{"_id": int64(1)},
		limit:      100,
		skip:       5,
		maxTimeMS:  3000,
		comment:    "tagged",
	}
	pc := c.buildPageContext()
	assert.NotNil(t, pc)
	assert.Equal(t, "users", pc.Collection)
	assert.Equal(t, int64(100), pc.UserLimit)
	assert.Equal(t, int64(5), pc.UserSkip)
	assert.Equal(t, int64(3000), pc.MaxTimeMS)
	assert.Equal(t, "tagged", pc.Comment)
}

func TestLazyCursor_BuildPageContext_FindOneReturnsNil(t *testing.T) {
	c := &lazyCursor{collection: "users", isFindOne: true}
	assert.Nil(t, c.buildPageContext())
}

func TestLazyCursor_CursorOptionsFieldsDefaults(t *testing.T) {
	c := &lazyCursor{}
	assert.Nil(t, c.hint)
	assert.Equal(t, int64(0), c.maxTimeMS)
	assert.Equal(t, int32(0), c.batchSize)
	assert.Nil(t, c.collation)
	assert.Equal(t, "", c.comment)
}
