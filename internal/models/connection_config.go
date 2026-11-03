package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
)

type AuthMethod string

const (
	AuthNone     AuthMethod = "none"
	AuthPassword AuthMethod = "password"
	AuthX509     AuthMethod = "x509"
	AuthOIDC     AuthMethod = "oidc"
	AuthAWS      AuthMethod = "aws"
	AuthGSSAPI   AuthMethod = "gssapi"
	AuthPLAIN    AuthMethod = "plain"
)

type OIDCConfig struct {
	ProviderURL      string   `json:"providerUrl"`
	ClientID         string   `json:"clientId"`
	Scopes           []string `json:"scopes,omitempty"`
	WorkloadIdentity bool     `json:"workloadIdentity"`
	Prompt           string   `json:"prompt,omitempty"`
	ManualURLMode    bool     `json:"manualUrlMode,omitempty"`
}

type ConnectionConfig struct {
	URI          string      `json:"uri"`
	AuthMethod   AuthMethod  `json:"authMethod"`
	OIDCConfig   *OIDCConfig `json:"oidcConfig,omitempty"`
	RefreshToken string      `json:"refreshToken,omitempty"`
}

// TokenBinding identifies the details an OIDC token was issued for. A token may
// only be used, kept or stored while its binding matches the current config, so
// it can't reach a host or provider the user has since changed. The prompt is
// included so that choosing to force the account picker takes effect, rather than
// refreshing silently into the old account.
func (c ConnectionConfig) TokenBinding() string {
	details := struct {
		URI         string
		AuthMethod  AuthMethod
		ProviderURL string
		ClientID    string
		Scopes      []string
		Prompt      string
	}{URI: c.URI, AuthMethod: c.AuthMethod, Scopes: []string{}}
	if c.OIDCConfig != nil {
		details.ProviderURL = c.OIDCConfig.ProviderURL
		details.ClientID = c.OIDCConfig.ClientID
		details.Prompt = c.OIDCConfig.Prompt
		details.Scopes = append(details.Scopes, c.OIDCConfig.Scopes...)
		slices.Sort(details.Scopes)
	}
	// Marshalling strings and string slices can't fail.
	data, _ := json.Marshal(details)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
