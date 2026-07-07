package queryengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNormalizeForJS_NestedContainers(t *testing.T) {
	in := bson.M{
		"a":   bson.M{"b": 1},
		"arr": bson.A{bson.M{"c": 2}},
	}

	out, ok := normalizeForJS(in).(map[string]any)
	require.True(t, ok, "top level should be plain map[string]any")

	nested, ok := out["a"].(map[string]any)
	require.True(t, ok, "nested bson.M should become map[string]any")
	assert.Equal(t, 1, nested["b"])

	arr, ok := out["arr"].([]any)
	require.True(t, ok, "bson.A should become []any")
	require.Len(t, arr, 1)
	elem, ok := arr[0].(map[string]any)
	require.True(t, ok, "element bson.M should become map[string]any")
	assert.Equal(t, 2, elem["c"])
}

func TestNormalizeForJS_ScalarPassthrough(t *testing.T) {
	oid := bson.NewObjectID()
	out := normalizeForJS(oid)
	got, ok := out.(bson.ObjectID)
	require.True(t, ok, "scalar BSON type should pass through unchanged")
	assert.Equal(t, oid, got)
}
