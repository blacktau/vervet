package export

import (
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"vervet/internal/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock SaveDialog ---

type mockSaveDialog struct {
	path string
	err  error
}

func (m *mockSaveDialog) SaveFile(_ *string, _ *string, _ []FileFilter) (string, error) {
	return m.path, m.err
}

// --- mock fileWriter ---

type mockFileWriter struct {
	written map[string][]byte
	err     error
}

func newMockFileWriter() *mockFileWriter {
	return &mockFileWriter{written: make(map[string][]byte)}
}

func (m *mockFileWriter) WriteFile(path string, data []byte) error {
	if m.err != nil {
		return m.err
	}
	m.written[path] = data
	return nil
}

// buildTestService creates a Service with mocked dialog and writer.
func buildTestService(dialog SaveDialog, writer fileWriter) *Service {
	return &Service{
		log:        testServiceLogger(),
		dialog:     dialog,
		fileWriter: writer,
	}
}

func testServiceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestService_JSONSuccess(t *testing.T) {
	dialog := &mockSaveDialog{path: "/tmp/results.json"}
	writer := newMockFileWriter()
	svc := buildTestService(dialog, writer)

	req := api.ExportRequest{
		Format:          "json",
		EJSON:           `[{"a":1},{"a":2}]`,
		DefaultFilename: "results.json",
	}

	path, err := svc.Export(req)

	require.NoError(t, err)
	assert.Equal(t, "/tmp/results.json", path)
	assert.Contains(t, writer.written, "/tmp/results.json")
}

func TestService_CancelledDialog(t *testing.T) {
	dialog := &mockSaveDialog{path: ""}
	writer := newMockFileWriter()
	svc := buildTestService(dialog, writer)

	req := api.ExportRequest{
		Format:          "json",
		EJSON:           `[{"a":1}]`,
		DefaultFilename: "results.json",
	}

	path, err := svc.Export(req)

	require.NoError(t, err)
	assert.Equal(t, "", path)
	assert.Empty(t, writer.written)
}

func TestService_UnknownFormat(t *testing.T) {
	dialog := &mockSaveDialog{path: "/tmp/out.txt"}
	writer := newMockFileWriter()
	svc := buildTestService(dialog, writer)

	req := api.ExportRequest{
		Format: "xml",
		EJSON:  `[{"a":1}]`,
	}

	_, err := svc.Export(req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
}

func TestService_WriteErrorPropagates(t *testing.T) {
	dialog := &mockSaveDialog{path: "/tmp/results.json"}
	writer := &mockFileWriter{
		written: make(map[string][]byte),
		err:     errors.New("disk full"),
	}
	svc := buildTestService(dialog, writer)

	req := api.ExportRequest{
		Format: "json",
		EJSON:  `[{"a":1}]`,
	}

	_, err := svc.Export(req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestService_CSVTabSeparator(t *testing.T) {
	dialog := &mockSaveDialog{path: "/tmp/results.csv"}
	writer := newMockFileWriter()
	svc := buildTestService(dialog, writer)

	req := api.ExportRequest{
		Format:  "csv",
		EJSON:   `[{"a":"x","b":"y"}]`,
		Columns: []string{"a", "b"},
		CSV: &api.ExportCSVOptions{
			Separator:     "\t",
			IncludeHeader: true,
			UTF8BOM:       false,
		},
	}

	path, err := svc.Export(req)

	require.NoError(t, err)
	assert.Equal(t, "/tmp/results.csv", path)

	data := writer.written["/tmp/results.csv"]
	content := string(data)
	assert.True(t, strings.Contains(content, "a\tb"), "expected tab-separated content, got: %q", content)
}
