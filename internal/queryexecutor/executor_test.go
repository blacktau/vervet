package queryexecutor

import "testing"

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
