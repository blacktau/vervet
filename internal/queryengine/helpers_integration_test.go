//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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

// skipWithoutDocumentsStage skips a test that needs $documents, which MongoDB 5.0
// refuses in user requests.
func skipWithoutDocumentsStage(t *testing.T) {
	t.Helper()
	engine := NewGojaEngine(testClient, 0, "")
	if _, err := engine.ExecuteQuery(context.Background(), testURI, dbName(t), `db.aggregate([{ $documents: [] }])`); err != nil {
		t.Skipf("server has no $documents: %v", err)
	}
}

// skipBeforeVersion skips a test that needs a server of at least major.
func skipBeforeVersion(t *testing.T, major string) {
	t.Helper()
	engine := NewGojaEngine(testClient, 0, "")
	result, err := engine.ExecuteQuery(context.Background(), testURI, dbName(t), `print(db.version())`)
	if err != nil {
		t.Fatal(err)
	}
	if version := printedText(result); version < major {
		t.Skipf("needs MongoDB %s, server is %s", major, version)
	}
}
