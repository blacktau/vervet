package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFlattenDoc_Scalars(t *testing.T) {
	doc := bson.M{"name": "Alice", "age": 30, "active": true}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)

	m := pairsToMap(pairs)
	assert.Equal(t, "Alice", m["name"])
	assert.Equal(t, "30", m["age"])
	assert.Equal(t, "true", m["active"])
}

func TestFlattenDoc_NestedObjectsUseDotPaths(t *testing.T) {
	doc := bson.M{
		"address": bson.M{"city": "Paris", "geo": bson.M{"lat": 48.85}},
	}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)

	m := pairsToMap(pairs)
	assert.Equal(t, "Paris", m["address.city"])
	assert.Equal(t, "48.85", m["address.geo.lat"])
}

func TestFlattenDoc_ArraysOfPrimitivesStayAsEJSONString(t *testing.T) {
	doc := bson.M{"tags": bson.A{"admin", "ops"}}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)

	m := pairsToMap(pairs)
	assert.Equal(t, `["admin","ops"]`, m["tags"])
}

func TestFlattenDoc_ArraysOfObjectsStayAsEJSONString(t *testing.T) {
	doc := bson.M{"items": bson.A{bson.M{"id": 1}, bson.M{"id": 2}}}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)

	m := pairsToMap(pairs)
	assert.Contains(t, m["items"], `"id"`)
	assert.Contains(t, m["items"], `1`)
}

func TestFlattenDoc_ObjectIDSerializedAsExtJSON(t *testing.T) {
	oid := primitive.NewObjectID()
	doc := bson.M{"_id": oid}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)

	m := pairsToMap(pairs)
	assert.Contains(t, m["_id"], `"$oid"`)
	assert.Contains(t, m["_id"], oid.Hex())
}

func TestFlattenDoc_StringsNotQuoted(t *testing.T) {
	doc := bson.M{"s": "hello"}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)
	m := pairsToMap(pairs)
	// Plain strings are emitted as-is, not JSON-quoted, so CSV quoting handles them.
	assert.Equal(t, "hello", m["s"])
}

func TestFlattenDoc_NilBecomesEmptyString(t *testing.T) {
	doc := bson.M{"x": nil}
	pairs, err := flattenDoc(doc)
	require.NoError(t, err)
	m := pairsToMap(pairs)
	assert.Equal(t, "", m["x"])
}

func pairsToMap(pairs []flatPair) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.path] = p.value
	}
	return m
}
