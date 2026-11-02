package export

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"vervet/internal/api"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FileFilter describes a file type filter for the save dialog.
type FileFilter struct {
	DisplayName string
	Pattern     string
}

// SaveDialog is the interface used to show an OS save-file dialog.
type SaveDialog interface {
	SaveFile(title *string, name *string, filters []FileFilter) (string, error)
}

// fileWriter is the interface used to write bytes to a file path.
type fileWriter interface {
	WriteFile(path string, data []byte) error
}

// osFileWriter wraps os.WriteFile with the fileWriter interface.
type osFileWriter struct{}

func (w *osFileWriter) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// Service orchestrates export: serialize docs then save via dialog.
type Service struct {
	log        *slog.Logger
	dialog     SaveDialog
	fileWriter fileWriter
}

// NewService constructs a production Service. The files.Service satisfies
// SaveDialog via an adapter because its SaveFile uses api.FileFilter.
func NewService(log *slog.Logger, dialog SaveDialog) *Service {
	return &Service{
		log:        log.With(slog.String("source", "ExportService")),
		dialog:     dialog,
		fileWriter: &osFileWriter{},
	}
}

// Init satisfies the service lifecycle interface. ExportService has no
// context-dependent state of its own; the underlying dialog is initialised by
// files.Service independently.
func (s *Service) Init(_ context.Context) {
}

// Export serializes docs from req.EJSON and writes them to a user-chosen path.
// Returns the path written (empty string if user cancelled).
func (s *Service) Export(req api.ExportRequest) (string, error) {
	docs, err := parseEJSON(req.EJSON)
	if err != nil {
		return "", fmt.Errorf("failed to parse EJSON: %w", err)
	}

	opts, err := buildOptions(req)
	if err != nil {
		return "", err
	}

	data, err := Serialize(docs, opts)
	if err != nil {
		return "", fmt.Errorf("failed to serialize: %w", err)
	}

	filters := filtersFor(opts.Format)
	title := "Export results"
	defaultFilename := req.DefaultFilename

	path, err := s.dialog.SaveFile(&title, &defaultFilename, filters)
	if err != nil {
		return "", fmt.Errorf("failed to open save dialog: %w", err)
	}

	if path == "" {
		return "", nil
	}

	if err := s.fileWriter.WriteFile(path, data); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return path, nil
}

// nonDocumentField holds a result that isn't a document, such as a string from
// db.getCollectionNames(), so every exported row is still a document.
const nonDocumentField = "value"

// parseEJSON decodes a JSON array of EJSON-encoded results into []bson.M. A
// result that isn't a document is placed under nonDocumentField.
func parseEJSON(raw string) ([]bson.M, error) {
	var rawDocs []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawDocs); err != nil {
		return nil, err
	}

	docs := make([]bson.M, len(rawDocs))
	for i, r := range rawDocs {
		d, err := parseResult(r)
		if err != nil {
			return nil, fmt.Errorf("doc[%d]: %w", i, err)
		}
		docs[i] = d
	}

	return docs, nil
}

func parseResult(raw json.RawMessage) (bson.M, error) {
	// A top-level EJSON value can't decode into a bson.M, so read it as a field.
	wrapped, err := json.Marshal(map[string]json.RawMessage{nonDocumentField: raw})
	if err != nil {
		return nil, err
	}

	var result bson.M
	if err := bson.UnmarshalExtJSON(wrapped, false, &result); err != nil {
		return nil, err
	}

	if _, isDocument := result[nonDocumentField].(bson.D); !isDocument {
		return result, nil
	}

	var doc bson.M
	if err := bson.UnmarshalExtJSON(raw, false, &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// buildOptions maps the api.ExportRequest fields to export.Options.
func buildOptions(req api.ExportRequest) (Options, error) {
	var format Format
	switch req.Format {
	case "csv":
		format = FormatCSV
	case "json":
		format = FormatJSON
	case "ndjson":
		format = FormatNDJSON
	default:
		return Options{}, fmt.Errorf("unknown format %q", req.Format)
	}

	opts := Options{
		Format:  format,
		Columns: req.Columns,
	}

	if req.CSV != nil {
		sep := ','
		if req.CSV.Separator != "" {
			runes := []rune(req.CSV.Separator)
			sep = runes[0]
		}
		opts.CSV = CSVOptions{
			Separator:     sep,
			IncludeHeader: req.CSV.IncludeHeader,
			UTF8BOM:       req.CSV.UTF8BOM,
		}
	}

	return opts, nil
}

// filtersFor returns FileFilter entries appropriate for the given format.
func filtersFor(f Format) []FileFilter {
	switch f {
	case FormatCSV:
		return []FileFilter{{DisplayName: "CSV files (*.csv)", Pattern: "*.csv"}}
	case FormatJSON:
		return []FileFilter{{DisplayName: "JSON files (*.json)", Pattern: "*.json"}}
	case FormatNDJSON:
		return []FileFilter{{DisplayName: "NDJSON files (*.ndjson)", Pattern: "*.ndjson"}}
	default:
		return nil
	}
}
