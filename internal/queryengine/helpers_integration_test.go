//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"strconv"
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

// skipBeforeMajor skips a test that needs a server of at least the given major
// version, for features newer than the oldest supported MongoDB.
func skipBeforeMajor(t *testing.T, major int) {
	t.Helper()
	engine := NewGojaEngine(testClient, 0, "")
	result, err := engine.ExecuteQuery(context.Background(), testURI, dbName(t), `print(db.version())`)
	if err != nil {
		t.Fatal(err)
	}
	version := printedText(result)
	got, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil {
		t.Fatalf("unreadable server version %q: %v", version, err)
	}
	if got < major {
		t.Skipf("needs MongoDB %d, server is %s", major, version)
	}
}
