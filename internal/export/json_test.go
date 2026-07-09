package export

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSerializeJSON_BasicDocs(t *testing.T) {
	docs := []bson.M{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}
	out, err := Serialize(docs, Options{Format: FormatJSON})
	require.NoError(t, err)

	// Round-trip through json.Unmarshal to compare semantically.
	var parsed []map[string]any
	require.NoError(t, json.Unmarshal(out, &parsed))
	assert.Len(t, parsed, 2)
	assert.Equal(t, "Alice", parsed[0]["name"])
	assert.Equal(t, "Bob", parsed[1]["name"])
}

func TestSerializeJSON_PreservesBSONTypesAsExtJSON(t *testing.T) {
	oid := bson.NewObjectID()
	docs := []bson.M{{"_id": oid, "value": "x"}}

	out, err := Serialize(docs, Options{Format: FormatJSON})
	require.NoError(t, err)

	assert.Contains(t, string(out), `"$oid"`)
	assert.Contains(t, string(out), oid.Hex())
}

func TestSerializeJSON_PrettyPrinted(t *testing.T) {
	out, err := Serialize([]bson.M{{"a": 1}}, Options{Format: FormatJSON})
	require.NoError(t, err)
	// pretty-printed output has newlines
	assert.Contains(t, string(out), "\n")
}
