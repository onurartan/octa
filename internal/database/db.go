package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"octa/internal/appinfo"
	"octa/internal/config"
	"octa/pkg/logger"
)

var DB *gorm.DB

// InitDB initializes the SQLite connection with performance-tuned settings (WAL mode).
// It handles directory creation, connection pooling configuration, schema migrations,
// and pre-loading of statistical data.
//
// The application will terminate if the database connection cannot be established.
func InitDB() {
	dbPath := config.AppConfig.Database.Path

	if err := ensureDir(dbPath); err != nil {
		log.Fatalf("[FATAL] Failed to ensure database directory: %v", err)
	}

	// WAL mode enables concurrent readers and a single writer without locking the entire file.
	// busy_timeout ensures the driver waits for the lock instead of failing immediately.
	dsn := fmt.Sprintf(
		"%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-20000",
		dbPath,
	)

	gormConfig := &gorm.Config{
		Logger:                 gormLogger.Default.LogMode(gormLogger.Silent),
		PrepareStmt:            true,
		SkipDefaultTransaction: true, // Improves write performance by ~30%
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}

	configurePool(DB)
	runMigrations(DB)
	loadInitialStats(DB)

		logger.LogInfo("Database initialized successfully")
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0750) // 0750: Restricted access for security
	}
	return nil
}

func configurePool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("[FATAL] Failed to retrieve generic database interface: %v", err)
	}

	// Limit concurrency to prevent disk I/O throttling on the single SQLite file.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)
}

func runMigrations(db *gorm.DB) {
	if err := db.AutoMigrate(&Image{}, &KeyMapping{}); err != nil {
		log.Fatalf("[FATAL] Schema migration failed: %v", err)
	}

	// Raw SQL is used here to ensure idempotent index creation
	indices := []string{
		"CREATE INDEX IF NOT EXISTS idx_images_updated_at ON images(updated_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_key_mappings_image_id ON key_mappings(image_id);",
	}

	for _, idx := range indices {
		if err := db.Exec(idx).Error; err != nil {
			logger.LogWarn("Failed to create index: %v", err)
		}
	}
}

func loadInitialStats(db *gorm.DB) {
	var count int64
	var totalSize int64

	// IFNULL is required to handle the case where the table is empty (returns 0 instead of NULL)
	row := db.Model(&Image{}).Select("count(*), IFNULL(SUM(size), 0)").Row()
	
	if err := row.Scan(&count, &totalSize); err != nil {
			logger.LogWarn("Failed to load initial stats: %v", err)
		return
	}

	appinfo.SetInitialStats(count, totalSize)
}