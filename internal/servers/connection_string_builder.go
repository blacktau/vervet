package servers

import (
	"fmt"
	"net/url"
)

// buildOIDCURI returns rawURI with authMechanism=MONGODB-OIDC and
// authMechanismProperties=ALLOWED_HOSTS:* injected into its query string.
// Existing values for either parameter are preserved (idempotent).
//
// The defaults match what internal/clientregistry/registry.go applies at
// connect time, so the returned URI is usable as-is in mongosh.
func buildOIDCURI(rawURI string) (string, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse OIDC URI: %w", err)
	}

	q := u.Query()
	if q.Get("authMechanism") == "" {
		q.Set("authMechanism", "MONGODB-OIDC")
	}
	if q.Get("authMechanismProperties") == "" {
		q.Set("authMechanismProperties", "ALLOWED_HOSTS:*")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
