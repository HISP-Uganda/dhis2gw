package main

import (
	"sync"
)

type TokenStore struct {
	mu           sync.RWMutex
	AccessToken  string
	RefreshToken string
}

func (t *TokenStore) SetTokens(access, refresh string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.AccessToken = access
	t.RefreshToken = refresh
}

func (t *TokenStore) GetAccessToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.AccessToken
}

func (t *TokenStore) GetRefreshToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.RefreshToken
}
