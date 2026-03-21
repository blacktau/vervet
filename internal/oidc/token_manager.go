package oidc

import (
	"sync"
	"time"

	"vervet/internal/connectionStrings"
)

type CachedToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type TokenManager struct {
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
