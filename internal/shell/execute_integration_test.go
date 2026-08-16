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
)

var testURI string

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Disable the Ryuk reaper container — it fails under rootless Podman
	// because it needs Docker-socket-level access. The container is
	// terminated below instead.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	mongoContainer, err := mongodb.Run(ctx, "mongo:7")
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
