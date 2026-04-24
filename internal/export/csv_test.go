package export

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func serializeCSVHelper(t *testing.T, docs []bson.M, opts CSVOptions) string {
	t.Helper()
	out, err := Serialize(docs, Options{Format: FormatCSV, CSV: opts})
	require.NoError(t, err)
	return string(out)
}

func TestSerializeCSV_HeaderIsUnionOfKeys(t *testing.T) {
	docs := []bson.M{
		{"name": "A", "age": 1},
		{"name": "B", "city": "X"},
	}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	firstLine := strings.SplitN(out, "\n", 2)[0]
	for _, k := range []string{"age", "city", "name"} {
		assert.Contains(t, firstLine, k)
	}
}

func TestSerializeCSV_MissingFieldsAreEmpty(t *testing.T) {
	docs := []bson.M{
		{"name": "A", "age": 1},
		{"name": "B"},
	}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 3)
	// Header is sorted alphabetically: age,name → row for B is ",B"
	assert.Equal(t, ",B", lines[2])
}

func TestSerializeCSV_NoHeader(t *testing.T) {
	docs := []bson.M{{"a": 1}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: false})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 1)
	assert.Equal(t, "1", lines[0])
}

func TestSerializeCSV_TabSeparator(t *testing.T) {
	docs := []bson.M{{"a": 1, "b": 2}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: '\t', IncludeHeader: true})
	assert.Contains(t, out, "a\tb")
	assert.Contains(t, out, "1\t2")
}

func TestSerializeCSV_SemicolonSeparator(t *testing.T) {
	docs := []bson.M{{"a": 1, "b": 2}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ';', IncludeHeader: true})
	assert.Contains(t, out, "a;b")
	assert.Contains(t, out, "1;2")
}

func TestSerializeCSV_UTF8BOM(t *testing.T) {
	docs := []bson.M{{"a": "ñ"}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true, UTF8BOM: true})
	assert.Equal(t, "\xEF\xBB\xBF", out[:3])
}

func TestSerializeCSV_NoBOMByDefault(t *testing.T) {
	docs := []bson.M{{"a": "x"}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	assert.NotEqual(t, "\xEF\xBB\xBF", out[:3])
}

func TestSerializeCSV_QuotesCellsContainingSeparator(t *testing.T) {
	docs := []bson.M{{"s": "a,b"}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	assert.Contains(t, out, `"a,b"`)
}

func TestSerializeCSV_EscapesQuotesInCells(t *testing.T) {
	docs := []bson.M{{"s": `say "hi"`}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	assert.Contains(t, out, `"say ""hi"""`)
}

func TestSerializeCSV_NestedObjectsFlattenedToDotPaths(t *testing.T) {
	docs := []bson.M{{"addr": bson.M{"city": "Paris"}}}
	out := serializeCSVHelper(t, docs, CSVOptions{Separator: ',', IncludeHeader: true})
	firstLine := strings.SplitN(out, "\n", 2)[0]
	assert.Contains(t, firstLine, "addr.city")
	assert.Contains(t, out, "Paris")
}

func TestSerializeCSV_ExplicitColumns(t *testing.T) {
	docs := []bson.M{{"a": 1, "b": 2, "c": 3}}
	out, err := Serialize(docs, Options{
		Format:  FormatCSV,
		Columns: []string{"b", "a"},
		CSV:     CSVOptions{Separator: ',', IncludeHeader: true},
	})
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	assert.Equal(t, "b,a", lines[0])
	assert.Equal(t, "2,1", lines[1])
}

func TestSerializeCSV_DefaultSeparatorIsComma(t *testing.T) {
	// Separator zero-value (0 rune) should fall back to comma.
	docs := []bson.M{{"a": 1, "b": 2}}
	out := serializeCSVHelper(t, docs, CSVOptions{IncludeHeader: true})
	firstLine := strings.SplitN(out, "\n", 2)[0]
	assert.Contains(t, firstLine, "a,b")
}
