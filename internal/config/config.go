package config

import (
	"fmt"
	"log"

	"strings"
	"time"

	"github.com/spf13/viper"

	"octa/pkg/logger"
)

var AppConfig *Config

func (c *Config) GetBaseUrl() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return fmt.Sprintf("http://localhost:%d", c.Server.Port)
}
func Load() {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	v.SetEnvPrefix("OCTA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// if err := v.ReadInConfig(); err != nil {
	// 	log.Printf("⚠️  Config file not found, using defaults & env. Error: %v", err)
	// }

	// v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// v.AutomaticEnv()

	v.BindEnv("database.path", "AVATAR_DATABASE_PATH")

	v.BindEnv("security.upload_secret", "AVATAR_SECURITY_UPLOAD_SECRET")

	v.BindEnv("consoleui.user.username", "ADMIN_DASHBOARD_USERNAME")
	
	v.BindEnv("consoleui.user.password", "ADMIN_DASHBOARD_PASSWORD")

	v.BindEnv("server.port", "APP_PORT")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.LogInfo("Config file not found. Using Environment Variables and Defaults.")
		} else {
			logger.LogWarn("Config file found but unreadable: %v", err)
		}
	}

	if err := v.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("[CRITICAL] Error: Failed to parse configuration: %v", err)
	}

	AppConfig.BaseURL = AppConfig.GetBaseUrl()

	if err := AppConfig.Validate(); err != nil {
		log.Fatalf("[FATAL] CONFIGURATION ERROR: %v", err)
	}

	logger.LogInfo("⚙️  %s v%s Initialized | Env: %s | Port: %d",
		AppConfig.App.Name,
		AppConfig.App.Version,
		AppConfig.Server.Env,
		AppConfig.Server.Port,
	)
}

func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "Octa")
	v.SetDefault("app.version", "0.0.1")
	// REMOVED
	// v.SetDefault("app.landing_page", true)

	// Server
	v.SetDefault("server.port", 9980)
	v.SetDefault("server.env", "development")

	// Image Engine
	v.SetDefault("image.size", 256)
	v.SetDefault("image.quality", 80)
	v.SetDefault("image.max_upload_size", "5MB")
	v.SetDefault("image.max_key_limit", 7)

	// Caching
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.max_capacity", 100) // 100 MB
	v.SetDefault("cache.ttl", "30m")

	// Security & Limits
	v.SetDefault("security.rate_limit.enabled", true)
	v.SetDefault("security.rate_limit.requests", 20)
	v.SetDefault("security.rate_limit.window", "1s")
	v.SetDefault("security.rate_limit.burst", 50)

	// Console UI
	v.SetDefault("consoleui.enabled", true)

	// Database
	v.SetDefault("database.max_size", "2GB")
	v.SetDefault("database.prune_interval", "5m")
}

func (c *Config) Validate() error {
	// Security: Upload Secret Check
	if c.Security.UploadSecret == "" || c.Security.UploadSecret == "secret" {
		if c.Server.Env == "production" {
			return fmt.Errorf("security.upload_secret cannot be default or empty in production environment")
		}
		logger.LogWarn("Security Alert: Using unsafe default Upload Secret. Do not use this in production!")
	}

	// Cache: TTL Parsing Check
	if _, err := time.ParseDuration(c.Cache.TTL); err != nil {
		return fmt.Errorf("invalid cache.ttl format '%s': %v", c.Cache.TTL, err)
	}

	// RateLimit: Window Parsing Check
	if _, err := time.ParseDuration(c.Security.RateLimit.Window); err != nil {
		return fmt.Errorf("invalid rate_limit.window format '%s': %v", c.Security.RateLimit.Window, err)
	}

	// Console UI Credentials Check
	if c.ConsoleUI.Enabled {

		if c.ConsoleUI.User.Username == "" || c.ConsoleUI.User.Password == "" {
			return fmt.Errorf(
				"consoleui is enabled but credentials are missing. " +
					"Set 'consoleui.user.username/password' in config.yaml or use " +
					"ADMIN_DASHBOARD_USERNAME / ADMIN_DASHBOARD_PASSWORD env vars",
			)
		}
	}
	return nil
}
