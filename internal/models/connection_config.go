package models

type AuthMethod string

const (
	AuthNone     AuthMethod = "none"
	AuthPassword AuthMethod = "password"
	AuthX509     AuthMethod = "x509"
	AuthOIDC     AuthMethod = "oidc"
	AuthAWS      AuthMethod = "aws"
)

type OIDCConfig struct {
	ProviderURL      string   `json:"providerUrl"`
	ClientID         string   `json:"clientId"`
	Scopes           []string `json:"scopes,omitempty"`
	WorkloadIdentity bool     `json:"workloadIdentity"`
}

type ConnectionConfig struct {
	URI          string      `json:"uri"`
	AuthMethod   AuthMethod  `json:"authMethod"`
	OIDCConfig   *OIDCConfig `json:"oidcConfig,omitempty"`
	RefreshToken string      `json:"refreshToken,omitempty"`
}
