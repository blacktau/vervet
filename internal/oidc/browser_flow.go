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

// activeCallbackServer tracks an in-flight OIDC login so it can be stopped from
// outside the browserLogin goroutine. cancel cancels the derived context that
// browserLogin waits on, so closing it also unblocks the pending select when
// the user never completes (or cancels) the provider flow. server is nil until
// discovery has finished and the callback listener is up.
type activeCallbackServer struct {
	serverID string
	server   *http.Server
	listener net.Listener
	cancel   context.CancelFunc
	canceled bool
}

// closeBrowserServer stops any in-flight OIDC login, releasing the listener port
// and cancelling the context so the pending browserLogin select returns
// promptly. Callers block otherwise until the 5-minute timeout fires.
func (tm *TokenManager) closeBrowserServer() {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	tm.closeActiveLoginLocked()
}

func (tm *TokenManager) closeActiveLoginLocked() {
	if tm.activeServer == nil {
		return
	}
	if tm.activeServer.cancel != nil {
		tm.activeServer.cancel()
	}
	if tm.activeServer.server != nil {
		tm.activeServer.server.Close()
	}
	tm.activeServer = nil
}

// beginLogin registers a login before any network call, so a cancel also stops
// one that is still waiting on OIDC discovery. It replaces any earlier login.
func (tm *TokenManager) beginLogin(serverID string, cancel context.CancelFunc) *activeCallbackServer {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	tm.closeActiveLoginLocked()
	tm.activeServer = &activeCallbackServer{serverID: serverID, cancel: cancel}
	return tm.activeServer
}

// attachServer records the callback server. It returns false if the login has
// been stopped in the meantime, in which case the caller must not serve.
func (tm *TokenManager) attachServer(login *activeCallbackServer, server *http.Server, listener net.Listener) bool {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	if tm.activeServer != login {
		return false
	}
	login.server = server
	login.listener = listener
	return true
}

// endLogin closes this login only: a newer login may have replaced it.
func (tm *TokenManager) endLogin(login *activeCallbackServer) {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	if tm.activeServer == login {
		tm.closeActiveLoginLocked()
	}
}

func (tm *TokenManager) wasCanceled(login *activeCallbackServer) bool {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	return login.canceled
}

func (tm *TokenManager) browserLogin(ctx context.Context, serverID, providerURL, clientID, prompt string, scopes []string, manualURL bool) (*BrowserFlowResult, error) {
	loginCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	login := tm.beginLogin(serverID, cancel)
	defer tm.endLogin(login)

	provider, err := gooidc.NewProvider(loginCtx, providerURL)
	if err != nil {
		if tm.wasCanceled(login) {
			return nil, ErrLoginCanceled
		}
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

	if !tm.attachServer(login, server, listener) {
		listener.Close()
		if tm.wasCanceled(login) {
			return nil, ErrLoginCanceled
		}
		return nil, context.Canceled
	}

	go server.Serve(listener)

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if prompt != "" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", prompt))
	}
	authURL := oauth2Cfg.AuthCodeURL(state, opts...)

	if manualURL {
		if tm.emitAuthURL != nil {
			tm.emitAuthURL(serverID, authURL)
		}
	} else if tm.openBrowser != nil {
		tm.openBrowser(authURL)
	}

	timeout := time.After(5 * time.Minute)

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-timeout:
		return nil, fmt.Errorf("authentication timed out after 5 minutes")
	case <-loginCtx.Done():
		if tm.wasCanceled(login) {
			return nil, ErrLoginCanceled
		}
		return nil, loginCtx.Err()
	}
}
