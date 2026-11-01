package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vervet/internal/models"
)

// runOutput evaluates src in a runtime with print/printjson/console installed
// and returns everything the script wrote.
func runOutput(t *testing.T, src string) string {
	t.Helper()
	rt := goja.New()
	require.NoError(t, registerBSONTypes(rt))
	out := &scriptOutput{}
	require.NoError(t, registerOutput(rt, out))
	_, err := rt.RunString(src)
	require.NoError(t, err)
	return out.text()
}

func TestPrint_JoinsArgumentsOnOneLine(t *testing.T) {
	assert.Equal(t, "a b 1", runOutput(t, `print("a", "b", 1)`))
}

func TestPrint_StringsAreNotQuoted(t *testing.T) {
	assert.Equal(t, "hello", runOutput(t, `print("hello")`))
}

func TestPrint_ObjectIsInspectedNotStringified(t *testing.T) {
	assert.Equal(t, "{ a: 1, b: [ 1, 2 ] }", runOutput(t, `print({a: 1, b: [1, 2]})`))
}

func TestPrintjson_EmptyObject(t *testing.T) {
	assert.Equal(t, "{}", runOutput(t, `printjson({})`))
}

func TestPrintjson_QuotesStringValues(t *testing.T) {
	assert.Equal(t, "{\n  name: 'alice'\n}", runOutput(t, `printjson({name: "alice"})`))
}

// printjson expands every non-empty container, however short — print does not.
func TestPrintjson_AlwaysExpandsContainers(t *testing.T) {
	assert.Equal(t, "{\n  a: 1,\n  b: 2\n}", runOutput(t, `printjson({a: 1, b: 2})`))
	assert.Equal(t, "{ a: 1, b: 2 }", runOutput(t, `print({a: 1, b: 2})`))
}

// Wide values wrap one entry per line, as the mongosh shell does.
func TestPrintjson_WrapsWideValues(t *testing.T) {
	got := runOutput(t, `printjson([{ customerId: 2039505, state: 'NoSearchEverFound_NeedsQueueing', failedOn: 'contacts' }])`)
	assert.Equal(t, `[
  {
    customerId: 2039505,
    state: 'NoSearchEverFound_NeedsQueueing',
    failedOn: 'contacts'
  }
]`, got)
}

func TestPrintjson_RendersBSONValues(t *testing.T) {
	got := runOutput(t, `print({ id: NumberLong("2309825"), at: ISODate("2026-08-10T20:00:00Z") })`)
	assert.Equal(t, `{ id: 2309825, at: ISODate("2026-08-10T20:00:00.000Z") }`, got)
}

// console.* is captured rather than dropped: scripts use console.error for
// summaries they still expect to see.
func TestConsole_IsCaptured(t *testing.T) {
	got := runOutput(t, `console.error("to stderr"); console.log("to stdout"); console.warn("careful")`)
	assert.Equal(t, "to stderr\nto stdout\ncareful", got)
}

func TestInspect_NullAndUndefined(t *testing.T) {
	assert.Equal(t, "null undefined", runOutput(t, `print(null, undefined)`))
}

// runOutputLines is runOutput keeping each call as its own entry with a level.
func runOutputLines(t *testing.T, src string) []models.OutputLine {
	t.Helper()
	rt := goja.New()
	require.NoError(t, registerBSONTypes(rt))
	out := &scriptOutput{}
	require.NoError(t, registerOutput(rt, out))
	_, err := rt.RunString(src)
	require.NoError(t, err)
	return out.lines
}

func TestOutput_ConsoleLevels(t *testing.T) {
	got := runOutputLines(t, `
		print("p"); printjson(1);
		console.log("l"); console.info("i"); console.debug("d"); console.trace("t");
		console.warn("w"); console.error("e");
	`)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "p"},
		{Level: models.OutputInfo, Text: "1"},
		{Level: models.OutputInfo, Text: "l"},
		{Level: models.OutputInfo, Text: "i"},
		{Level: models.OutputInfo, Text: "d"},
		{Level: models.OutputInfo, Text: "t"},
		{Level: models.OutputWarning, Text: "w"},
		{Level: models.OutputError, Text: "e"},
	}, got)
}

// A printjson that spans several lines is still one call, so one entry.
func TestOutput_PrintjsonIsOneEntry(t *testing.T) {
	got := runOutputLines(t, `printjson({a: 1, b: 2})`)
	require.Len(t, got, 1)
	assert.Equal(t, "{\n  a: 1,\n  b: 2\n}", got[0].Text)
}
