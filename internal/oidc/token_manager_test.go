package oidc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// stubStore is a minimal connectionStrings.Store for OIDC callback tests.
type stubStore struct {
	cfg models.ConnectionConfig
}

func (s *stubStore) StoreRegisteredServerURI(string, string) error { return nil }
func (s *stubStore) DeleteRegisteredServerURI(string) error        { return nil }
func (s *stubStore) StoreConnectionConfig(string, models.ConnectionConfig) error {
	return nil
}
func (s *stubStore) GetConnectionConfig(string) (models.ConnectionConfig, error) {
	return s.cfg, nil
}
func (s *stubStore) UpdateRefreshToken(string, string, string) error { return nil }

func TestGetCachedToken_ReturnsCachedToken(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "tok-123",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})

	tok := tm.getCachedToken("server-1", "")
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

	tok := tm.getCachedToken("server-1", "")
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

	tok := tm.getCachedToken("server-1", "")
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

	cb := tm.HumanCallback("server-1", models.ConnectionConfig{OIDCConfig: &models.OIDCConfig{}})
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

func newTestManager() *TokenManager {
	// Discard logger keeps test output pristine; store is unused on the paths
	// under test (cached-token, resolveIDPInfo, resolveScopes, MachineCallback).
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewTokenManager(log, nil)
}

func TestHumanCallback_ReturnsCachedToken(t *testing.T) {
	tm := newTestManager()
	exp := time.Now().Add(time.Hour)
	cfg := models.ConnectionConfig{URI: "mongodb://host", OIDCConfig: &models.OIDCConfig{}}
	tm.cacheToken("srv1", &CachedToken{AccessToken: "cached-abc", ExpiresAt: exp, Binding: cfg.TokenBinding()})

	cb := tm.HumanCallback("srv1", cfg)
	cred, err := cb(context.Background(), &options.OIDCArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "cached-abc" {
		t.Errorf("AccessToken = %q, want %q", cred.AccessToken, "cached-abc")
	}
	if cred.ExpiresAt == nil || !cred.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt = %v, want %v", cred.ExpiresAt, exp)
	}
}

func TestResolveIDPInfo_ConfigOverridesServer(t *testing.T) {
	tm := newTestManager()
	idp := &options.IDPInfo{Issuer: "https://server-issuer", ClientID: "server-cid"}

	// No config override → server values win.
	url, cid := tm.resolveIDPInfo(idp, &models.OIDCConfig{})
	if url != "https://server-issuer" || cid != "server-cid" {
		t.Errorf("got (%q,%q), want server values", url, cid)
	}

	// Config set → config wins.
	url, cid = tm.resolveIDPInfo(idp, &models.OIDCConfig{ProviderURL: "https://cfg-issuer", ClientID: "cfg-cid"})
	if url != "https://cfg-issuer" || cid != "cfg-cid" {
		t.Errorf("got (%q,%q), want config values", url, cid)
	}
}

func TestResolveScopes_Precedence(t *testing.T) {
	tm := newTestManager()

	// Config scopes win.
	got := tm.resolveScopes(&options.IDPInfo{RequestScopes: []string{"srv"}}, &models.OIDCConfig{Scopes: []string{"cfg"}})
	if len(got) != 1 || got[0] != "cfg" {
		t.Errorf("got %v, want [cfg]", got)
	}

	// Else server-requested scopes.
	got = tm.resolveScopes(&options.IDPInfo{RequestScopes: []string{"srv"}}, &models.OIDCConfig{})
	if len(got) != 1 || got[0] != "srv" {
		t.Errorf("got %v, want [srv]", got)
	}

	// Else default openid.
	got = tm.resolveScopes(nil, nil)
	if len(got) != 1 || got[0] != "openid" {
		t.Errorf("got %v, want [openid]", got)
	}
}

func TestMachineCallback_NotImplemented(t *testing.T) {
	tm := newTestManager()
	cb := tm.MachineCallback("srv1")
	if _, err := cb(context.Background(), &options.OIDCArgs{}); err == nil {
		t.Fatal("expected not-implemented error, got nil")
	}
}

// While a server is disconnecting, the OIDC human callback must not start an
// interactive browser login. The driver re-authenticates during Disconnect, so
// closing a tab for a dead OIDC connection would otherwise pop a login window.
func TestHumanCallback_DoesNotLaunchBrowserWhileDisconnecting(t *testing.T) {
	store := &stubStore{cfg: models.ConnectionConfig{AuthMethod: models.AuthOIDC}}
	tm := NewTokenManager(slog.Default(), store)
	tm.Init(context.Background())

	browserOpened := false
	tm.SetOpenBrowser(func(string) { browserOpened = true })

	release := tm.SuppressLogin("server-1")

	cb := tm.HumanCallback("server-1", models.ConnectionConfig{OIDCConfig: &models.OIDCConfig{}})
	args := &options.OIDCArgs{
		IDPInfo: &options.IDPInfo{
			Issuer:   "https://idp.example.com",
			ClientID: "client-1",
		},
	}

	if _, err := cb(context.Background(), args); !errors.Is(err, ErrDisconnecting) {
		t.Fatalf("expected ErrDisconnecting, got %v", err)
	}
	if browserOpened {
		t.Fatal("browser login was launched during disconnect")
	}

	// Other servers are unaffected, and releasing lifts the suppression.
	if tm.isDisconnecting("server-2") {
		t.Fatal("suppression leaked to another server")
	}
	release()
	if tm.isDisconnecting("server-1") {
		t.Fatal("suppression outlived its release")
	}
}

// Nested/concurrent disconnects must not unsuppress each other early.
func TestSuppressLogin_Nested(t *testing.T) {
	tm := NewTokenManager(slog.Default(), &stubStore{})

	releaseOuter := tm.SuppressLogin("server-1")
	releaseInner := tm.SuppressLogin("server-1")

	releaseInner()
	if !tm.isDisconnecting("server-1") {
		t.Fatal("inner release cleared suppression while outer still held it")
	}

	releaseOuter()
	if tm.isDisconnecting("server-1") {
		t.Fatal("suppression outlived the outer release")
	}
}

func TestForgetSession_RemovesCachedToken(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.cacheToken("server-1", &CachedToken{AccessToken: "old", ExpiresAt: time.Now().Add(10 * time.Minute)})
	tm.cacheToken("server-2", &CachedToken{AccessToken: "other", ExpiresAt: time.Now().Add(10 * time.Minute)})

	tm.ForgetSession("server-1")

	if tm.getCachedToken("server-1", "") != nil {
		t.Error("expected server-1's cached token to be gone")
	}
	if tm.getCachedToken("server-2", "") == nil {
		t.Error("expected server-2's cached token to survive")
	}
}

func TestForgetSession_CancelsOnlyThatServersLogin(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	loginCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	login := tm.beginLogin("server-1", cancel)

	tm.ForgetSession("server-2")

	if loginCtx.Err() != nil {
		t.Fatal("another server's edit must not cancel this login")
	}
	if tm.wasCanceled(login) {
		t.Fatal("another server's edit must not mark the login as cancelled")
	}

	tm.ForgetSession("server-1")

	if loginCtx.Err() == nil {
		t.Error("expected the login to be cancelled")
	}
	if !tm.wasCanceled(login) {
		t.Error("expected the login to report as cancelled, not failed")
	}
}

// A token issued for old details must not be sent once the details have changed,
// however it got there: a late login, a missed reset or a keyring timeout.
func TestHumanCallback_IgnoresTokensForOtherDetails(t *testing.T) {
	var refreshAttempts atomic.Int32
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshAttempts.Add(1)
		http.NotFound(w, r)
	}))
	defer idp.Close()

	oldDetails := models.ConnectionConfig{
		URI:          "mongodb://old-host",
		AuthMethod:   models.AuthOIDC,
		OIDCConfig:   &models.OIDCConfig{ProviderURL: idp.URL, ClientID: "vervet"},
		RefreshToken: "old-refresh",
	}
	newDetails := oldDetails
	newDetails.URI = "mongodb://new-host"
	newDetails.RefreshToken = ""

	tm := NewTokenManager(slog.New(slog.NewTextHandler(io.Discard, nil)), &stubStore{cfg: oldDetails})
	tm.Init(context.Background())
	tm.cacheToken("server-1", &CachedToken{
		AccessToken: "old-access",
		ExpiresAt:   time.Now().Add(time.Hour),
		Binding:     oldDetails.TokenBinding(),
	})
	// Stops the callback where a browser login would start.
	defer tm.SuppressLogin("server-1")()

	_, err := tm.HumanCallback("server-1", newDetails)(context.Background(), &options.OIDCArgs{})

	if !errors.Is(err, ErrDisconnecting) {
		t.Fatalf("expected the callback to reach the login step, got %v", err)
	}
	if n := refreshAttempts.Load(); n != 0 {
		t.Errorf("old refresh token was sent to the provider %d times", n)
	}
}

func startBlockedLogin(t *testing.T, tm *TokenManager, ctx context.Context) <-chan error {
	t.Helper()
	inDiscovery := make(chan struct{})
	release := make(chan struct{})
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(inDiscovery)
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		idp.Close()
	})

	result := make(chan error, 1)
	go func() {
		_, err := tm.browserLogin(ctx, "server-1", idp.URL, "vervet", "", nil, false)
		result <- err
	}()
	<-inDiscovery
	return result
}

func TestCancelLogin_StopsLoginStillInDiscovery(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	browserOpened := false
	tm.SetOpenBrowser(func(string) { browserOpened = true })
	result := startBlockedLogin(t, tm, context.Background())

	tm.CancelLogin("server-1")

	select {
	case err := <-result:
		if !errors.Is(err, ErrLoginCanceled) {
			t.Fatalf("expected ErrLoginCanceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("login kept waiting on discovery after being cancelled")
	}
	if browserOpened {
		t.Fatal("browser opened for a cancelled login")
	}
}

func TestCancelLogin_LeavesAnotherServersLoginRunning(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	result := startBlockedLogin(t, tm, ctx)

	tm.CancelLogin("server-2")

	select {
	case err := <-result:
		t.Fatalf("login stopped by another server's cancel: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	<-result
}

// Cancelling a connect that has no login running, as any password server's
// cancel button does, must not make a later real failure look like a cancel.
func TestCancelLogin_WithNoLoginDoesNotAffectLaterFailures(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	tm.CancelLogin("server-1")

	ctx, cancel := context.WithCancel(context.Background())
	result := startBlockedLogin(t, tm, ctx)
	cancel()

	if err := <-result; errors.Is(err, ErrLoginCanceled) {
		t.Fatal("a failed login was reported as a user cancel")
	}
}

func TestEndLogin_LeavesNewerLoginRunning(t *testing.T) {
	tm := NewTokenManager(slog.Default(), nil)
	first := tm.beginLogin("server-1", func() {})
	newerCtx, cancelNewer := context.WithCancel(context.Background())
	defer cancelNewer()
	tm.beginLogin("server-2", cancelNewer)

	tm.endLogin(first)

	if newerCtx.Err() != nil {
		t.Fatal("an older login's cleanup closed the newer one")
	}
}
