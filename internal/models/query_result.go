package models

// PageContext describes a paginated find() so the frontend can request
// further pages without re-running the user's script. Emitted by the Goja
// engine when a script's final value resolves to a find() lazy cursor.
//
// Filter, Projection, Sort, Hint and Collation hold canonical Extended JSON.
// The frontend treats them as opaque and hands them back unchanged; a plain
// JSON object would come back as a Go map, losing both key order (which
// MongoDB depends on for sort and hint) and BSON types such as ObjectId.
// Hint is wrapped as {"v": hint} because it may be a name or a document.
type PageContext struct {
	// Database is where the find ran, which getSiblingDB() can make differ
	// from the database the script was started on.
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter,omitempty"`
	Projection string `json:"projection,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Hint       string `json:"hint,omitempty"`
	Collation  string `json:"collation,omitempty"`
	UserLimit  int64  `json:"userLimit,omitempty"`
	UserSkip   int64  `json:"userSkip,omitempty"`
	MaxTimeMS  int64  `json:"maxTimeMS,omitempty"`
	BatchSize  int32  `json:"batchSize,omitempty"`
	Comment    string `json:"comment,omitempty"`
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
	// JSValue overrides what a script sees for operations whose display shape
	// differs from their mongosh return value — distinct (displayed as a
	// {values: [...]} document, returned to scripts as a bare array) and the
	// count methods (displayed as {count: N}, returned as a number).
	// Not serialised to the frontend.
	JSValue any `json:"-"`
}
