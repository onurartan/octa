package database

import (
	"os"
	"time"

	"octa/internal/config"
	"octa/pkg/logger"
	"octa/pkg/utils"
)

/*
WORKER DETAILS: Smart Storage Management Strategy
=================================================

This worker implements a hybrid maintenance strategy to manage SQLite storage efficiently
without compromising I/O performance.

1. Hysteresis / Allocation Buffer:
   We do not shrink the database file immediately after deletions. Keeping the file size
   constant allows SQLite to overwrite freed pages with new data, avoiding expensive
   OS-level file allocation calls (ftruncate/fallocate).

2. Dual-Mode Operation:
   The worker checks the database state periodically and decides between two actions:

   Mode A: VACUUM (De-bloat)
   - Trigger: Physical file size > Limit AND Logical data size is low (>50% empty space).
   - Action: Rebuilds the database file to reclaim disk space.
   - Use Case: Occurs after massive deletions (e.g., post-benchmark cleanup).

   Mode B: PRUNE (Retention Policy)
   - Trigger: Physical file size > Limit AND database is logically full.
   - Action: Deletes the oldest records (LRU) until size drops to 85% of the limit.
   - Use Case: Normal operation when storage capacity is reached.

3. Safety:
   - Uses `PRAGMA wal_checkpoint(TRUNCATE)` before vacuuming to commit pending WAL transactions.
   - Pruning is batched (50 items at a time) with sleeps to prevent DB locking.
*/

// StartCleaner initializes the background storage maintenance worker.
// It runs periodically based on the configuration interval.
func StartCleaner() {
	maxSizeStr := config.AppConfig.Database.MaxSize
	maxSize := utils.SizeToBytes(maxSizeStr, 2*1024*1024*1024) // Default 2GB

	intervalStr := config.AppConfig.Database.PruneInterval
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		interval = 30 * time.Minute
	}

	logger.LogInfo("Storage Cleaner started. Limit: %s, Interval: %s", maxSizeStr, interval)

	ticker := time.NewTicker(interval)

	// Run immediately on startup to fix potential "Zombie/Bloated" states from previous runs.
	go checkAndPrune(maxSize)

	for range ticker.C {
		checkAndPrune(maxSize)
	}
}

// checkAndPrune analyzes the database size and performs Vacuum or Prune operations.
func checkAndPrune(limitBytes int64) {
	dbPath := config.AppConfig.Database.Path

	// 1. Check Physical Size (Disk Usage)
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
	
		logger.LogError("Cleaner failed to stat DB file: %v", err)
		return
	}

	physicalSize := fileInfo.Size()
	// Include WAL file in size calculation as it consumes disk space
	if walInfo, err := os.Stat(dbPath + "-wal"); err == nil {
		physicalSize += walInfo.Size()
	}

	// Performance Optimization:
	// If below limit, do nothing. We keep the allocated space for future writes.
	if physicalSize < limitBytes {
		return
	}

	// 2. Check Logical Size (Actual Data Usage)
	var logicalSize int64
	row := DB.Model(&Image{}).Select("IFNULL(SUM(size), 0)").Row()
	if err := row.Scan(&logicalSize); err != nil {
		
		logger.LogError("[ERR] Failed to calculate logical size: %v", err)
		return
	}

	// Calculate "Bloat" (Empty space inside the file)
	emptySpace := physicalSize - logicalSize
	isBloated := float64(emptySpace) > (float64(physicalSize) * 0.50)



	logger.LogInfo("Storage Analysis - Phys: %s | Logic: %s | Free: %s",
		utils.FormatBytes(physicalSize),
		utils.FormatBytes(logicalSize),
		utils.FormatBytes(emptySpace))

	// MODE A: VACUUM (The file is large but mostly empty)
	if isBloated {
	

		logger.LogWarn("DB is bloated (>50% empty). Starting VACUUM to reclaim space...")

		// Safety: Commit WAL to main DB before vacuuming to prevent data loss risk
		DB.Exec("PRAGMA wal_checkpoint(TRUNCATE);")

		// Vacuum rebuilds the DB file. This is blocking but necessary here.
		startTime := time.Now()
		if err := DB.Exec("VACUUM;").Error; err != nil {
			
					logger.LogError("VACUUM failed: %v", err)
		} else {
			

			logger.LogInfo("VACUUM completed in %v. Disk space reclaimed.", time.Since(startTime))
		}
		return
	}

	// MODE B: PRUNE (The file is full of data)
	// Target: Reduce to 85% of the limit to create a buffer for new uploads.
	targetSize := int64(float64(limitBytes) * 0.85)
	bytesToRemove := logicalSize - targetSize

	if bytesToRemove <= 0 {
		return
	}


	logger.LogInfo("Storage limit reached. Pruning ~%s of old data...", utils.FormatBytes(bytesToRemove))

	deletedCount := 0
	var freedBytes int64 = 0
	loopGuard := 0

	// Batch processing to avoid long locks
	for freedBytes < bytesToRemove && loopGuard < 1000 {
		loopGuard++
		var images []Image

		// Fetch oldest images (LRU strategy)
		if err := DB.Select("id, size").Order("updated_at ASC").Limit(50).Find(&images).Error; err != nil {
			logger.LogError("Prune fetch failed: %v", err)
			break
		}

		if len(images) == 0 {
			break
		}

		idsToDelete := make([]string, 0, len(images))
		for _, img := range images {
			idsToDelete = append(idsToDelete, img.ID)
			freedBytes += img.Size
		}

		// Delete batch
		if err := DB.Where("id IN ?", idsToDelete).Delete(&Image{}).Error; err != nil {
			

				logger.LogError("Prune delete failed: %v", err)
			break
		}

		deletedCount += len(idsToDelete)
		
		time.Sleep(50 * time.Millisecond)
	}



	
	logger.LogInfo("Pruning complete. Removed %d items.", deletedCount)
}
