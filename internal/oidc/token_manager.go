package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/oauth2"

	"vervet/internal/connectionStrings"
	"vervet/internal/models"
)

// ErrLoginCanceled is returned when the user cancels the OIDC browser login
// flow (e.g. closes the manual URL dialog). Callers should treat this as a
// benign user action, not an error to surface as a failure notification.
var ErrLoginCanceled = errors.New("OIDC login canceled by user")

// ErrShuttingDown is returned by the OIDC human callback when the application
// is shutting down. The MongoDB driver invokes the callback during
// disconnect-time re-authentication; without this guard a dead OIDC
// connection would pop a browser login while Vervet is closing.
var ErrShuttingDown = errors.New("OIDC login skipped: application shutting down")

type CachedToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type TokenManager struct {
	ctx         context.Context
	log         *slog.Logger
	store       connectionStrings.Store
	mu          sync.RWMutex
	cache       map[string]*CachedToken
	openBrowser func(url string)
	emitAuthURL func(serverID, url string)

	// browserMu protects activeServer — the in-flight OIDC callback HTTP server.
	browserMu    sync.Mutex
	activeServer *activeCallbackServer
	canceled     bool
	shuttingDown bool

	promptMu        sync.Mutex
	forcePromptOnce map[string]string
}

func NewTokenManager(log *slog.Logger, store connectionStrings.Store) *TokenManager {
	return &TokenManager{
		log:             log,
		store:           store,
		cache:           make(map[string]*CachedToken),
		forcePromptOnce: make(map[string]string),
	}
}

// ResetSession clears cached + persisted OIDC state for a server and arms a
// one-shot account-picker prompt for the next browser login. Caller should
// disconnect the active MongoDB client first. No-op if the server isn't
// configured with OIDC.
func (tm *TokenManager) ResetSession(serverID string) error {
	cfg, err := tm.store.GetConnectionConfig(serverID)
	if err != nil {
		return fmt.Errorf("read connection config: %w", err)
	}
	if cfg.AuthMethod != models.AuthOIDC {
		return nil
	}
	tm.CleanupServer(serverID)
	if cfg.RefreshToken != "" {
		if err := tm.store.UpdateRefreshToken(serverID, ""); err != nil {
			return fmt.Errorf("clear refresh token: %w", err)
		}
	}
	tm.promptMu.Lock()
	tm.forcePromptOnce[serverID] = "select_account"
	tm.promptMu.Unlock()
	return nil
}

func (tm *TokenManager) consumeForcePrompt(serverID string) string {
	tm.promptMu.Lock()
	defer tm.promptMu.Unlock()
	p, ok := tm.forcePromptOnce[serverID]
	if ok {
		delete(tm.forcePromptOnce, serverID)
	}
	return p
}

func (tm *TokenManager) Init(ctx context.Context) {
	tm.ctx = ctx
}

func (tm *TokenManager) SetOpenBrowser(fn func(url string)) {
	tm.openBrowser = fn
}

func (tm *TokenManager) SetEmitAuthURL(fn func(serverID, url string)) {
	tm.emitAuthURL = fn
}

// CancelLogin shuts down any in-flight OIDC browser login for the given
// server. Currently a single in-flight server is tracked globally, so the
// serverID is advisory only.
func (tm *TokenManager) CancelLogin(serverID string) {
	tm.browserMu.Lock()
	tm.canceled = true
	tm.browserMu.Unlock()
	tm.closeBrowserServer()
}

func (tm *TokenManager) consumeCanceled() bool {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	c := tm.canceled
	tm.canceled = false
	return c
}

func (tm *TokenManager) cacheToken(serverID string, token *CachedToken) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.cache[serverID] = token
}

func (tm *TokenManager) getCachedToken(serverID string) *CachedToken {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tok, ok := tm.cache[serverID]
	if !ok {
		return nil
	}
	if time.Now().After(tok.ExpiresAt) {
		return nil
	}
	return tok
}

func (tm *TokenManager) CleanupServer(serverID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.cache, serverID)
}

// Shutdown cancels any in-flight browser login and releases its listener.
// Call this on application exit.
func (tm *TokenManager) Shutdown() {
	tm.browserMu.Lock()
	tm.shuttingDown = true
	tm.browserMu.Unlock()
	tm.closeBrowserServer()
}

func (tm *TokenManager) isShuttingDown() bool {
	tm.browserMu.Lock()
	defer tm.browserMu.Unlock()
	return tm.shuttingDown
}

// HumanCallback returns the OIDCHumanCallback for the given server.
// The callback is invoked by the MongoDB driver during the OIDC handshake.
// It uses IDPInfo provided by the server (issuer, clientID, scopes),
// falling back to user-configured values if the server doesn't provide them.
func (tm *TokenManager) HumanCallback(serverID string, cfg *models.OIDCConfig) options.OIDCCallback {
	return func(ctx context.Context, args *options.OIDCArgs) (*options.OIDCCredential, error) {
		// 1. Return cached token if valid
		if cached := tm.getCachedToken(serverID); cached != nil {
			tm.log.Debug("Returning cached OIDC token", slog.String("serverID", serverID))
			return &options.OIDCCredential{
				AccessToken: cached.AccessToken,
				ExpiresAt:   &cached.ExpiresAt,
			}, nil
		}

		// 2. Try refresh token
		storedCfg, err := tm.store.GetConnectionConfig(serverID)
		if err == nil && storedCfg.RefreshToken != "" {
			providerURL, clientID := tm.resolveIDPInfo(args.IDPInfo, cfg)
			if providerURL != "" && clientID != "" {
				tok, refreshErr := tm.refreshToken(ctx, providerURL, clientID, storedCfg.RefreshToken)
				if refreshErr == nil {
					tm.log.Debug("Refreshed OIDC token", slog.String("serverID", serverID))
					tm.cacheToken(serverID, tok)
					if tok.RefreshToken != "" {
						_ = tm.store.UpdateRefreshToken(serverID, tok.RefreshToken)
					}
					return &options.OIDCCredential{
						AccessToken: tok.AccessToken,
						ExpiresAt:   &tok.ExpiresAt,
					}, nil
				}
				tm.log.Warn("Token refresh failed, falling back to browser login",
					slog.String("serverID", serverID), slog.Any("error", refreshErr))
			}
		}

		// Don't pop an interactive browser login while the app is closing.
		// The driver invokes this callback during disconnect-time re-auth; a
		// dead OIDC connection would otherwise launch a browser on exit.
		if tm.isShuttingDown() {
			return nil, ErrShuttingDown
		}

		// 3. Browser login — resolve provider info from server or user config
		providerURL, clientID := tm.resolveIDPInfo(args.IDPInfo, cfg)
		if providerURL == "" {
			return nil, fmt.Errorf("OIDC provider URL not available — the server did not advertise one and none was configured")
		}
		if clientID == "" {
			return nil, fmt.Errorf("OIDC client ID not available — the server did not advertise one and none was configured")
		}

		scopes := tm.resolveScopes(args.IDPInfo, cfg)

		tm.log.Debug("Starting OIDC browser login",
			slog.String("serverID", serverID),
			slog.String("providerURL", providerURL),
			slog.String("clientID", clientID),
			slog.Any("scopes", scopes))

		var prompt string
		var manualURL bool
		if cfg != nil {
			prompt = cfg.Prompt
			manualURL = cfg.ManualURLMode
		}
		if forced := tm.consumeForcePrompt(serverID); forced != "" {
			prompt = forced
		}

		result, err := tm.browserLogin(ctx, serverID, providerURL, clientID, prompt, scopes, manualURL)
		if err != nil {
			if errors.Is(err, ErrLoginCanceled) {
				return nil, err
			}
			return nil, fmt.Errorf("OIDC browser login failed: %w", err)
		}

		tm.cacheToken(serverID, &CachedToken{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		})

		if result.RefreshToken != "" {
			if err := tm.store.UpdateRefreshToken(serverID, result.RefreshToken); err != nil {
				tm.log.Warn("Failed to persist refresh token", slog.String("serverID", serverID), slog.Any("error", err))
			}
		}

		return &options.OIDCCredential{
			AccessToken: result.AccessToken,
			ExpiresAt:   &result.ExpiresAt,
		}, nil
	}
}

// MachineCallback returns the OIDCMachineCallback for workload identity.
func (tm *TokenManager) MachineCallback(serverID string) options.OIDCCallback {
	return func(ctx context.Context, args *options.OIDCArgs) (*options.OIDCCredential, error) {
		// TODO: Implement workload identity (Azure IMDS, GCP metadata)
		return nil, fmt.Errorf("workload identity not yet implemented")
	}
}

// resolveIDPInfo prefers server-provided IDPInfo, falls back to user config.
func (tm *TokenManager) resolveIDPInfo(idpInfo *options.IDPInfo, cfg *models.OIDCConfig) (providerURL, clientID string) {
	if idpInfo != nil {
		providerURL = idpInfo.Issuer
		clientID = idpInfo.ClientID
	}
	// User config overrides if set
	if cfg != nil {
		if cfg.ProviderURL != "" {
			providerURL = cfg.ProviderURL
		}
		if cfg.ClientID != "" {
			clientID = cfg.ClientID
		}
	}
	return providerURL, clientID
}

func (tm *TokenManager) resolveScopes(idpInfo *options.IDPInfo, cfg *models.OIDCConfig) []string {
	// User-configured scopes override everything
	if cfg != nil && len(cfg.Scopes) > 0 {
		return cfg.Scopes
	}
	// Use server-requested scopes
	if idpInfo != nil && len(idpInfo.RequestScopes) > 0 {
		return idpInfo.RequestScopes
	}
	return []string{"openid"}
}

func (tm *TokenManager) refreshToken(ctx context.Context, providerURL, clientID, refreshToken string) (*CachedToken, error) {
	provider, err := gooidc.NewProvider(ctx, providerURL)
	if err != nil {
		return nil, err
	}

	oauth2Cfg := &oauth2.Config{
		ClientID: clientID,
		Endpoint: provider.Endpoint(),
	}

	token := &oauth2.Token{RefreshToken: refreshToken}
	newToken, err := oauth2Cfg.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	return &CachedToken{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
	}, nil
}
