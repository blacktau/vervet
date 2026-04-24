package api

import (
	"log/slog"
)

// ExportCSVOptions is the CSV-specific sub-options from the frontend.
type ExportCSVOptions struct {
	Separator     string `json:"separator"` // may be "\t" — first rune wins
	IncludeHeader bool   `json:"includeHeader"`
	UTF8BOM       bool   `json:"utf8Bom"`
}

// ExportRequest is the request shape sent by the frontend for an export operation.
type ExportRequest struct {
	Format          string            `json:"format"` // "csv" | "json" | "ndjson"
	EJSON           string            `json:"ejson"`
	CollectionName  string            `json:"collectionName"`
	DefaultFilename string            `json:"defaultFilename"`
	CSV             *ExportCSVOptions `json:"csv,omitempty"`
	Columns         []string          `json:"columns,omitempty"`
}

// ExportProvider is the interface ExportProxy depends on.
type ExportProvider interface {
	Export(req ExportRequest) (string, error)
}

// ExportProxy is the Wails-bound proxy for export operations.
type ExportProxy struct {
	log      *slog.Logger
	provider ExportProvider
}

// NewExportProxy constructs an ExportProxy.
func NewExportProxy(log *slog.Logger, provider ExportProvider) *ExportProxy {
	return &ExportProxy{
		log:      log,
		provider: provider,
	}
}

// ExportResults serializes docs and writes them to a user-chosen file.
// Returns the written path on success; empty string if the user cancelled.
func (ep *ExportProxy) ExportResults(req ExportRequest) Result[string] {
	path, err := ep.provider.Export(req)
	if err != nil {
		logFail(ep.log, "ExportResults", err)
		return FailResult[string](err)
	}

	return SuccessResult(path)
}
