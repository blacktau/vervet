package servers

import "testing"

func TestInferURIShape(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		isCluster bool
		isSrv     bool
	}{
		{"plain single host", "mongodb://localhost:27017", false, false},
		{"plain single host with path", "mongodb://localhost:27017/admin", false, false},
		{"plain cluster", "mongodb://a,b,c/db", true, false},
		{"srv URI", "mongodb+srv://cluster.example.com/", false, true},
		{"srv URI never reports cluster", "mongodb+srv://cluster.example.com/", false, true},
		{"missing slash before query (bug 257)", "mongodb://host?authMechanism=PLAIN", false, false},
		{"cluster missing slash before query", "mongodb://a,b?authMechanism=PLAIN", true, false},
		{"userinfo with @ does not count", "mongodb://user@host/db", false, false},
		{"cluster with userinfo", "mongodb://user:pass@a,b/db", true, false},
		{"unknown scheme", "https://host", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCluster, gotSrv := inferURIShape(tc.uri)
			if gotCluster != tc.isCluster || gotSrv != tc.isSrv {
				t.Errorf("inferURIShape(%q) = (%v, %v); want (%v, %v)",
					tc.uri, gotCluster, gotSrv, tc.isCluster, tc.isSrv)
			}
		})
	}
}
