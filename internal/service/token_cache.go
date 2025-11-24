package service

import (
	"sync"
	"time"

	"github.com/bdsw/bdsw-im-ws/api/muc"
)

type tokenCacheEntry struct {
	response   *muc.TokenValidateResponse
	expireTime time.Time
}

type TokenCache struct {
	entries         map[string]*tokenCacheEntry
	mu              sync.RWMutex
	defaultTTL      time.Duration
	cleanupInterval time.Duration
}

func NewTokenCache(defaultTTL, cleanupInterval time.Duration) *TokenCache {
	cache := &TokenCache{
		entries:         make(map[string]*tokenCacheEntry),
		defaultTTL:      defaultTTL,
		cleanupInterval: cleanupInterval,
	}

	go cache.startCleanup()

	return cache
}

func (c *TokenCache) Get(userID, token string) *muc.TokenValidateResponse {
	key := c.generateKey(userID, token)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.entries[key]; exists {
		if time.Now().Before(entry.expireTime) {
			return entry.response
		}
	}
	return nil
}

func (c *TokenCache) Set(userID, token string, response *muc.TokenValidateResponse) {
	key := c.generateKey(userID, token)

	c.mu.Lock()
	defer c.mu.Unlock()

	ttl := c.defaultTTL
	if response.ExpireTime > 0 {
		expireDuration := time.Unix(response.ExpireTime, 0).Sub(time.Now())
		if expireDuration > 0 && expireDuration < c.defaultTTL {
			ttl = expireDuration
		}
	}

	c.entries[key] = &tokenCacheEntry{
		response:   response,
		expireTime: time.Now().Add(ttl),
	}
}

func (c *TokenCache) Invalidate(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) > len(userID) && key[:len(userID)] == userID && key[len(userID)] == ':' {
			delete(c.entries, key)
		}
	}
}

func (c *TokenCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *TokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expireTime) {
			delete(c.entries, key)
		}
	}
}

func (c *TokenCache) generateKey(userID, token string) string {
	return userID + ":" + token
}
