package config

type Config struct {
	// App: Global application metadata and landing page behavior
	App InConfigAppConfig `mapstructure:"app"`

	// Server: Network configuration and execution environment
	Server ServerConfig `mapstructure:"server"`

	// Database: SQLite engine parameters and retention policies
	Database DatabaseConfig `mapstructure:"database"`

	// Image: Global constraints for dynamic generation and uploads
	Image ImageConfig `mapstructure:"image"`

	// Cache: In-memory LRU cache settings to reduce Disk I/O
	Cache CacheConfig `mapstructure:"cache"`

	// Security: Authentication, CORS whitelist, and DDoS protection
	Security SecurityConfig `mapstructure:"security"`

	// BaseURL: The public-facing root URL used for absolute link generation
	BaseURL string `mapstructure:"base_url"`

	// ConsoleUI: Administrative dashboard access and credentials
	ConsoleUI ConsoleUIConfig `mapstructure:"consoleui"`
}

type InConfigAppConfig struct {
	// Name: Identity of the service used in headers and dashboard (e.g., "Octa")
	Name string `mapstructure:"name"`

	// Version: Application semantic version (e.g., "0.0.1")
	Version string `mapstructure:"version"`

	StartMessage bool `mapstructure:"start_message"`

	// REMOVED
	// LandingPage: Toggle to enable/disable the built-in welcome screen at root "/"
	// LandingPage bool `mapstructure:"landing_page"`
}

type ServerConfig struct {
	// Port: The TCP port the HTTP server will bind to (default: 9980)
	Port int `mapstructure:"port"`

	// Env: Execution context (development, staging, production)
	Env string `mapstructure:"env"`
}

type DatabaseConfig struct {
	// Path: Physical location of the SQLite database file (e.g., ./data/octa.db)
	Path string `mapstructure:"path"`

	// MaxSize: Soft limit for DB size before pruning triggers (e.g., "2GB")
	MaxSize string `mapstructure:"max_size"`

	// PruneInterval: Frequency of background cleanup tasks (e.g., "5m", "1h")
	PruneInterval string `mapstructure:"prune_interval"`
}

type ImageConfig struct {
	// DefaultSize: Fallback dimensions for avatars if not specified in request (e.g., 360)
	DefaultSize int `mapstructure:"default_size"`

	// Quality: Compression level for image output (1-100)
	Quality int `mapstructure:"quality"`


	// MaxUploadSize: Maximum payload size for the /upload endpoint (e.g., "5MB")
	MaxUploadSize string `mapstructure:"max_upload_size"`

	// MaxKeyLimit: Maximum number of aliases allowed for a single asset mapping (e.g., 7)
	MaxKeyLimit int `mapstructure:"max_key_limit"`
}

type CacheConfig struct {
	// Enabled: Toggles the in-memory asset caching layer
	Enabled bool `mapstructure:"enabled"`

	// MaxCapacity: Maximum RAM allocated for cache in MB (e.g., 100)
	MaxCapacity int `mapstructure:"max_capacity"`

	// TTL: Expiration time for cached items (e.g., "30m", "24h")
	TTL string `mapstructure:"ttl"`
}

type SecurityConfig struct {
	// UploadSecret: Static token required in X-Upload-Secret header for write operations
	UploadSecret string `mapstructure:"upload_secret"`

	// CorsOrigins: List of allowed domains for browser-based cross-origin requests
	CorsOrigins []string `mapstructure:"cors_origins"`

	// RateLimit: DDoS protection logic using a token-bucket algorithm
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
	// Enabled: Global toggle for the rate limiting middleware
	Enabled bool `mapstructure:"enabled"`

	// Requests: Number of allowed requests per time window
	Requests int `mapstructure:"requests"`

	// Window: The timeframe for the request limit (e.g., "1s", "1m")
	Window string `mapstructure:"window"`

	// Burst: Temporary allowed spike capacity above the steady-rate limit
	Burst int `mapstructure:"burst"`
}

type ConsoleUIConfig struct {
	// Enabled: Toggles the built-in administrative dashboard
	Enabled bool `mapstructure:"enabled"`

	// User: Basic Auth credentials for dashboard access
	User struct {
		// Username: Admin login identifier
		Username string `mapstructure:"username"`
		// Password: Admin login secret
		Password string `mapstructure:"password"`
	} `mapstructure:"user"`
}