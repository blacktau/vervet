//go:build integration

package queryengine

import (
	"encoding/json"

	"vervet/internal/models"
)

// resultText returns a searchable string view of a query result. Scalar and
// string results live in RawOutput; objects and arrays (e.g. db.runCommand,
// db.getUsers, db.getCollectionNames) are structured and land in Documents
// instead, so fall back to their JSON encoding.
func resultText(r models.QueryResult) string {
	if r.RawOutput != "" {
		return r.RawOutput
	}
	b, _ := json.Marshal(r.Documents)
	return string(b)
}
