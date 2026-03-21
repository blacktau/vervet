package oidc

import (
	"context"
	"fmt"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"

	"vervet/internal/connectionStrings"
	"vervet/internal/models"
)

type CachedToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type TokenManager struct {
	ctx         context.Context
	store       connectionStrings.Store
	mu          sync.RWMutex
	cache       map[string]*CachedToken
	openBrowser func(url string)
}

func NewTokenManager(store connectionStrings.Store) *TokenManager {
	return &TokenManager{
		store: store,
		cache: make(map[string]*CachedToken),
	}
}

func (tm *TokenManager) Init(ctx context.Context) {
	tm.ctx = ctx
}

func (tm *TokenManager) SetOpenBrowser(fn func(url string)) {
	tm.openBrowser = fn
}

func (tm *TokenManager) cacheToken(serverID string, token *CachedToken) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.cache[serverID] = token
}

func (tm *TokenManager) getCachedToken(serverID string) (*CachedToken, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tok, ok := tm.cache[serverID]
	if !ok {
		return nil, nil
	}
	if time.Now().After(tok.ExpiresAt) {
		return nil, nil
	}
	return tok, nil
}

func (tm *TokenManager) CleanupServer(serverID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.cache, serverID)
}

func (tm *TokenManager) Authenticate(ctx context.Context, serverID string) error {
	cfg, err := tm.store.GetConnectionConfig(serverID)
	if err != nil {
		return fmt.Errorf("failed to read connection config: %w", err)
	}

	if cfg.AuthMethod != models.AuthOIDC || cfg.OIDCConfig == nil {
		return fmt.Errorf("server %s is not configured for OIDC", serverID)
	}

	oidcCfg := cfg.OIDCConfig
	scopes := oidcCfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	if oidcCfg.WorkloadIdentity {
		return tm.workloadLogin(ctx, serverID, oidcCfg)
	}

	result, err := tm.browserLogin(ctx, oidcCfg.ProviderURL, oidcCfg.ClientID, scopes)
	if err != nil {
		return err
	}

	tm.cacheToken(serverID, &CachedToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
	})

	if result.RefreshToken != "" {
		if err := tm.store.UpdateRefreshToken(serverID, result.RefreshToken); err != nil {
			// Log but don't fail — token is cached in memory
			return nil
		}
	}

	return nil
}

func (tm *TokenManager) GetAccessToken(ctx context.Context, serverID string) (string, time.Time, error) {
	tok, err := tm.getCachedToken(serverID)
	if err != nil {
		return "", time.Time{}, err
	}
	if tok != nil {
		return tok.AccessToken, tok.ExpiresAt, nil
	}

	// Try refreshing
	cfg, err := tm.store.GetConnectionConfig(serverID)
	if err != nil {
		return "", time.Time{}, err
	}

	if cfg.RefreshToken != "" && cfg.OIDCConfig != nil && !cfg.OIDCConfig.WorkloadIdentity {
		tok, err := tm.refreshToken(ctx, cfg)
		if err == nil {
			tm.cacheToken(serverID, tok)
			if tok.RefreshToken != "" {
				_ = tm.store.UpdateRefreshToken(serverID, tok.RefreshToken)
			}
			return tok.AccessToken, tok.ExpiresAt, nil
		}
		// Refresh failed — notify frontend
		if tm.ctx != nil {
			wailsRuntime.EventsEmit(tm.ctx, "oidc-reauth-required", serverID)
		}
	}

	return "", time.Time{}, fmt.Errorf("no valid token for server %s — re-authentication required", serverID)
}

func (tm *TokenManager) refreshToken(ctx context.Context, cfg models.ConnectionConfig) (*CachedToken, error) {
	provider, err := gooidc.NewProvider(ctx, cfg.OIDCConfig.ProviderURL)
	if err != nil {
		return nil, err
	}

	oauth2Cfg := &oauth2.Config{
		ClientID: cfg.OIDCConfig.ClientID,
		Endpoint: provider.Endpoint(),
	}

	token := &oauth2.Token{RefreshToken: cfg.RefreshToken}
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

func (tm *TokenManager) workloadLogin(ctx context.Context, serverID string, cfg *models.OIDCConfig) error {
	// TODO: Implement workload identity (Azure IMDS, GCP metadata)
	return fmt.Errorf("workload identity not yet implemented")
}
