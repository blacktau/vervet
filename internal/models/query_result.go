package models

// PageContext describes a paginated find() so the frontend can request
// further pages without re-running the user's script. Emitted by the Goja
// engine when a script's final value resolves to a find() lazy cursor.
type PageContext struct {
	Collection string         `json:"collection"`
	Filter     any            `json:"filter,omitempty"`
	Projection any            `json:"projection,omitempty"`
	Sort       any            `json:"sort,omitempty"`
	Hint       any            `json:"hint,omitempty"`
	Collation  map[string]any `json:"collation,omitempty"`
	UserLimit  int64          `json:"userLimit,omitempty"`
	UserSkip   int64          `json:"userSkip,omitempty"`
	MaxTimeMS  int64          `json:"maxTimeMS,omitempty"`
	Comment    string         `json:"comment,omitempty"`
}

// QueryResult holds the parsed output of a query.
// Documents contains structured data when the engine returns documents.
// RawOutput contains raw text for non-JSON results (e.g. db.stats()).
type QueryResult struct {
	Documents     []any        `json:"documents"`
	RawOutput     string       `json:"rawOutput"`
	OperationType string       `json:"operationType,omitempty"`
	AffectedCount int          `json:"affectedCount,omitempty"`
	PageContext   *PageContext `json:"pageContext,omitempty"`
	// Single marks results that semantically represent one object (write acks,
	// counts, findOneAnd* matches, explain output) rather than a document list.
	// Consumed by the Goja engine so scripts see `result.insertedIds` instead
	// of `result[0].insertedIds`. Not serialised to the frontend.
	Single bool `json:"-"`
}
