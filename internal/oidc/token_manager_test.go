package oidc

import (
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

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
