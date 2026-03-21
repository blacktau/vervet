package oidc

import (
	"testing"
	"time"
)

func TestGetAccessToken_ReturnsCachedToken(t *testing.T) {
	tm := NewTokenManager(nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-123",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})

	tok, err := tm.getCachedToken("server-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "tok-123" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "tok-123")
	}
}

func TestGetAccessToken_ReturnsNilForExpiredToken(t *testing.T) {
	tm := NewTokenManager(nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-expired",
		ExpiresAt:   time.Now().Add(-1 * time.Minute),
	})

	tok, _ := tm.getCachedToken("server-1")
	if tok != nil {
		t.Error("expected nil for expired token")
	}
}

func TestCleanupServer_RemovesCachedToken(t *testing.T) {
	tm := NewTokenManager(nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-123",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})

	tm.CleanupServer("server-1")

	tok, _ := tm.getCachedToken("server-1")
	if tok != nil {
		t.Error("expected nil after cleanup")
	}
}
