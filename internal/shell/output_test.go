package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"vervet/internal/models"
)

const testNonce = "n0nce"

// framed builds stdout the way the wrapped script writes it.
func framed(parts ...string) string {
	return strings.Join(parts, "\n") + "\n"
}

func begin(level string) string { return frameMarker(testNonce) + "B:" + level }
func end() string               { return frameMarker(testNonce) + "E" }
func result() string            { return frameMarker(testNonce) + "R" }

func TestSplitOutput_FramedCallsAndResult(t *testing.T) {
	stdout := framed(
		begin("info"), "Top users:", end(),
		begin("warning"), "slow", end(),
		begin("info"), "{", "  a: 1", "}", end(),
		result(), `[{"n":1}]`,
	)
	out, res, hasResult := splitOutput(stdout, testNonce)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "Top users:"},
		{Level: models.OutputWarning, Text: "slow"},
		{Level: models.OutputInfo, Text: "{\n  a: 1\n}"},
	}, out)
	assert.True(t, hasResult)
	assert.Equal(t, `[{"n":1}]`, res)
}

// A script whose final value is undefined writes no result marker.
func TestSplitOutput_NoResult(t *testing.T) {
	out, res, hasResult := splitOutput(framed(begin("error"), "e", end()), testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputError, Text: "e"}}, out)
	assert.False(t, hasResult)
	assert.Empty(t, res)
}

// A call that prints nothing (print()) is still an entry, as in the builtin engine.
func TestSplitOutput_EmptyCall(t *testing.T) {
	out, _, _ := splitOutput(framed(begin("info"), "", end()), testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: ""}}, out)
}

// Text written around the wrappers (process.stdout.write, mongosh notices)
// is kept as info, one entry per run of lines.
func TestSplitOutput_StrayTextIsInfo(t *testing.T) {
	stdout := framed("notice one", "notice two", begin("info"), "p", end(), "tail")
	out, _, _ := splitOutput(stdout, testNonce)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "notice one\nnotice two"},
		{Level: models.OutputInfo, Text: "p"},
		{Level: models.OutputInfo, Text: "tail"},
	}, out)
}

// If printjson calls the wrapped print itself, the inner frame is part of the
// outer call, not an entry of its own.
func TestSplitOutput_NestedFramesStayOneEntry(t *testing.T) {
	stdout := framed(begin("info"), begin("info"), "{ a: 1 }", end(), end())
	out, _, _ := splitOutput(stdout, testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "{ a: 1 }"}}, out)
}

// Markers for another run's nonce are just text.
func TestSplitOutput_OtherNonceIsText(t *testing.T) {
	other := frameMarker("other") + "B:info"
	out, _, _ := splitOutput(framed(begin("info"), other, end()), testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: other}}, out)
}

// A script that fails mid-call leaves a frame open: what it wrote is kept.
func TestSplitOutput_UnclosedFrameIsKept(t *testing.T) {
	out, _, hasResult := splitOutput(framed(begin("info"), "partial"), testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "partial"}}, out)
	assert.False(t, hasResult)
}

func TestBuildResult_PrintedLinesAndDocuments(t *testing.T) {
	stdout := framed(begin("info"), "heading", end(), result(), `{"n":1}`)
	got := buildResult(stdout, "", testNonce)
	assert.Equal(t, []any{map[string]any{"n": float64(1)}}, got.Documents)
	assert.Empty(t, got.RawOutput)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "heading"}}, got.Output)
}

// A result that is not JSON (the epilogue's String() fallback) stays text.
func TestBuildResult_TextResult(t *testing.T) {
	got := buildResult(framed(begin("info"), "checking", end(), result(), "[object Foo]"), "", testNonce)
	assert.Equal(t, "[object Foo]", got.RawOutput)
	assert.Len(t, got.Output, 1)
}

// mongosh's own stderr notices come after the script's output, as a warning.
func TestBuildResult_StderrIsAWarning(t *testing.T) {
	got := buildResult(framed(result(), `[{"n":1}]`), "  deprecated thing\n", testNonce)
	assert.Len(t, got.Documents, 1)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputWarning, Text: "deprecated thing"}}, got.Output)
}

func TestWrapQuery_FramesOutputWithNonce(t *testing.T) {
	got, _ := wrapQuery("print('a'); 1", testNonce)
	first, _, _ := strings.Cut(got, "\n")
	assert.Contains(t, first, testNonce, "the preamble is one line before the body")
	assert.NotContains(t, first, "print('a')")
	assert.Contains(t, got, resultGlobal+" = 1")
}

// The preamble keeps the body starting on line 2, which remapError counts on.
func TestWrapQuery_BodyStartsOnLine2(t *testing.T) {
	got, _ := wrapQuery("db.c.find()\nprint('x')", testNonce)
	lines := strings.Split(got, "\n")
	assert.Equal(t, "db.c.find()", lines[1])
}

// On failure the printed calls are kept and mongosh's report is the error.
func TestFailedRun_SeparatesPrintedOutputFromError(t *testing.T) {
	stdout := framed(begin("info"), "checked 25", end(), "ReferenceError: foo is not defined")
	got, errMsg := failedRun(stdout, "", testNonce)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "checked 25"}}, got.Output)
	assert.Equal(t, "ReferenceError: foo is not defined", errMsg)
}

func TestFailedRun_StderrComesFirst(t *testing.T) {
	_, errMsg := failedRun(framed("from stdout"), "from stderr\n", testNonce)
	assert.Equal(t, "from stderr\nfrom stdout", errMsg)
}

// process.stdout.write leaves no newline, so the next marker shares its line.
// The marker still counts, and the text before it is kept.
func TestSplitOutput_MarkerAfterTextWithoutNewline(t *testing.T) {
	stdout := framed("working..."+begin("info"), "next", end(), "done"+result(), `[{"n":1}]`)
	out, res, hasResult := splitOutput(stdout, testNonce)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "working..."},
		{Level: models.OutputInfo, Text: "next"},
		{Level: models.OutputInfo, Text: "done"},
	}, out)
	assert.True(t, hasResult)
	assert.Equal(t, `[{"n":1}]`, res)
}

// Inside a call, text written without a newline before the end marker is
// part of that call.
func TestSplitOutput_EndMarkerAfterTextWithoutNewline(t *testing.T) {
	out, _, _ := splitOutput(framed(begin("info"), "partial"+end(), begin("warning"), "w", end()), testNonce)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "partial"},
		{Level: models.OutputWarning, Text: "w"},
	}, out)
}

// console.trace prints through print like the other console methods, so its
// message is on stdout inside its frame (mongosh's own trace adds a stack of
// transpiled line numbers that match nothing the user wrote).
func TestOutputPreamble_TracePrintsItsMessage(t *testing.T) {
	assert.Contains(t, outputPreamble(testNonce), `["trace", "info"]`)
}

func TestBuildResult_HasValue(t *testing.T) {
	assert.True(t, buildResult(framed(result(), "[]"), "", testNonce).HasValue)
	assert.False(t, buildResult(framed(begin("info"), "x", end()), "", testNonce).HasValue)
}

// A promise callback can print after the epilogue. Its frame comes after the
// result's end marker and is output, not part of the result.
func TestSplitOutput_PrintAfterTheResult(t *testing.T) {
	stdout := framed(begin("info"), "now", end(), result(), `[{"n":1}]`, end(), begin("info"), "micro", end())
	out, res, hasResult := splitOutput(stdout, testNonce)
	assert.True(t, hasResult)
	assert.Equal(t, `[{"n":1}]`, res)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "now"},
		{Level: models.OutputInfo, Text: "micro"},
	}, out)
}

// A run cut short while writing the result still has what it wrote.
func TestSplitOutput_UnendedResult(t *testing.T) {
	_, res, hasResult := splitOutput(framed(result(), `[{"n":1}]`), testNonce)
	assert.True(t, hasResult)
	assert.Equal(t, `[{"n":1}]`, res)
}

// A timed-out or cancelled run keeps all it wrote, process.stdout.write text
// and stderr included: none of it is an error report.
func TestInterruptedRun_KeepsEverything(t *testing.T) {
	got := interruptedRun(framed(begin("info"), "step 1", end(), "50%"), "notice\n", testNonce)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "step 1"},
		{Level: models.OutputInfo, Text: "50%"},
		{Level: models.OutputWarning, Text: "notice"},
	}, got.Output)
}

func TestWrapQuery_ResultHasAnEndMarker(t *testing.T) {
	got, _ := wrapQuery("1", testNonce)
	assert.Contains(t, got, `"\n" + "\x1eVV:`+testNonce+`:" + "E")`)
}
