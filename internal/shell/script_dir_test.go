package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The temp script must sit beside the user's saved script: mongosh derives
// __dirname from the file it runs, so a script in the system temp dir would
// report /tmp and break load(__dirname + '/helper.js').
func TestWriteQueryFile_LandsInScriptDir(t *testing.T) {
	dir := t.TempDir()

	path, cleanup, err := writeQueryFile("print(1)", dir)
	require.NoError(t, err)
	defer cleanup()

	assert.Equal(t, dir, filepath.Dir(path))
	assert.True(t, strings.HasSuffix(path, ".js"), "expected a .js file, got %s", path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "print(1)", string(data))
}

func TestWriteQueryFile_EmptyDirUsesTempDir(t *testing.T) {
	path, cleanup, err := writeQueryFile("print(1)", "")
	require.NoError(t, err)
	defer cleanup()

	assert.Equal(t, filepath.Clean(os.TempDir()), filepath.Dir(path))
}

func TestWriteQueryFile_CleanupRemovesFile(t *testing.T) {
	dir := t.TempDir()

	path, cleanup, err := writeQueryFile("print(1)", dir)
	require.NoError(t, err)

	cleanup()
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
