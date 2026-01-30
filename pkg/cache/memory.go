// Package cache provides a thread-safe, in-memory key-value store with
// TTL-based expiration and active memory management (eviction).
package cache

import (
	"log"
	"sort"
	"sync"
	"time"

	"octa/internal/config"
	"octa/pkg/logger"
	"octa/pkg/utils"
)

const (
	// 100 * 1024 * 1024
	DefaultMaxSize = 100 // 100 MB Limit
	DefaultTTL     = 30 * time.Minute

	// GCInterval: Expired items cleanup frequency.
	// 10 minutes is a good balance to avoid frequent locking overhead.
	GCInterval = 5 * time.Minute

	// MonitorInterval: Heartbeat logging.
	// 30 minutes is sufficient for production observability.
	// Reduce this only during active debugging.
	MonitorInterval = 30 * time.Minute
)

type Item struct {
	Data      []byte
	ExpiresAt time.Time
	Size      int64
}

type MemoryCache struct {
	sync.RWMutex
	items     map[string]Item
	totalSize int64
	maxSize   int64
	ttl       time.Duration
	enabled   bool
}

// New initializes the in-memory cache system.
// It configures size limits and starts background maintenance routines (GC & Monitor).
func New() *MemoryCache {

	limitMB := int64(config.AppConfig.Cache.MaxCapacity)
	if limitMB <= 0 {
		limitMB = DefaultMaxSize
	}
	maxSize := limitMB * 1024 * 1024

	ttlStr := config.AppConfig.Cache.TTL
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = DefaultTTL

		logger.LogWarn("Invalid cache TTL '%s', using default 30m", ttlStr)
	}

	isEnabled := config.AppConfig.Cache.Enabled
	c := &MemoryCache{
		// items:   make(map[string]Item),
		maxSize: maxSize,
		ttl:     ttl,
		enabled: isEnabled,
	}

	if c.enabled {
		c.items = make(map[string]Item)

		// Go Workers
		go c.startGC()      // Garbage Worker
		go c.startMonitor() // Statistics Worker

		
		logger.LogInfo("Memory Cache Initialized: %d MB Limit, TTL: %s", limitMB, ttl)
	} else {
		
		logger.LogWarn("Memory Cache is DISABLED via config (Running in pass-through mode).")
	}
	return c
}

// Set stores a value in the cache with the configured TTL.
// Large items (>512KB) are skipped to preserve RAM for high-frequency small assets.
func (c *MemoryCache) Set(key string, data []byte) {
	if !c.enabled {
		return
	}

	c.Lock()
	defer c.Unlock()

	size := int64(len(data))

	// Safety Check: Single item shouldn't take more than 50% of the cache.
	if size > c.maxSize/2 {
		return
	}

	// Optimization Strategy:
	// Files larger than 512KB are better handled by the OS Page Cache (SQLite).
	// Storing them in Go Heap creates GC pressure. We strictly cache small avatars/thumbnails.
	if size > 512*1024 {
		return
	}

	// Eviction Strategy: If full, make room.
	if c.totalSize+size > c.maxSize {
		c.prune(size)
	}

	// Overwrite logic: Remove old size before adding new
	if oldItem, exists := c.items[key]; exists {
		c.totalSize -= oldItem.Size
	}

	c.items[key] = Item{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
		Size:      size,
	}
	c.totalSize += size
}

// Get retrieves an item if it exists and hasn't expired.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}

	c.RLock()
	defer c.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}
	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Data, true
}

// Delete explicitly removes an item from the cache.
func (c *MemoryCache) Delete(key string) {
	if !c.enabled {
		return
	}

	c.Lock()
	defer c.Unlock()

	if item, found := c.items[key]; found {
		delete(c.items, key)
		c.totalSize -= item.Size
		// log.Printf("🧹 Cache Invalidated: %s", key)
	}
}

// prune evicts items sorted by expiration time until memory usage drops below 80%.
// Note: This operation holds the Write Lock.
func (c *MemoryCache) prune(needed int64) {
	// Theoretically, it won't come here, but I wanted to use it anyway.
	if c.items == nil || len(c.items) == 0 {
		return
	}

	// Target: Free up to 20% of capacity to avoid frequent pruning
	targetSize := int64(float64(c.maxSize) * 0.80)

	type candidate struct {
		Key       string
		ExpiresAt time.Time
		Size      int64
	}

	// Collect candidates (O(N) allocation)
	candidates := make([]candidate, 0, len(c.items))
	for k, v := range c.items {
		candidates = append(candidates, candidate{k, v.ExpiresAt, v.Size})
	}

	// Sort by Expiration: Delete items that will expire soonest first.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ExpiresAt.Before(candidates[j].ExpiresAt)
	})

	for _, cand := range candidates {
		if c.totalSize <= targetSize {
			break
		}

		delete(c.items, cand.Key)
		c.totalSize -= cand.Size
	}
}

// startGC is a background worker that removes expired items.
func (c *MemoryCache) startGC() {
	ticker := time.NewTicker(GCInterval)
	for range ticker.C {
		c.Lock() // Write Lock
		if c.items == nil || len(c.items) == 0 {
			c.Unlock()
			continue
		}
		now := time.Now()
		removedCount := 0
		removedBytes := int64(0)

		for k, v := range c.items {
			if now.After(v.ExpiresAt) {
				delete(c.items, k)
				c.totalSize -= v.Size
				removedBytes += v.Size
				removedCount++
			}
		}
		c.Unlock()

		if removedCount > 0 {
			log.Printf("[CACHE] GC: Cleaned %d items (%s freed)", removedCount, utils.FormatBytes(removedBytes))
		}
	}
}

// startMonitor logs cache statistics periodically.
func (c *MemoryCache) startMonitor() {
	ticker := time.NewTicker(MonitorInterval)
	for range ticker.C {
		c.RLock()
		if c.items == nil || len(c.items) == 0 {
			c.RUnlock()
			continue
		}

		count := len(c.items)
		used := c.totalSize
		max := c.maxSize
		c.RUnlock()

		percent := 0.0
		if max > 0 {
			percent = (float64(used) / float64(max)) * 100
		}

		log.Printf("[CACHE] Cache: %d items | Usage: %s / %s (%.2f%%)",
			count,
			utils.FormatBytes(used),
			utils.FormatBytes(max),
			percent,
		)
	}
}
