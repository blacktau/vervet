package queryengine

import (
	"encoding/json"
	"testing"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
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
		ec:         &execContext{dbName: "shop"},
		collection: "users",
		filter:     map[string]any{"status": "active"},
		projection: map[string]any{"name": int64(1)},
		sort:       map[string]any{"_id": int64(1)},
		limit:      100,
		skip:       5,
		maxTimeMS:  3000,
		comment:    "tagged",
	}
	pc, err := c.buildPageContext()
	require.NoError(t, err)
	require.NotNil(t, pc)
	assert.Equal(t, "shop", pc.Database)
	assert.Equal(t, "users", pc.Collection)
	assert.JSONEq(t, `{"status":"active"}`, pc.Filter)
	assert.JSONEq(t, `{"name":{"$numberLong":"1"}}`, pc.Projection)
	assert.Equal(t, int64(100), pc.UserLimit)
	assert.Equal(t, int64(5), pc.UserSkip)
	assert.Equal(t, int64(3000), pc.MaxTimeMS)
	assert.Equal(t, "tagged", pc.Comment)
}

func TestLazyCursor_BuildPageContext_FindOneReturnsNil(t *testing.T) {
	c := &lazyCursor{collection: "users", isFindOne: true}
	pc, err := c.buildPageContext()
	require.NoError(t, err)
	assert.Nil(t, pc)
}

func TestLazyCursor_CursorOptionsFieldsDefaults(t *testing.T) {
	c := &lazyCursor{}
	assert.Nil(t, c.hint)
	assert.Equal(t, int64(0), c.maxTimeMS)
	assert.Equal(t, int32(0), c.batchSize)
	assert.Nil(t, c.collation)
	assert.Equal(t, "", c.comment)
}

// TestPageContext_RoundTripKeepsOrderAndTypes guards the paging round-trip:
// the PageContext goes to the frontend as JSON and comes back for page 2, so
// sort and hint order (and ObjectId filters) must survive the trip.
func TestPageContext_RoundTripKeepsOrderAndTypes(t *testing.T) {
	oid := bson.NewObjectID()
	c := &lazyCursor{
		ec:         &execContext{dbName: "shop"},
		collection: "users",
		filter:     bson.D{{Key: "owner", Value: map[string]any{"__bsonValue": &bsonWrapper{Value: oid}}}},
		sort:       bson.D{{Key: "z", Value: int64(1)}, {Key: "a", Value: int64(-1)}, {Key: "m", Value: int64(1)}},
		hint:       bson.D{{Key: "z", Value: int64(1)}, {Key: "a", Value: int64(1)}},
	}
	pc, err := c.buildPageContext()
	require.NoError(t, err)

	wire, err := json.Marshal(pc)
	require.NoError(t, err)
	var back models.PageContext
	require.NoError(t, json.Unmarshal(wire, &back))

	for i := 0; i < 20; i++ {
		q, err := decodePageContext(back)
		require.NoError(t, err)
		assert.Equal(t, bson.D{{Key: "owner", Value: oid}}, q.filter)
		assert.Equal(t, bson.D{{Key: "z", Value: int64(1)}, {Key: "a", Value: int64(-1)}, {Key: "m", Value: int64(1)}}, q.sort)
		assert.Equal(t, bson.D{{Key: "z", Value: int64(1)}, {Key: "a", Value: int64(1)}}, q.hint)
		assert.Nil(t, q.projection)
	}
}

func TestPageContext_HintByName(t *testing.T) {
	pc, err := (&lazyCursor{ec: &execContext{}, collection: "users", hint: "name_1"}).buildPageContext()
	require.NoError(t, err)
	q, err := decodePageContext(*pc)
	require.NoError(t, err)
	assert.Equal(t, "name_1", q.hint)
}

func TestPageContext_CarriesBatchSize(t *testing.T) {
	pc, err := (&lazyCursor{ec: &execContext{}, collection: "users", batchSize: 50}).buildPageContext()
	require.NoError(t, err)
	assert.Equal(t, int32(50), pc.BatchSize)
}

// Non-document values fall back to "absent", as dispatch does, instead of
// failing the query.
func TestPageContext_NonDocumentFilterIsAbsent(t *testing.T) {
	pc, err := (&lazyCursor{ec: &execContext{}, collection: "users", filter: int64(1), sort: "x"}).buildPageContext()
	require.NoError(t, err)
	assert.Empty(t, pc.Filter)
	assert.Empty(t, pc.Sort)
}

// Any hint value and the collation's types must reach later pages unchanged.
func TestPageContext_HintAndCollationRoundTrip(t *testing.T) {
	c := &lazyCursor{
		ec:         &execContext{},
		collection: "users",
		hint:       int64(7),
		collation:  map[string]any{"locale": "en", "strength": int64(2), "numericOrdering": true},
	}
	pc, err := c.buildPageContext()
	require.NoError(t, err)

	wire, err := json.Marshal(pc)
	require.NoError(t, err)
	var back models.PageContext
	require.NoError(t, json.Unmarshal(wire, &back))

	q, err := decodePageContext(back)
	require.NoError(t, err)
	assert.Equal(t, int64(7), q.hint)
	assert.Equal(t, map[string]any{"locale": "en", "strength": int64(2), "numericOrdering": true}, q.collation)
}
