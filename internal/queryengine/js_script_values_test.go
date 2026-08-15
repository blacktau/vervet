package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// evalWithValue converts v the way a query result is converted for a script,
// binds it as `doc`, and evaluates expr against it.
func evalWithValue(t *testing.T, v any, expr string) goja.Value {
	t.Helper()
	rt := goja.New()
	require.NoError(t, rt.Set("doc", toJSValue(rt, v)))
	val, err := rt.RunString(expr)
	require.NoError(t, err)
	return val
}

// Results reach scripts as canonical Extended JSON. Left as-is, `String(id)`
// yields "[object Object]" and every id collapses to the same map key — which
// silently produces wrong answers rather than an error.
func TestScriptValue_LongStringifiesToItsDigits(t *testing.T) {
	v := map[string]any{"$numberLong": "2309825"}
	assert.Equal(t, "2309825", evalWithValue(t, v, `String(doc)`).Export())
	assert.Equal(t, "2309825", evalWithValue(t, v, "`${doc}`").Export())
}

func TestScriptValue_ObjectIDHasHexStringAndTimestamp(t *testing.T) {
	v := map[string]any{"$oid": "6a8007006ae0e594998563b1"}
	assert.Equal(t, "6a8007006ae0e594998563b1", evalWithValue(t, v, `String(doc)`).Export())
	assert.Equal(t, true, evalWithValue(t, v, `doc.getTimestamp() instanceof Date`).Export())
}

func TestScriptValue_DateIsARealJSDate(t *testing.T) {
	v := map[string]any{"$date": map[string]any{"$numberLong": "1786393408000"}}
	assert.Equal(t, true, evalWithValue(t, v, `doc instanceof Date`).Export())
	assert.Equal(t, "2026-08-10T20:23:28.000Z", evalWithValue(t, v, `doc.toISOString()`).Export())
}

func TestScriptValue_NestedDocumentKeepsPlainFields(t *testing.T) {
	v := map[string]any{
		"CustomerId": map[string]any{"$numberLong": "1107812"},
		"IsError":    false,
		"Tags":       []any{"a", "b"},
	}
	assert.Equal(t, "1107812|false|a,b",
		evalWithValue(t, v, `[String(doc.CustomerId), doc.IsError, doc.Tags.join(",")].join("|")`).Export())
}

func TestScriptValue_BinarySubtype4RendersAsUUID(t *testing.T) {
	v := map[string]any{"$binary": map[string]any{
		"base64":  "zpMT0UyqRbaWnRFh4ArYUw==",
		"subType": "04",
	}}
	assert.Equal(t, "ce9313d1-4caa-45b6-969d-1161e00ad853", evalWithValue(t, v, `String(doc)`).Export())
}

func TestScriptValue_RegexBecomesUsableRegExp(t *testing.T) {
	v := map[string]any{"$regularExpression": map[string]any{"pattern": "^ab", "options": "i"}}
	assert.Equal(t, true, evalWithValue(t, v, `doc.test("ABc")`).Export())
}

// Converted values must still convert back to BSON when used in a filter.
func TestScriptValue_RoundTripsBackToBSON(t *testing.T) {
	rt := goja.New()
	require.NoError(t, rt.Set("doc", toJSValue(rt, map[string]any{"$numberLong": "2309825"})))
	val, err := rt.RunString(`({ CustomerId: doc })`)
	require.NoError(t, err)

	doc, ok := convertToBson(exportValue(val)).(bson.D)
	require.True(t, ok, "expected a bson.D filter")
	require.Len(t, doc, 1)
	assert.Equal(t, "CustomerId", doc[0].Key)
	assert.Equal(t, int64(2309825), doc[0].Value, "Long must survive as int64, not become a float")
}
