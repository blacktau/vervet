package connectionStrings

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"vervet/internal/models"
)

func newTestStore() *store {
	return NewStore(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestWithTimeoutLateReturnClearsUnavailable(t *testing.T) {
	s := newTestStore()

	// fn takes longer than the timeout: cooldown should be set.
	start := make(chan struct{})
	err := s.withTimeout(20*time.Millisecond, func() error {
		<-start
		return nil
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	s.mu.Lock()
	if s.keyringAvailable {
		s.mu.Unlock()
		t.Fatal("expected keyring marked unavailable after timeout")
	}
	s.mu.Unlock()

	// Allow the goroutine to finish; it should clear the unavailable flag.
	close(start)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		available := s.keyringAvailable
		s.mu.Unlock()
		if available {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("late goroutine return did not clear keyringAvailable")
}

func TestWithTimeoutCooldownFailsFast(t *testing.T) {
	s := newTestStore()
	s.markKeyringUnavailable()

	called := false
	err := s.withTimeout(time.Second, func() error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected error during cooldown")
	}
	if called {
		t.Fatal("fn should not run during cooldown")
	}
}

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
