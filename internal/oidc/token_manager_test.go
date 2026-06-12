package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// stubStore is a minimal connectionStrings.Store for OIDC callback tests.
type stubStore struct {
	cfg models.ConnectionConfig
}

func (s *stubStore) StoreRegisteredServerURI(string, string) error { return nil }
func (s *stubStore) GetRegisteredServerURI(string) (string, error) { return "", nil }
func (s *stubStore) DeleteRegisteredServerURI(string) error        { return nil }
func (s *stubStore) StoreConnectionConfig(string, models.ConnectionConfig) error {
	return nil
}
func (s *stubStore) GetConnectionConfig(string) (models.ConnectionConfig, error) {
	return s.cfg, nil
}
func (s *stubStore) UpdateRefreshToken(string, string) error { return nil }

func TestGetCachedToken_ReturnsCachedToken(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-123",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})

	tok := tm.getCachedToken("server-1")
	if tok == nil {
		t.Fatal("expected non-nil token")
	}
	if tok.AccessToken != "tok-123" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "tok-123")
	}
}

func TestGetCachedToken_ReturnsNilForExpiredToken(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-expired",
		ExpiresAt:   time.Now().Add(-1 * time.Minute),
	})

	tok := tm.getCachedToken("server-1")
	if tok != nil {
		t.Error("expected nil for expired token")
	}
}

func TestCleanupServer_RemovesCachedToken(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-123",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})

	tm.CleanupServer("server-1")

	tok := tm.getCachedToken("server-1")
	if tok != nil {
		t.Error("expected nil after cleanup")
	}
}

func TestShutdown_ClosesActiveCallbackServer(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)

	// Simulate an active callback server by binding the OIDC listener port.
	listener, err := net.Listen("tcp", "127.0.0.1:27097")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	server := &http.Server{}
	go server.Serve(listener)
	time.Sleep(50 * time.Millisecond) // let Serve start

	tm.browserMu.Lock()
	tm.activeServer = &activeCallbackServer{server: server, listener: listener}
	tm.browserMu.Unlock()

	// Shutdown should release the port.
	tm.Shutdown()
	time.Sleep(50 * time.Millisecond) // let OS release the socket

	// Verify the port is free by binding to it again.
	listener2, err := net.Listen("tcp", "127.0.0.1:27097")
	if err != nil {
		t.Fatalf("port still bound after Shutdown: %v", err)
	}
	listener2.Close()
}

func TestCloseBrowserServer_AllowsRetry(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)

	// Simulate a leftover listener.
	listener, err := net.Listen("tcp", "127.0.0.1:27097")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	server := &http.Server{}
	go server.Serve(listener)
	time.Sleep(50 * time.Millisecond) // let Serve start

	tm.browserMu.Lock()
	tm.activeServer = &activeCallbackServer{server: server, listener: listener}
	tm.browserMu.Unlock()

	// closeBrowserServer should release the port so a new listener can bind.
	tm.closeBrowserServer()
	time.Sleep(50 * time.Millisecond) // let OS release the socket

	listener2, err := net.Listen("tcp", "127.0.0.1:27097")
	if err != nil {
		t.Fatalf("port still bound after closeBrowserServer: %v", err)
	}
	listener2.Close()
}

// After Shutdown, the OIDC human callback must not start an interactive
// browser login. Otherwise closing Vervet while an OIDC connection is dead
// (no valid cached token, refresh unavailable) triggers the driver's
// disconnect-time re-auth, popping a login browser during app exit.
func TestHumanCallback_DoesNotLaunchBrowserAfterShutdown(t *testing.T) {
	store := &stubStore{cfg: models.ConnectionConfig{AuthMethod: models.AuthOIDC}}
	tm := NewTokenManager(slog.Default(), store)
	tm.Init(context.Background())

	browserOpened := false
	tm.SetOpenBrowser(func(string) { browserOpened = true })

	tm.Shutdown()

	cb := tm.HumanCallback("server-1", &models.OIDCConfig{})
	_, err := cb(context.Background(), &options.OIDCArgs{
		IDPInfo: &options.IDPInfo{
			Issuer:   "https://idp.example.com",
			ClientID: "client-1",
		},
	})

	if !errors.Is(err, ErrShuttingDown) {
		t.Fatalf("expected ErrShuttingDown, got %v", err)
	}
	if browserOpened {
		t.Fatal("browser login was launched during shutdown")
	}
}
