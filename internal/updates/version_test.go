package updates

import "testing"

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2026.04.4", "2026.04.4", 0},
		{"v2026.04.4", "2026.04.4", 0},
		{"2026.04.4", "2026.04.5", -1},
		{"2026.05.1", "2026.04.9", 1},
		{"2027.01.0", "2026.12.9", 1},
		{"v2026.04.5", "2026.04.4", 1},
	}
	for _, c := range cases {
		if got := CompareVersion(c.a, c.b); got != c.want {
			t.Errorf("CompareVersion(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsReleaseVersion(t *testing.T) {
	cases := map[string]bool{
		"2026.04.4":  true,
		"v2026.04.4": true,
		"dev":        false,
		"":           false,
		"2026.4.4":   false,
		"2026.04":    false,
	}
	for v, want := range cases {
		if got := IsReleaseVersion(v); got != want {
			t.Errorf("IsReleaseVersion(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestSemverVersion(t *testing.T) {
	cases := map[string]string{
		"2026.08.4":  "2026.8.4",
		"v2026.11.1": "2026.11.1",
		"2026.01.10": "2026.1.10",
		"dev":        "dev",
		"":           "",
	}
	for in, want := range cases {
		if got := SemverVersion(in); got != want {
			t.Errorf("SemverVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
