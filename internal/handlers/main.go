package handlers

import (
		"golang.org/x/sync/singleflight"
		"octa/pkg/cache"
)

var (
	// Global in-memory cache with 100MB limit
	globalCache *cache.MemoryCache

	// SingleFlight group to prevent cache stampedes
	requestGroup singleflight.Group
)


func SetCache(c *cache.MemoryCache) {
    globalCache = c
}