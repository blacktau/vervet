package shell

import (
	"context"

	"vervet/internal/models"
)

// ExecuteWithOIDC runs a mongosh query using the OIDC auth-code flow.
// This uses the fallback approach: mongosh handles its own OIDC browser flow.
// Since the user already authenticated for the Go driver, the OIDC provider's
// session cookie is still active — mongosh's browser redirect completes
// instantly without user interaction.
func ExecuteWithOIDC(ctx context.Context, uri string, query string, cfg Config) (models.QueryResult, error) {
	return runMongosh(ctx, query, cfg, func(queryFile string) []string {
		return []string{
			uri,
			"--quiet",
			"--norc",
			"--authenticationMechanism", "MONGODB-OIDC",
			"--oidcFlows", "auth-code",
			"--file", queryFile,
		}
	})
}
