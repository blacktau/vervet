package export

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSerializeNDJSON_OneLinePerDoc(t *testing.T) {
	docs := []bson.M{{"a": 1}, {"b": 2}, {"c": 3}}
	out, err := Serialize(docs, Options{Format: FormatNDJSON})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	assert.Len(t, lines, 3)
	assert.Contains(t, lines[0], `"a"`)
	assert.Contains(t, lines[1], `"b"`)
	assert.Contains(t, lines[2], `"c"`)
}

func TestSerializeNDJSON_TrailingNewline(t *testing.T) {
	out, err := Serialize([]bson.M{{"a": 1}}, Options{Format: FormatNDJSON})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(out), "\n"))
}

func TestSerializeNDJSON_EmptyDocsYieldsEmptyBytes(t *testing.T) {
	out, err := Serialize([]bson.M{}, Options{Format: FormatNDJSON})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestSerializeNDJSON_PreservesBSONTypes(t *testing.T) {
	oid := primitive.NewObjectID()
	out, err := Serialize([]bson.M{{"_id": oid}}, Options{Format: FormatNDJSON})
	require.NoError(t, err)
	assert.Contains(t, string(out), `"$oid"`)
	assert.Contains(t, string(out), oid.Hex())
}
