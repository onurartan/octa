package handlers

import (
	"context"
	"fmt"

	"octa/internal/appinfo"
	"octa/internal/database"
	"octa/pkg/utils"
)

// CoreDeleteAsset performs a safe, transactional deletion of an asset.
// It handles database records, key mappings, and cache invalidation.
func CoreDeleteAsset(ctx context.Context, assetID string) error {
	tx := database.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer tx.Rollback()

	var sizeToDelete int64
	if err := tx.Model(&database.Image{}).Where("id = ?", assetID).Select("size").Scan(&sizeToDelete).Error; err != nil {
		return fmt.Errorf("failed to fetch image size: %w", err)
	}

	var keys []string
	if err := tx.Model(&database.KeyMapping{}).Where("image_id = ?", assetID).Pluck("key", &keys).Error; err != nil {
		return fmt.Errorf("failed to fetch associated keys: %w", err)
	}

	// Delete Key Mappings (Children First)
	if err := tx.Where("image_id = ?", assetID).Delete(&database.KeyMapping{}).Error; err != nil {
		return fmt.Errorf("failed to delete mappings: %w", err)
	}

	// Delete the Photo (Then Dad)
	result := tx.Where("id = ?", assetID).Delete(&database.Image{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete image blob: %w", result.Error)
	}

	// If no rows have been deleted, the ID is incorrect.
	if result.RowsAffected == 0 {
		return utils.ErrAssetNotFound
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("transaction commit failed: %w", err)
	}

	appinfo.RemoveAsset(sizeToDelete)

	if globalCache != nil {
		for _, k := range keys {
			globalCache.Delete("map:" + k)
		}

		globalCache.Delete("img:" + assetID)
	}

	return nil
}
