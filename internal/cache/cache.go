package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

// CacheEntry holds a cached completion response and expiration time.
type CacheEntry struct {
	Response  provider.ChatCompletionResponse `json:"response"`
	ExpiresAt time.Time                       `json:"expires_at"`
	CreatedAt time.Time                       `json:"created_at"`
}

// CacheStats tracks cache performance metrics.
type CacheStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	Total      int64   `json:"total"`
	HitRatio   float64 `json:"hit_ratio"`
	TotalItems int     `json:"total_items"`
}

// PromptCache manages fast in-memory normalized prompt lookups.
type PromptCache struct {
	mu      sync.RWMutex
	items   map[string]CacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
}

// NewPromptCache creates a prompt cache with given capacity and TTL.
func NewPromptCache(maxSize int, ttl time.Duration) *PromptCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	c := &PromptCache{
		items:   make(map[string]CacheEntry, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}

	go c.cleanupLoop()
	return c
}

// GenerateKey produces a deterministic SHA-256 hash from request messages.
func (c *PromptCache) GenerateKey(req *provider.ChatCompletionRequest) string {
	hasher := sha256.New()

	for _, m := range req.Messages {
		hasher.Write([]byte(m.Role))
		hasher.Write([]byte(":"))
		hasher.Write([]byte(m.ContentString()))
		hasher.Write([]byte("|"))
	}

	if req.Temperature != nil {
		hasher.Write([]byte(fmt.Sprintf("t:%.2f|", *req.Temperature)))
	}
	if req.ResponseFormat != nil {
		hasher.Write([]byte(fmt.Sprintf("rf:%s|", req.ResponseFormat.Type)))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// Get checks if an active cache entry exists for the request.
func (c *PromptCache) Get(req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, bool) {
	key := c.GenerateKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.items[key]
	if !found {
		c.misses++
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(c.items, key)
		c.misses++
		return nil, false
	}

	c.hits++

	// Clone response with fresh timestamp and unique ID
	respCopy := entry.Response
	respCopy.ID = fmt.Sprintf("chatcmpl-cache-%d", time.Now().UnixNano())
	respCopy.Created = time.Now().Unix()

	return &respCopy, true
}

// Set stores a completion response in the cache.
func (c *PromptCache) Set(req *provider.ChatCompletionRequest, resp *provider.ChatCompletionResponse) {
	if resp == nil || len(resp.Choices) == 0 {
		return
	}

	key := c.GenerateKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction if max capacity reached
	if len(c.items) >= c.maxSize {
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}

	c.items[key] = CacheEntry{
		Response:  *resp,
		ExpiresAt: time.Now().Add(c.ttl),
		CreatedAt: time.Now(),
	}
}

// Stats returns current cache effectiveness metrics.
func (c *PromptCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = (float64(c.hits) / float64(total)) * 100.0
	}

	return CacheStats{
		Hits:       c.hits,
		Misses:     c.misses,
		Total:      total,
		HitRatio:   hitRatio,
		TotalItems: len(c.items),
	}
}

// Clear flushes all entries from memory.
func (c *PromptCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CacheEntry, c.maxSize)
	c.hits = 0
	c.misses = 0
}

func (c *PromptCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.ExpiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// MarshalJSON helper for serializing cache state
func (c *PromptCache) ExportJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.items)
}
