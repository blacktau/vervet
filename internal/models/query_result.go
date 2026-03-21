package models

// QueryResult holds the parsed output of a mongosh query.
// Documents contains structured JSON data when mongosh returns JSON output.
// RawOutput contains the raw text for non-JSON results (e.g. db.stats()).
type QueryResult struct {
	Documents     []any  `json:"documents"`
	RawOutput     string `json:"rawOutput"`
	OperationType string `json:"operationType,omitempty"`
	AffectedCount int    `json:"affectedCount,omitempty"`
}
