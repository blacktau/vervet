package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

func generateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type BrowserFlowResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// activeCallbackServer tracks an in-flight OIDC callback HTTP server so it
// can be shut down from outside the browserLogin goroutine. cancel cancels
// the derived context that browserLogin waits on, so closing the server
// also unblocks the pending select when the user never completes (or
// cancels) the provider flow.
type activeCallbackServer struct {
	server   *http.Server
	listener net.Listener
	cancel   context.CancelFunc
}

// closeBrowserServer shuts down any in-flight OIDC callback server, releasing
// the listener port and cancelling the context so the pending browserLogin
// select returns promptly. Callers block otherwise until the 5-minute
// timeout fires.
func (tm *TokenManager) closeBrowserServer() {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	if tm.activeServer != nil {
		if tm.activeServer.cancel != nil {
			tm.activeServer.cancel()
		}
		tm.activeServer.server.Close()
		tm.activeServer = nil
	}
}

func (tm *TokenManager) browserLogin(ctx context.Context, providerURL, clientID string, scopes []string) (*BrowserFlowResult, error) {
	// Close any leftover listener from a previous failed attempt.
	tm.closeBrowserServer()

	loginCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	provider, err := gooidc.NewProvider(loginCtx, providerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed for %s: %w", providerURL, err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:27097")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback listener: %w", err)
	}
	redirectURL := "http://localhost:27097/redirect"

	oauth2Cfg := &oauth2.Config{
		ClientID:    clientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	verifier, challenge, err := generatePKCE()
	if err != nil {
		listener.Close()
		return nil, err
	}

	state, err := generateState()
	if err != nil {
		listener.Close()
		return nil, err
	}

	resultCh := make(chan *BrowserFlowResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.Query().Get("error_description"))
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}

		token, err := oauth2Cfg.Exchange(loginCtx, code,
			oauth2.SetAuthURLParam("code_verifier", verifier),
		)
		if err != nil {
			errCh <- fmt.Errorf("token exchange failed: %w", err)
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			return
		}

		accessToken := token.AccessToken

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Authentication successful</h2><p>You can close this tab.</p></body></html>")

		resultCh <- &BrowserFlowResult{
			AccessToken:  accessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    token.Expiry,
		}
	})

	server := &http.Server{Handler: mux}

	// Track the active server so it can be closed externally.
	tm.browserMu.Lock()
	tm.activeServer = &activeCallbackServer{server: server, listener: listener, cancel: cancel}
	tm.browserMu.Unlock()

	go server.Serve(listener)

	authURL := oauth2Cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	if tm.openBrowser != nil {
		tm.openBrowser(authURL)
	}

	timeout := time.After(5 * time.Minute)
	defer tm.closeBrowserServer()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-timeout:
		return nil, fmt.Errorf("authentication timed out after 5 minutes")
	case <-loginCtx.Done():
		return nil, loginCtx.Err()
	}
}
