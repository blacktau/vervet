//go:build integration

package shell

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"vervet/internal/models"
	"vervet/internal/testmongo"
)

var testURI string

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Disable the Ryuk reaper container — it fails under rootless Podman
	// because it needs Docker-socket-level access. The container is
	// terminated below instead.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	mongoContainer, err := mongodb.Run(ctx, testmongo.Image())
	if err != nil {
		log.Fatalf("failed to start MongoDB container: %v", err)
	}

	testURI, err = mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	// Run the tests in a func so the container is terminated before
	// os.Exit — os.Exit runs no deferred calls.
	code := m.Run()

	if err := testcontainers.TerminateContainer(mongoContainer); err != nil {
		log.Printf("failed to terminate container: %v", err)
	}

	os.Exit(code)
}

// The whole point of writing the temp script into the user's script
// directory: mongosh takes __dirname from the file it is given, so a script
// in the system temp dir would report /tmp and load() of a sibling file
// would fail.
func TestExecute_ScriptDirDrivesDirnameAndLoad(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "helper.js"),
		[]byte(`function helperValue() { return 'from-helper'; }`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "data.csv"),
		[]byte("name,age\nalice,30\n"),
		0o644,
	))

	cfg := Config{Timeout: 60 * time.Second, ScriptDir: dir}
	query := `
		load(__dirname + '/helper.js');
		const rows = require('fs').readFileSync('data.csv', 'utf8').trim().split('\n');
		__dirname + '|' + helperValue() + '|' + rows[1]
	`

	result, err := Execute(context.Background(), testURI, query, cfg)
	require.NoError(t, err)
	// A JSON-decodable value comes back as a document, not raw text.
	assert.Equal(t, []any{dir + "|from-helper|alice,30"}, result.Documents)
}

// Without a script dir the temp file lands in the system temp dir, which is
// what an unsaved tab gets. It must still run.
func TestExecute_NoScriptDirStillRuns(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, `1 + 1`, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	// Numbers come back as canonical Extended JSON.
	assert.Equal(t, []any{map[string]any{"$numberInt": "2"}}, result.Documents)
}

// Printed lines and the result both come back, apart (#310). Before, the
// prints broke the JSON parse and the documents arrived as raw text.
func TestExecute_PrintedOutputAndResult(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `print("Top:"); console.warn("slow"); console.error("bad"); printjson({ a: 1, b: 2 }); [{ n: 1 }]`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"n": map[string]any{"$numberInt": "1"}}}, result.Documents)
	assert.Empty(t, result.RawOutput)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "Top:"},
		{Level: models.OutputWarning, Text: "slow"},
		{Level: models.OutputError, Text: "bad"},
		{Level: models.OutputInfo, Text: "{\n  a: 1,\n  b: 2\n}"},
	}, result.Output)
}

func TestExecute_PrintOnlyHasNoResult(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, `print("done")`, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Empty(t, result.Documents)
	assert.Empty(t, result.RawOutput)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "done"}}, result.Output)
}

// A failing script's prints come back in Output and stay out of the error.
func TestExecute_ErrorKeepsPrintedOutputSeparate(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := "print(\"checked 25\");\nthrow new Error(\"Wrong database\")"
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Wrong database")
	assert.NotContains(t, err.Error(), "checked 25")
	assert.NotContains(t, err.Error(), "\x1e")
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "checked 25"}}, result.Output)
}

// Syntax error positions point at the user's own lines, not the wrapper's.
func TestExecute_SyntaxErrorLineIsTheUsersLine(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := "print(\"a\");\nlet x = (;"
	_, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "> 2 | let x = (;")
}

// A script that times out still returns what it printed: the progress lines
// show where it stalled.
func TestExecute_TimeoutKeepsPrintedOutput(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `print("step 1"); sleep(20000); print("step 2")`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 3 * time.Second})
	require.ErrorIs(t, err, ErrQueryTimeout)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "step 1"}}, result.Output)
}

// process.stdout.write leaves the line open; the result must still be found.
func TestExecute_StdoutWriteWithoutNewline(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `process.stdout.write("working..."); [{ n: 1 }]`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"n": map[string]any{"$numberInt": "1"}}}, result.Documents)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "working..."}}, result.Output)
}

// A print from a promise callback runs after the epilogue. It goes to Output,
// and the result still parses.
func TestExecute_PrintAfterTheResult(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `Promise.resolve().then(() => print("late")); [{ n: 1 }]`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"n": map[string]any{"$numberInt": "1"}}}, result.Documents)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "late"}}, result.Output)
}

// A syntax error before the capture prefix keeps its column, caret included.
func TestExecute_SyntaxErrorBeforeTheCapturePrefix(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	_, err := Execute(context.Background(), testURI, "x = {a:,}; 5", Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "(1:7)")
	assert.Contains(t, err.Error(), "> 1 | x = {a:,}; 5\n    |        ^")
}

// mongosh reports a runtime error with no position or stack, so there is no
// wrapper line number in it to get wrong.
func TestExecute_RuntimeErrorHasNoWrapperPosition(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := "print(1)\nprint(2)\nthrow new Error(\"boom\")"
	_, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Equal(t, "Error: boom", err.Error())
}

// console.trace prints its message inside its frame, on stdout.
func TestExecute_ConsoleTrace(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, `console.trace("here"); 1`, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "here"}}, result.Output)
}

// A timeout keeps text written outside any print call too.
func TestExecute_TimeoutKeepsStdoutWrite(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `process.stdout.write("50%\n"); sleep(20000)`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 3 * time.Second})
	require.ErrorIs(t, err, ErrQueryTimeout)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "50%"}}, result.Output)
}

// A promise callback runs between the script's statements. Its print is an
// entry of its own, in the order it ran, not merged into another call's.
func TestExecute_PromisePrintIsItsOwnEntry(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `Promise.resolve().then(() => console.warn("late")); print("a"); printjson({ x: 1 }); 1`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "a"},
		{Level: models.OutputWarning, Text: "late"},
		{Level: models.OutputInfo, Text: "{\n  x: 1\n}"},
	}, result.Output)
}

// A cursor read to the end leaves no value, as in the builtin engine, so the
// UI opens Messages instead of an empty Results tab. A cursor that has not run
// is a value even when it matches nothing, and a part-read one shows the rest.
func TestExecute_UsedUpCursorHasNoValue(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	setup := `db.usedup.drop(); db.usedup.insertMany([{ n: 1 }, { n: 2 }]); `
	for _, tc := range []struct {
		script   string
		hasValue bool
		docs     int
	}{
		{`const c = db.usedup.find(); while (c.hasNext()) { printjson(c.next()); } c`, false, 0},
		{`const c = db.usedup.find(); c.forEach(d => print(d.n)); c`, false, 0},
		{`const c = db.usedup.find({ n: 99 }); c`, true, 0},
		{`const c = db.usedup.find(); c.next(); c`, true, 1},
	} {
		result, err := Execute(context.Background(), testURI, setup+tc.script, Config{Timeout: 60 * time.Second})
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.hasValue, result.HasValue, tc.script)
		assert.Len(t, result.Documents, tc.docs, tc.script)
	}
}

// The console methods the preamble doesn't wrap by name go through print or
// console.warn, so each is still its own entry, in order.
func TestExecute_OtherConsoleMethodsAreFramed(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `console.dir({ d: 1 }); console.count("c"); console.assert(false, "nope"); console.log("end")`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []models.OutputLine{
		{Level: models.OutputInfo, Text: "{ d: 1 }"},
		{Level: models.OutputInfo, Text: "c: 1"},
		{Level: models.OutputWarning, Text: "Assertion failed: nope"},
		{Level: models.OutputInfo, Text: "end"},
	}, result.Output)
}

// On a failure, text written outside print calls is output, not part of the error.
func TestExecute_FailureKeepsStdoutWriteOutOfTheError(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `process.stdout.write("note\n"); throw new Error("boom")`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Equal(t, "Error: boom", err.Error())
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "note"}}, result.Output)
}

// mongosh formats a cursor or insert result after print returns. All of it
// stays in the one entry, at its level, and the next statement comes after.
func TestExecute_PrintingAsyncValuesIsOneEntry(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `db.asyncprint.drop(); print(db.asyncprint.find()); print("next"); ` +
		`console.error(db.asyncprint.insertOne({ _id: 1 })); print("after"); 1`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	require.Len(t, result.Output, 4, "%+v", result.Output)
	assert.Equal(t, models.OutputLine{Level: models.OutputInfo, Text: "next"}, result.Output[1])
	assert.Equal(t, models.OutputError, result.Output[2].Level)
	assert.Contains(t, result.Output[2].Text, "acknowledged: true")
	assert.Equal(t, models.OutputLine{Level: models.OutputInfo, Text: "after"}, result.Output[3])
}

// Calls from callbacks and loops stay in the order they ran.
func TestExecute_PrintOrderInLoopsAndCallbacks(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	query := `[1, 2].forEach(n => print(n)); for (let i = 3; i < 5; i++) { print(i); } print("end")`
	result, err := Execute(context.Background(), testURI, query, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	var got []string
	for _, l := range result.Output {
		got = append(got, l.Text)
	}
	assert.Equal(t, []string{"1", "2", "3", "4", "end"}, got)
}

// What was printed before quit() is not lost in the queue.
func TestExecute_PrintBeforeQuit(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, `print("bye"); quit()`, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "bye"}}, result.Output)
}

// A saved script's hashbang is fine, and lines keep the user's numbers.
func TestExecute_Hashbang(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, "#!/usr/bin/env mongosh\nprint(\"a\");\n[{ n: 1 }]", Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Len(t, result.Documents, 1)
	assert.Equal(t, []models.OutputLine{{Level: models.OutputInfo, Text: "a"}}, result.Output)

	_, err = Execute(context.Background(), testURI, "#!/usr/bin/env mongosh\nlet x = (;", Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "> 2 | let x = (;")
}

func TestExecute_UnclosedBlockPointsAtTheEndOfTheScript(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	_, err := Execute(context.Background(), testURI, "function f() {\n  print(1)", Config{Timeout: 60 * time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "(2:10)")
	assert.NotContains(t, err.Error(), "^")
	assert.NotContains(t, err.Error(), "__vervet")
}

// findOne with no match shows null, as in the builtin engine.
func TestExecute_FindOneNoMatchIsNull(t *testing.T) {
	if !CheckMongosh() {
		t.Skip("mongosh not in PATH")
	}

	result, err := Execute(context.Background(), testURI, `db.nomatch.findOne({ _id: 1 })`, Config{Timeout: 60 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, "null", result.RawOutput)
	assert.True(t, result.HasValue)
}
