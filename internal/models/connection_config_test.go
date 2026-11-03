package models

import "testing"

func TestTokenBinding(t *testing.T) {
	base := func() ConnectionConfig {
		return ConnectionConfig{
			URI:        "mongodb://host",
			AuthMethod: AuthOIDC,
			OIDCConfig: &OIDCConfig{ProviderURL: "https://idp", ClientID: "vervet", Scopes: []string{"a", "b"}},
		}
	}

	changed := map[string]func(*ConnectionConfig){
		"uri":         func(c *ConnectionConfig) { c.URI = "mongodb://other" },
		"auth method": func(c *ConnectionConfig) { c.AuthMethod = AuthPassword },
		"provider":    func(c *ConnectionConfig) { c.OIDCConfig.ProviderURL = "https://other" },
		"client":      func(c *ConnectionConfig) { c.OIDCConfig.ClientID = "other" },
		"scopes":      func(c *ConnectionConfig) { c.OIDCConfig.Scopes = []string{"a"} },
	}
	for name, change := range changed {
		t.Run("changes with "+name, func(t *testing.T) {
			cfg := base()
			change(&cfg)
			if cfg.TokenBinding() == base().TokenBinding() {
				t.Error("expected a different binding")
			}
		})
	}

	unchanged := map[string]func(*ConnectionConfig){
		"prompt":        func(c *ConnectionConfig) { c.OIDCConfig.Prompt = "login" },
		"manual url":    func(c *ConnectionConfig) { c.OIDCConfig.ManualURLMode = true },
		"refresh token": func(c *ConnectionConfig) { c.RefreshToken = "token" },
		"scope order":   func(c *ConnectionConfig) { c.OIDCConfig.Scopes = []string{"b", "a"} },
	}
	for name, change := range unchanged {
		t.Run("ignores "+name, func(t *testing.T) {
			cfg := base()
			change(&cfg)
			if cfg.TokenBinding() != base().TokenBinding() {
				t.Error("expected the same binding")
			}
		})
	}

	// Scopes come back nil after a JSON round trip through the keyring, while the
	// frontend sends an empty list. That mustn't look like a change.
	t.Run("treats nil and empty scopes alike", func(t *testing.T) {
		withNil := ConnectionConfig{OIDCConfig: &OIDCConfig{}}
		withEmpty := ConnectionConfig{OIDCConfig: &OIDCConfig{Scopes: []string{}}}
		if withNil.TokenBinding() != withEmpty.TokenBinding() {
			t.Error("expected the same binding")
		}
	})
}
