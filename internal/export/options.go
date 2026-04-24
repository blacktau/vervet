package export

type Format string

const (
	FormatCSV    Format = "csv"
	FormatJSON   Format = "json"
	FormatNDJSON Format = "ndjson"
)

type CSVOptions struct {
	Separator     rune
	IncludeHeader bool
	UTF8BOM       bool
}

type Options struct {
	Format  Format
	Columns []string   // optional dot-paths; nil means "derive from docs"
	CSV     CSVOptions // ignored when Format != FormatCSV
}
