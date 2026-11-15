package updates

import (
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^v?(\d{4})\.(\d{2})\.(\d+)$`)

// IsReleaseVersion reports whether v looks like a Vervet release version
// (e.g. "2026.04.4" or "v2026.04.4"). Dev builds ("dev", empty, etc.) return false.
func IsReleaseVersion(v string) bool {
	return versionPattern.MatchString(strings.TrimSpace(v))
}

// CompareVersion returns -1 if a<b, 0 if equal, 1 if a>b.
// Non-release inputs compare as 0 on both sides, -1 vs release, and 1 vs non-release.
func CompareVersion(a, b string) int {
	aParts, aOk := parseVersion(a)
	bParts, bOk := parseVersion(b)
	switch {
	case !aOk && !bOk:
		return 0
	case !aOk:
		return -1
	case !bOk:
		return 1
	}
	for i := 0; i < 3; i++ {
		if aParts[i] != bParts[i] {
			if aParts[i] < bParts[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseVersion(v string) ([3]int, bool) {
	m := versionPattern.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return [3]int{}, false
	}
	var out [3]int
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// SemverVersion returns v in the form the Wails updater compares, which is
// strict semver: "2026.08.4" becomes "2026.8.4". Semver rejects leading
// zeros, and the updater treats every invalid version as equal, so a padded
// month would hide every update from January to September. Non-release
// inputs are returned unchanged.
func SemverVersion(v string) string {
	parts, ok := parseVersion(v)
	if !ok {
		return v
	}
	return strconv.Itoa(parts[0]) + "." + strconv.Itoa(parts[1]) + "." + strconv.Itoa(parts[2])
}
