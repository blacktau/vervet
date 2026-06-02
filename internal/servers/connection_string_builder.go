package servers

import (
	"fmt"
	"net/url"
)

// buildOIDCURI returns rawURI with authMechanism=MONGODB-OIDC and
// authSource=$external injected into its query string. Existing values for
// either parameter are preserved (idempotent). url.Values.Encode emits the
// "$" of "$external" as "%24external".
//
// ALLOWED_HOSTS is deliberately NOT injected: it is a client-side-only OIDC
// property that the MongoDB OIDC spec forbids in connection strings, and tools
// such as Compass reject a URI that carries it. Vervet applies ALLOWED_HOSTS:*
// programmatically at connect time (internal/clientregistry/registry.go), so
// the copied URI works in external clients without it.
func buildOIDCURI(rawURI string) (string, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse OIDC URI: %w", err)
	}

	q := u.Query()
	if q.Get("authMechanism") == "" {
		q.Set("authMechanism", "MONGODB-OIDC")
	}
	if q.Get("authSource") == "" {
		q.Set("authSource", "$external")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
