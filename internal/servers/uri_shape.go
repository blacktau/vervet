package servers

import "strings"

// inferURIShape returns (isCluster, isSrv) for a MongoDB connection string
// without invoking the driver's strict parser. The driver's connstring.Parse
// is unsuitable at save time because (1) it performs DNS SRV lookups for
// mongodb+srv:// URIs and (2) it rejects URIs that omit the "/" between the
// host list and the "?" query — both of which would cause "save server" to
// fail for otherwise-recoverable input.
func inferURIShape(uri string) (isCluster, isSrv bool) {
	const srvPrefix = "mongodb+srv://"
	const stdPrefix = "mongodb://"

	if len(uri) >= len(srvPrefix) && strings.EqualFold(uri[:len(srvPrefix)], srvPrefix) {
		return false, true
	}
	if len(uri) < len(stdPrefix) || !strings.EqualFold(uri[:len(stdPrefix)], stdPrefix) {
		return false, false
	}

	rest := uri[len(stdPrefix):]

	// Find the host section: everything up to the first "/" or "?" after
	// optional userinfo (which ends at "@", but only if "@" appears before
	// the path/query delimiters).
	end := len(rest)
	if i := strings.IndexAny(rest, "/?"); i >= 0 {
		end = i
	}
	hostSection := rest[:end]
	if at := strings.LastIndex(hostSection, "@"); at >= 0 {
		hostSection = hostSection[at+1:]
	}

	return strings.Contains(hostSection, ","), false
}
