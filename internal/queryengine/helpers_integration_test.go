//go:build integration

package queryengine

import (
	"encoding/json"
	"strings"

	"vervet/internal/models"
)

// resultText returns a searchable string view of a query result. Scalar and
// string results live in RawOutput; objects and arrays (e.g. db.runCommand,
// db.getUsers, db.getCollectionNames) are structured and land in Documents
// instead, so fall back to their JSON encoding.
// printedText joins what a script printed, one call per line.
func printedText(r models.QueryResult) string {
	texts := make([]string, len(r.Output))
	for i, l := range r.Output {
		texts[i] = l.Text
	}
	return strings.Join(texts, "\n")
}

func resultText(r models.QueryResult) string {
	if r.RawOutput != "" {
		return r.RawOutput
	}
	b, _ := json.Marshal(r.Documents)
	return string(b)
}
