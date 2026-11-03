package connectionStrings

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vervet/internal/models"

	"github.com/zalando/go-keyring"
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

func TestUpdateRefreshToken(t *testing.T) {
	oldDetails := models.ConnectionConfig{URI: "mongodb://old", AuthMethod: models.AuthOIDC}
	newDetails := models.ConnectionConfig{URI: "mongodb://new", AuthMethod: models.AuthOIDC}

	t.Run("stores a token issued for the current details", func(t *testing.T) {
		keyring.MockInit()
		s := newTestStore()
		if err := s.StoreConnectionConfig("1", oldDetails); err != nil {
			t.Fatal(err)
		}

		if err := s.UpdateRefreshToken("1", "token", oldDetails.TokenBinding()); err != nil {
			t.Fatal(err)
		}

		cfg, _ := s.GetConnectionConfig("1")
		if cfg.RefreshToken != "token" {
			t.Errorf("RefreshToken = %q, want %q", cfg.RefreshToken, "token")
		}
	})

	t.Run("discards a late token issued for previous details", func(t *testing.T) {
		keyring.MockInit()
		s := newTestStore()
		if err := s.StoreConnectionConfig("1", newDetails); err != nil {
			t.Fatal(err)
		}

		if err := s.UpdateRefreshToken("1", "old-token", oldDetails.TokenBinding()); err != nil {
			t.Fatal(err)
		}

		cfg, _ := s.GetConnectionConfig("1")
		if cfg.RefreshToken != "" {
			t.Errorf("RefreshToken = %q, want it discarded", cfg.RefreshToken)
		}
		if cfg.URI != newDetails.URI {
			t.Errorf("URI = %q, want the new details kept", cfg.URI)
		}
	})
}

func TestGetConnectionConfigMissingEntryIsErrConfigNotFound(t *testing.T) {
	keyring.MockInit()

	_, err := newTestStore().GetConnectionConfig("no-such-server")

	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestGetConnectionConfigKeyringFailureIsNotErrConfigNotFound(t *testing.T) {
	keyring.MockInitWithError(errors.New("keyring locked"))

	_, err := newTestStore().GetConnectionConfig("server")

	if err == nil || errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected a keyring failure, got %v", err)
	}
}
