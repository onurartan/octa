package handlers

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"image"
	_ "image/gif"  // Support GIF
	_ "image/jpeg" // Support JPEG
	_ "image/png"  // Support PNG
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"octa/internal/appinfo"
	"octa/internal/config"
	"octa/internal/database"

	"octa/pkg/utils"
)

const (
	DefaultMaxUploadSize = 5 << 20 // 5 MB
	DefaultMaxKeyLimit   = 7       // Max slugs per asset

	// MaxConcurrentDBOps limits the number of active SQLite write transactions.
	// Since SQLite allows only one writer at a time (even in WAL mode),
	// queueing requests in Go memory is more efficient than locking the DB file.
	MaxConcurrentDBOps = 10
)

// dbGuard acts as a semaphore to limit concurrent database writes.
// Buffered channel with capacity = MaxConcurrentDBOps.
var dbGuard = make(chan struct{}, MaxConcurrentDBOps)

// UploadHandler processes image uploads via multipart/form-data.
// It includes a concurrency guard to prevent SQLite 'database is locked' errors
// under heavy load (e.g., benchmarking or DDoS).
//
// Security: Protected by 'X-Secret-Key'.
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	//  Method Validation
	if r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, utils.ErrRequestInvalid, "Only POST allowed.")
		return
	}

	//  Security Check (Constant Time)
	clientSecret := r.Header.Get("X-Secret-Key")
	serverSecret := config.AppConfig.Security.UploadSecret
	if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(serverSecret)) != 1 {
		utils.WriteError(w, http.StatusForbidden, utils.ErrAuthInvalid, "Invalid secret key.")
		return
	}

	// Request Limits & Parsing
	maxKeyLimit := config.AppConfig.Image.MaxKeyLimit
	if maxKeyLimit == 0 {
		maxKeyLimit = DefaultMaxKeyLimit
	}

	maxUploadSize := utils.SizeToBytes(config.AppConfig.Image.MaxUploadSize, DefaultMaxUploadSize)
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		utils.WriteError(w, http.StatusBadRequest,  utils.ErrRequestBodyTooLarge, "File exceeds size limit.")
		return
	}

	// Validate Keys
	keysStr := r.FormValue("keys")
	validKeys := parseKeys(keysStr)

	if len(validKeys) == 0 {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "At least one valid key is required.")
		return
	}
	if len(validKeys) > maxKeyLimit {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Too many keys provided.")
		return
	}

	// File Validation
	file, header, err := r.FormFile("avatar")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Missing 'avatar' file field.")
		return
	}
	defer file.Close()

	if !utils.IsImageFile(header) {
		utils.WriteError(w, http.StatusUnsupportedMediaType, utils.ErrRequestUnSupportedMedia, "Unsupported file type.")
		return
	}

	//  Image Processing (CPU Intensive - Parallelized)
	// We do this BEFORE acquiring the DB lock to maximize throughput.
	finalData, meta, err := processUploadImage(file, r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrImageProcessingFailed, err.Error())
		return
	}

	// This block prevents "database is locked" errors by queueing requests here.
	dbGuard <- struct{}{}
	defer func() { <-dbGuard }() // Release token when function exits

	// Database Transaction (Serialized by Semaphore)
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	primaryKey := validKeys[0] // Authority Key
	var targetAssetID string
	var actionType string
	var oldSize int64 = 0

	var existingMapping database.KeyMapping

	// UPSERT LOGIC
	if err := tx.Where("key = ?", primaryKey).First(&existingMapping).Error; err == nil {
		// UPDATE
		targetAssetID = existingMapping.ImageID
		actionType = "updated"

		tx.Model(&database.Image{}).Where("id = ?", targetAssetID).Select("size").Scan(&oldSize)

		updateData := database.Image{
			Data: finalData, Width: meta.Width, Height: meta.Height, Format: meta.Format, Size: meta.Size,
			UpdatedAt: time.Now(),
		}
		if err := tx.Model(&database.Image{}).Where("id = ?", targetAssetID).Updates(updateData).Error; err != nil {
			tx.Rollback()
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Failed to update image.")
			return
		}
	} else {
		// CREATE
		targetAssetID = uuid.New().String()
		actionType = "created"

		newImage := database.Image{
			ID: targetAssetID, Data: finalData, Width: meta.Width, Height: meta.Height, Format: meta.Format, Size: meta.Size,
		}
		if err := tx.Create(&newImage).Error; err != nil {
			tx.Rollback()
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Failed to save image.")
			return
		}
		if err := tx.Create(&database.KeyMapping{Key: primaryKey, ImageID: targetAssetID}).Error; err != nil {
			tx.Rollback()
			utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Failed to map primary key.")
			return
		}
	}

	// Secondary Keys Logic (Ignore if taken)
	assignedKeys := []string{primaryKey}
	for _, k := range validKeys[1:] {
		var checkMap database.KeyMapping
		if err := tx.Where("key = ?", k).First(&checkMap).Error; err == nil {
			if checkMap.ImageID == targetAssetID {
				assignedKeys = append(assignedKeys, k)
			}
		} else {
			if err := tx.Create(&database.KeyMapping{Key: k, ImageID: targetAssetID}).Error; err == nil {
				assignedKeys = append(assignedKeys, k)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Transaction commit failed.")
		return
	}

	// Post-Transaction (Stats & Cache)
	updateStatsAndCache(actionType, targetAssetID, assignedKeys, meta.Size, oldSize)

		baseURL := config.AppConfig.GetBaseUrl()
	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"action":    actionType,
		"avatar_id": targetAssetID,
		"keys":      assignedKeys,
		"url":      baseURL + "/u/" + primaryKey,
		"size_kb":   meta.Size / 1024,
	})
}

// DeleteAPIHandler handles asset deletion via API.
func DeleteAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, utils.ErrRequestInvalid, "Use DELETE or POST method.")
		return
	}

	clientSecret := r.Header.Get("X-Secret-Key")
	serverSecret := config.AppConfig.Security.UploadSecret
	if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(serverSecret)) != 1 {
		utils.WriteError(w, http.StatusForbidden, utils.ErrAuthInvalid, "Invalid secret key.")
		return
	}

	targetKey := r.URL.Query().Get("key")
	targetID := r.URL.Query().Get("id")

	if targetKey == "" && targetID == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Parameter 'key' or 'id' is required.")
		return
	}

	assetID := targetID
	if assetID == "" {
		var mapping database.KeyMapping
		if err := database.DB.Where("key = ?", targetKey).First(&mapping).Error; err != nil {
			utils.WriteError(w, http.StatusNotFound, utils.ErrResourceNotFound, "Key not found.")
			return
		}
		assetID = mapping.ImageID
	}

	// CoreDeleteAsset logic (assumed to be available or imported)
	// For this snippet, we assume it's a wrapper around DB delete + Cache clear
	if err := database.DB.Where("id = ?", assetID).Delete(&database.Image{}).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Deletion failed.")
		return
	}

	// Clean up mappings
	database.DB.Where("image_id = ?", assetID).Delete(&database.KeyMapping{})

	// Clear Cache
	if globalCache != nil {
		globalCache.Delete("img:" + assetID)
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "success",
		"action": "deleted",
		"target": assetID,
	})
}

// --- Helpers ---

type ImageMeta struct {
	Width, Height int
	Format        string
	Size          int64
}

func parseKeys(keysStr string) []string {
	rawKeys := strings.Split(keysStr, ",")
	validKeys := make([]string, 0, len(rawKeys))
	seenKeys := make(map[string]bool)

	for _, k := range rawKeys {
		k = utils.NormalizeKey(k)
		k = strings.ToLower(strings.TrimSpace(k))
		cleaned := strings.Trim(k, "/")

		if cleaned != "" && utils.IsValidKeyFormat(cleaned) && !seenKeys[cleaned] {
			validKeys = append(validKeys, cleaned)
			seenKeys[cleaned] = true
		}
	}
	return validKeys
}

func processUploadImage(file io.Reader, r *http.Request) ([]byte, ImageMeta, error) {
	var finalData []byte
	var meta ImageMeta

	if r.FormValue("mode") == "original" {
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			return nil, meta, errors.New("failed to read file")
		}
		dcfg, formatName, err := image.DecodeConfig(bytes.NewReader(fileBytes))
		if err != nil {
			return nil, meta, errors.New("file is not a valid image")
		}
		finalData = fileBytes
		meta = ImageMeta{Width: dcfg.Width, Height: dcfg.Height, Format: formatName, Size: int64(len(fileBytes))}
	} else {
		img, _, err := image.Decode(file)
		if err != nil {
			return nil, meta, errors.New("corrupt image data")
		}
		targetSize := utils.ParseInt(r.FormValue("size"), 256, 16, 2048)
		targetScale := utils.ParseInt(r.FormValue("scale"), 75, 1, 100)
		mode := r.FormValue("mode")
		if mode == "" {
			mode = "square"
		}

		buf, w, h, err := utils.ProcessImage(img, utils.ProcessOptions{
			Mode: mode, Size: targetSize, Scale: targetScale, Quality: 85,
		})
		if err != nil {
			return nil, meta, err
		}
		finalData = buf.Bytes()
		meta = ImageMeta{Width: w, Height: h, Format: "jpeg", Size: int64(buf.Len())}
	}
	return finalData, meta, nil
}

func updateStatsAndCache(actionType, assetID string, keys []string, newSize, oldSize int64) {
	if actionType == "updated" {
		appinfo.RemoveAsset(oldSize)
		appinfo.AddAsset(newSize)
		if globalCache != nil {
			globalCache.Delete("img:" + assetID)
		}
	} else {
		appinfo.AddAsset(newSize)
	}

	if globalCache != nil {
		for _, k := range keys {
			globalCache.Delete("map:" + k)
		}
	}
}
