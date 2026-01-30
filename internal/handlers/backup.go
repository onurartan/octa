package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"octa/internal/database"
	"octa/pkg/utils"
)

var (
	backupMutex sync.Mutex
)

// BackupHandler generates a point-in-time snapshot of the SQLite database.
// It is protected by AuthMiddleware to ensure only authorized admins can trigger it.
func BackupHandler(w http.ResponseWriter, r *http.Request) {

	// Ensure only one backup runs at a time to prevent resource exhaustion.
	if !backupMutex.TryLock() {
		utils.WriteError(w, http.StatusTooManyRequests, utils.ErrBackupConcurrencyLimit, "Another backup is currently in progress.")
		return
	}
	defer backupMutex.Unlock()

	// Even with a cookie, we check if the request actually came from our own admin dashboard.
	referer := r.Header.Get("Referer")
	if !utils.IsAllowedOrigin(referer) {
		utils.WriteError(w, http.StatusForbidden, utils.ErrRequestForbidden, "Requests must originate from the dashboard.")
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("octa_vault_%s.db", timestamp)

	// Use OS temp directory for isolation
	tempPath := filepath.Join(os.TempDir(), filename)

	// ATOMIC DATABASE SNAPSHOT
	// VACUUM INTO creates a consistent copy without locking the live database.
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	query := fmt.Sprintf("VACUUM INTO '%s'", tempPath)
	if err := database.DB.WithContext(ctx).Exec(query).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Internal database snapshot failed.")
		return
	}

	// Immediate cleanup after the function exits.
	defer func() {
		if err := os.Remove(tempPath); err != nil {
		}
	}()

	info, err := os.Stat(tempPath)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, "Failed to verify backup integrity.")
		return
	}

	// Security Headers to prevent browser sniffing and unintended execution
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/x-sqlite3")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")

	http.ServeFile(w, r, tempPath)
}
