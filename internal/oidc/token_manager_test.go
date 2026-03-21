package oidc

import (
	"log/slog"
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
