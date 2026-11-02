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
	"go.mongodb.org/mongo-driver/v2/bson"
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

func TestParseEJSON_ValuesThatAreNotDocuments(t *testing.T) {
	oid, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.NoError(t, err)

	docs, err := parseEJSON(`["theaters", 7, 2.5, true, null, {"$oid":"507f1f77bcf86cd799439011"}, {"$numberLong":"9007199254740993"}, [1, 2]]`)
	require.NoError(t, err)

	values := make([]any, len(docs))
	for i, d := range docs {
		require.Len(t, d, 1)
		values[i] = d[nonDocumentField]
	}
	require.Equal(t, []any{
		"theaters", int32(7), 2.5, true, nil, oid, int64(9007199254740993), bson.A{int32(1), int32(2)},
	}, values)
}

func TestParseEJSON_DocumentsAreUnchanged(t *testing.T) {
	docs, err := parseEJSON(`[{"a":1},{},{"value":"x"}]`)
	require.NoError(t, err)
	require.Equal(t, []bson.M{{"a": int32(1)}, {}, {"value": "x"}}, docs)
}

func TestSerialize_ValuesThatAreNotDocuments(t *testing.T) {
	const input = `["theaters", null, {"a":1}]`

	tests := []struct {
		name string
		opts Options
		want string
	}{
		{
			name: "csv",
			opts: Options{Format: FormatCSV, CSV: CSVOptions{IncludeHeader: true}},
			want: "a,value\n,theaters\n,\n1,\n",
		},
		{
			name: "ndjson",
			opts: Options{Format: FormatNDJSON},
			want: `{"value":"theaters"}` + "\n" + `{"value":null}` + "\n" + `{"a":1}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs, err := parseEJSON(input)
			require.NoError(t, err)

			data, err := Serialize(docs, tt.opts)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(data))
		})
	}
}
