package queryengine

import (
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestEffectivePaging_NoUserLimit(t *testing.T) {
	pc := models.PageContext{}
	skip, limit, empty := effectivePaging(pc, 3, 25)
	assert.False(t, empty)
	assert.Equal(t, int64(75), skip)
	assert.Equal(t, int64(25), limit)
}

func TestEffectivePaging_WithUserSkip(t *testing.T) {
	pc := models.PageContext{UserSkip: 10}
	skip, limit, _ := effectivePaging(pc, 2, 25)
	assert.Equal(t, int64(60), skip)
	assert.Equal(t, int64(25), limit)
}

func TestEffectivePaging_UserLimitClampsLastPartial(t *testing.T) {
	pc := models.PageContext{UserLimit: 50}
	skip, limit, empty := effectivePaging(pc, 1, 30)
	assert.False(t, empty)
	assert.Equal(t, int64(30), skip)
	assert.Equal(t, int64(20), limit)
}

func TestEffectivePaging_UserLimitFullPage(t *testing.T) {
	pc := models.PageContext{UserLimit: 50}
	skip, limit, empty := effectivePaging(pc, 1, 25)
	assert.False(t, empty)
	assert.Equal(t, int64(25), skip)
	assert.Equal(t, int64(25), limit)
}

func TestEffectivePaging_BeyondUserLimitIsEmpty(t *testing.T) {
	pc := models.PageContext{UserLimit: 10}
	_, _, empty := effectivePaging(pc, 5, 25)
	assert.True(t, empty)
}

func TestIsEmptyFilter(t *testing.T) {
	assert.True(t, isEmptyFilter(nil))
	assert.True(t, isEmptyFilter(map[string]any{}))
	assert.False(t, isEmptyFilter(map[string]any{"x": 1}))
	assert.False(t, isEmptyFilter("not a map"))
}

func TestExtractCount(t *testing.T) {
	r := models.QueryResult{Documents: []any{map[string]any{"count": int64(42)}}}
	assert.Equal(t, int64(42), extractCount(r))
	r = models.QueryResult{Documents: []any{map[string]any{"count": float64(7)}}}
	assert.Equal(t, int64(7), extractCount(r))
	assert.Equal(t, int64(0), extractCount(models.QueryResult{}))
}
