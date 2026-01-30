package appinfo

import (
	"sync/atomic"
)

var (
	TotalAssetsCount atomic.Int64
	TotalAssetsSize  atomic.Int64
)

// AddAsset: Called when a new image is added
func AddAsset(size int64) {
	TotalAssetsCount.Add(1)
	TotalAssetsSize.Add(size)
}

// RemoveAsset: Called when the image is deleted
func RemoveAsset(size int64) {
	TotalAssetsCount.Add(-1)
	TotalAssetsSize.Add(-size)
}

// SetInitialStats: Writes the first data received from the database when the server starts up.
func SetInitialStats(count, size int64) {
	TotalAssetsCount.Store(count)
	TotalAssetsSize.Store(size)
}