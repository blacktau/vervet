package queryexecutor

import (
	"context"
	"testing"
)

func newTestExecutor() *QueryExecutor {
	return &QueryExecutor{
		ctx:     context.Background(),
		cancels: make(map[queryKey]context.CancelFunc),
	}
}

// Two queries against the same server must not cancel each other:
// registering a second query for a server leaves the first running.
func TestRegisterQuery_ConcurrentSameServerDoNotCancelEachOther(t *testing.T) {
	qe := newTestExecutor()

	cancelled1 := false
	cancelled2 := false
	qe.registerQuery("srv", "q1", func() { cancelled1 = true })
	qe.registerQuery("srv", "q2", func() { cancelled2 = true })

	if cancelled1 {
		t.Fatal("registering q2 must not cancel q1 on the same server")
	}
	if cancelled2 {
		t.Fatal("q2 must not be cancelled on registration")
	}
}

// CancelQuery targets exactly one (serverID, queryID) pair.
func TestCancelQuery_CancelsOnlyTargetedQuery(t *testing.T) {
	qe := newTestExecutor()

	cancelled1 := false
	cancelled2 := false
	qe.registerQuery("srv", "q1", func() { cancelled1 = true })
	qe.registerQuery("srv", "q2", func() { cancelled2 = true })

	qe.CancelQuery("srv", "q1")

	if !cancelled1 {
		t.Fatal("q1 should have been cancelled")
	}
	if cancelled2 {
		t.Fatal("q2 should not have been cancelled")
	}
}

// CloseAll cancels every in-flight query across servers.
func TestCloseAll_CancelsEveryQuery(t *testing.T) {
	qe := newTestExecutor()

	cancelled := make(map[string]bool)
	qe.registerQuery("srvA", "q1", func() { cancelled["a1"] = true })
	qe.registerQuery("srvA", "q2", func() { cancelled["a2"] = true })
	qe.registerQuery("srvB", "q1", func() { cancelled["b1"] = true })

	qe.CloseAll()

	for _, k := range []string{"a1", "a2", "b1"} {
		if !cancelled[k] {
			t.Fatalf("expected %s to be cancelled", k)
		}
	}
}

func TestAppendDatabase(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		dbName string
		want   string
	}{
		{
			name:   "no existing db",
			uri:    "mongodb://localhost:27017",
			dbName: "mydb",
			want:   "mongodb://localhost:27017/mydb",
		},
		{
			name:   "trailing slash",
			uri:    "mongodb://localhost:27017/",
			dbName: "mydb",
			want:   "mongodb://localhost:27017/mydb",
		},
		{
			name:   "existing db is replaced",
			uri:    "mongodb://localhost:27017/olddb",
			dbName: "newdb",
			want:   "mongodb://localhost:27017/newdb",
		},
		{
			name:   "existing db with query params",
			uri:    "mongodb://localhost:27017/olddb?authSource=admin",
			dbName: "newdb",
			want:   "mongodb://localhost:27017/newdb?authSource=admin",
		},
		{
			name:   "no existing db with query params",
			uri:    "mongodb://localhost:27017?replicaSet=rs0",
			dbName: "mydb",
			want:   "mongodb://localhost:27017/mydb?replicaSet=rs0",
		},
		{
			name:   "srv scheme",
			uri:    "mongodb+srv://cluster.example.com/olddb",
			dbName: "newdb",
			want:   "mongodb+srv://cluster.example.com/newdb",
		},
		{
			name:   "srv no existing db",
			uri:    "mongodb+srv://cluster.example.com",
			dbName: "mydb",
			want:   "mongodb+srv://cluster.example.com/mydb",
		},
		{
			name:   "uri with credentials",
			uri:    "mongodb://user:pass@host:27017/olddb",
			dbName: "newdb",
			want:   "mongodb://user:pass@host:27017/newdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendDatabase(tt.uri, tt.dbName)
			if got != tt.want {
				t.Errorf("appendDatabase(%q, %q) = %q, want %q", tt.uri, tt.dbName, got, tt.want)
			}
		})
	}
}

func TestValidDBName(t *testing.T) {
	valid := []string{"mydb", "my_db", "my-db", "my.db", "DB123"}
	for _, name := range valid {
		if !validDBName.MatchString(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "my db", "db/path", "db?query", "db&x=1", "../etc", "db;drop"}
	for _, name := range invalid {
		if validDBName.MatchString(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
