package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"octa/internal/appinfo"
	"octa/internal/config"
	"octa/internal/database"
	"octa/pkg/utils"
)

// AssetDTO defines a lightweight representation of an image asset for frontend consumption.
type AssetDTO struct {
	ID        string `json:"id"`
	Keys      string `json:"keys"` // "avatar-1, user-x"
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ExtendedStatsDTO struct {
	TotalCount    int64      `json:"total_count"`
	TotalSize     int64      `json:"total_size"`
	Uptime        string     `json:"uptime"`
	UptimeSeconds int64      `json:"uptime_seconds"`
	RamUsage      uint64     `json:"ram_usage"`
	NumGoroutines int        `json:"num_goroutines"`
	RecentUploads []AssetDTO `json:"recent_uploads"`
	MaxUploadSize string     `json:"max_upload_size"`
}

type PaginatedResponse struct {
	Items      []AssetDTO `json:"items"`
	TotalItems int64      `json:"total_items"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// OLD QUERY WHY I REMOVED
// When I ran benchmark tests, this query was 3x faster than normal queries on databases with light data loads, but when SQLite contained more than 30K images, processing with this SQL code took seconds. That's why I switched to the old, clunky but consistently fast structure.
// const queryAssets = `
//     SELECT
//         i.id, i.updated_at, i.created_at,
//         LENGTH(i.data) as size, i.width, i.height,
//         GROUP_CONCAT(k.key, ', ') as keys
//     FROM images i
//     LEFT JOIN key_mappings k ON k.image_id = i.id
//     GROUP BY i.id
//     ORDER BY i.updated_at DESC
// `

// GetStats returns server health, memory metrics, and recent activity.
// GET /api/admin/stats
func GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	count := appinfo.TotalAssetsCount.Load()
	totalSize := appinfo.TotalAssetsSize.Load()


	// Runtime Metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	type RawResult struct {
		ID        string
		UpdatedAt time.Time
		Size      int64
		Width     int
		Height    int
		Keys      string
	}
	var recentImages []RawResult
	// database.DB.WithContext(r.Context()).Raw(queryAssets + " LIMIT 5").Scan(&results)

	err := database.DB.WithContext(ctx).
		Table("images").
		Select("id, updated_at, size, width, height").
		Order("updated_at DESC").
		Limit(5).
		Scan(&recentImages).Error

	if err != nil {
		recentImages = []RawResult{}
	}

	// baseURL := getBaseURL(r)
	recentAssets := make([]AssetDTO, 0, len(recentImages))

	if len(recentImages) > 0 {
		imageIDs := make([]string, len(recentImages))
		for i, img := range recentImages {
			imageIDs[i] = img.ID
		}

		type KeyResult struct {
			ImageID string
			Key     string
		}
		var keys []KeyResult

		database.DB.WithContext(ctx).
			Table("key_mappings").
			Select("image_id, key").
			Where("image_id IN ?", imageIDs).
			Scan(&keys)

		keysMap := make(map[string][]string)
		for _, k := range keys {
			keysMap[k.ImageID] = append(keysMap[k.ImageID], k.Key)
		}

		baseURL := getBaseURL(r)
		for _, img := range recentImages {
			imgKeys := keysMap[img.ID]
			keysStr := strings.Join(imgKeys, ", ")

			urlKey := img.ID
			if len(imgKeys) > 0 {
				urlKey = imgKeys[0]
			}

			recentAssets = append(recentAssets, AssetDTO{
				ID:        img.ID,
				Keys:      keysStr,
				Size:      img.Size,
				Width:     img.Width,
				Height:    img.Height,
				CreatedAt: img.UpdatedAt.Format("2006-01-02 15:04"),
				URL:       fmt.Sprintf("%s/u/%s", baseURL, strings.TrimSpace(urlKey)),
			})
		}
	}

	stats := ExtendedStatsDTO{
		TotalCount:    count,
		TotalSize:     totalSize,
		Uptime:        time.Since(appinfo.StartTime).String(),
		UptimeSeconds: int64(time.Since(appinfo.StartTime).Seconds()),
		RamUsage:      m.Alloc,
		NumGoroutines: runtime.NumGoroutine(),
		RecentUploads: recentAssets,
		MaxUploadSize: config.AppConfig.Image.MaxUploadSize,
	}

	utils.WriteJSON(w, http.StatusOK, stats)
}

// ListAssets returns a paginated list of all stored assets without binary data.
// GET /api/admin/assets
func ListAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	var results []struct {
		ID        string
		UpdatedAt time.Time
		CreatedAt time.Time
		Size      int64
		Width     int
		Height    int
	}
	var totalItems int64

	if searchQuery == "" {
		totalItems = appinfo.TotalAssetsCount.Load()

		err := database.DB.WithContext(ctx).
			Table("images").
			Select("id, updated_at, created_at, size, width, height").
			Order("updated_at DESC").
			Limit(limit).
			Offset(offset).
			Scan(&results).Error

		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "DB Error")
			return
		}
	} else {

		likeStr := searchQuery
		if !strings.HasSuffix(likeStr, "%") {
			likeStr += "%"
		}

		likeStr = strings.TrimPrefix(likeStr, "%")

		var imageIDs []string
		err := database.DB.Table("key_mappings").
			Where("key LIKE ?", likeStr).
			Distinct("image_id").
			Count(&totalItems).Error

		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Unkown Error")
			return
		}

		if totalItems > 0 {

			err := database.DB.Table("key_mappings").
				Select("DISTINCT image_id").
				Where("key LIKE ?", likeStr).
				Limit(limit).
				Offset(offset).
				Pluck("image_id", &imageIDs).Error

			if err != nil {
				utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Unkown Search Error")
				return
			}
		}

		if len(imageIDs) > 0 {
			database.DB.WithContext(ctx).
				Table("images").
				Select("id, updated_at, created_at, size, width, height").
				Where("id IN ?", imageIDs).
				Scan(&results)
		}
	}

	if len(results) == 0 {
		utils.WriteJSON(w, http.StatusOK, PaginatedResponse{
			Items:      []AssetDTO{},
			TotalItems: totalItems,
			Page:       page,
			Limit:      limit,
			TotalPages: 0,
		})
		return
	}

	resultIDs := make([]string, len(results))
	for i, r := range results {
		resultIDs[i] = r.ID
	}

	type KeyRes struct {
		ImageID string
		Key     string
	}
	var keyRows []KeyRes
	database.DB.Table("key_mappings").
		Select("image_id, key").
		Where("image_id IN ?", resultIDs).
		Scan(&keyRows)

	keysMap := make(map[string][]string)
	for _, k := range keyRows {
		keysMap[k.ImageID] = append(keysMap[k.ImageID], k.Key)
	}

	baseURL := getBaseURL(r)
	assets := make([]AssetDTO, 0, len(results))

	for _, res := range results {
		imgKeys := keysMap[res.ID]
		keysStr := strings.Join(imgKeys, ", ")

		urlKey := res.ID
		if len(imgKeys) > 0 {
			urlKey = imgKeys[0]
		}

		assets = append(assets, AssetDTO{
			ID:        res.ID,
			Keys:      keysStr,
			Size:      res.Size,
			Width:     res.Width,
			Height:    res.Height,
			CreatedAt: res.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt: res.UpdatedAt.Format("2006-01-02 15:04"),
			URL:       fmt.Sprintf("%s/u/%s", baseURL, strings.TrimSpace(urlKey)),
		})
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages < 0 {
		totalPages = 0
	}

	utils.WriteJSON(w, http.StatusOK, PaginatedResponse{
		Items:      assets,
		TotalItems: totalItems,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// DELETE /api/admin/assets/{id}
func DeleteAssetHandler(w http.ResponseWriter, r *http.Request) {

	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Asset ID is required.")
		return
	}

	err := CoreDeleteAsset(r.Context(), id)

	if err != nil {
		if errors.Is(err, utils.ErrAssetNotFound) {
			utils.WriteError(w, http.StatusNotFound, utils.ErrResourceNotFound, "Asset not found.")
		} else {
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Could not delete asset.")
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"action":  "deleted",
		"message": "Asset and associated keys deleted successfully",
		"id":      id,
	})
}

type UpdateKeysRequest struct {
	Keys string `json:"keys"` // e.g., "new-key-1, new-key-2"
}

// UpdateAssetKeys replaces all custom keys/slugs for a specific asset.
// UPDATE /api/admin/assets/{id}
func UpdateAssetKeys(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Asset ID is required.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2048)

	var req UpdateKeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Invalid JSON body.")
		return
	}

	tx := database.DB.WithContext(r.Context()).Begin()

	//  Clear existing keys
	if err := tx.Where("image_id = ?", id).Delete(&database.KeyMapping{}).Error; err != nil {
		tx.Rollback()
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Failed to reset asset keys.")
		return
	}

	// Insert new keys
	newKeys := strings.Split(req.Keys, ",")
	for _, k := range newKeys {
		k = strings.TrimSpace(k)
		k = utils.NormalizeKey(k)
		k = strings.ToLower(k)

		if k == "" || len(k) > 30 {
			continue
		}

		if !utils.IsValidKeyFormat(k) {
			tx.Rollback()
			utils.WriteError(w, http.StatusBadRequest, utils.ErrValidationInvalidFormat,
				fmt.Sprintf("Key '%s' contains invalid characters. Allowed: a-z, 0-9, -, _, /, @", k))
			return
		}

		if err := tx.Create(&database.KeyMapping{Key: k, ImageID: id}).Error; err != nil {
			tx.Rollback()
			// Likely a unique constraint violation
			utils.WriteError(w, http.StatusConflict, utils.ErrResourceConflict, fmt.Sprintf("Key '%s' is already in use.", k))
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Transaction failed.")
		return
	}

	if globalCache != nil {
		newKeysList := strings.Split(req.Keys, ",")
		for _, k := range newKeysList {
			k = utils.NormalizeKey(k)
			if k != "" {

				globalCache.Delete("map:" + k)
			}
		}
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"action":  "updated",
		"message": "Asset keys updated successfully.",
	})
}

// Helper to construct dynamic base URLs (http vs https)
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
