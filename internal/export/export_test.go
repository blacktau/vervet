package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSerialize_UnknownFormat(t *testing.T) {
	_, err := Serialize([]bson.M{{"a": 1}}, Options{Format: "xml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
}

func TestSerialize_EmptyDocsProducesEmptyOutputForJSON(t *testing.T) {
	out, err := Serialize([]bson.M{}, Options{Format: FormatJSON})
	assert.NoError(t, err)
	assert.Equal(t, "[]", string(out))
}
