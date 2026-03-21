package connectionStrings

import (
	"testing"

	"vervet/internal/models"
)

func TestSerialiseConnectionConfig(t *testing.T) {
	cfg := models.ConnectionConfig{
		URI:        "mongodb://host:27017",
		AuthMethod: models.AuthOIDC,
		OIDCConfig: &models.OIDCConfig{
			ProviderURL: "https://accounts.google.com",
			ClientID:    "my-client-id",
			Scopes:      []string{"openid", "profile"},
		},
		RefreshToken: "refresh-tok",
	}

	data, err := serialiseConnectionConfig(cfg)
	if err != nil {
		t.Fatalf("serialise: %v", err)
	}

	got, err := deserialiseConnectionConfig(data)
	if err != nil {
		t.Fatalf("deserialise: %v", err)
	}

	if got.URI != cfg.URI {
		t.Errorf("URI = %q, want %q", got.URI, cfg.URI)
	}
	if got.AuthMethod != cfg.AuthMethod {
		t.Errorf("AuthMethod = %q, want %q", got.AuthMethod, cfg.AuthMethod)
	}
	if got.OIDCConfig == nil {
		t.Fatal("OIDCConfig is nil")
	}
	if got.OIDCConfig.ProviderURL != cfg.OIDCConfig.ProviderURL {
		t.Errorf("ProviderURL = %q, want %q", got.OIDCConfig.ProviderURL, cfg.OIDCConfig.ProviderURL)
	}
	if got.RefreshToken != cfg.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, cfg.RefreshToken)
	}
}

func TestDeserialiseRawURI(t *testing.T) {
	raw := "mongodb://user:pass@host:27017/admin"

	cfg, err := deserialiseConnectionConfig(raw)
	if err != nil {
		t.Fatalf("deserialise raw URI: %v", err)
	}

	if cfg.URI != raw {
		t.Errorf("URI = %q, want %q", cfg.URI, raw)
	}
	if cfg.AuthMethod != models.AuthPassword {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, models.AuthPassword)
	}
}
