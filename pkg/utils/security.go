package utils

import (
	"octa/internal/config"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
)

// generateSessionHash creates a deterministic hash for the session.
// Format: SHA256(username + ":" + password)
func GenerateSessionHash(user, pass string) string {
	hash := sha256.Sum256([]byte(user + ":" + pass + ":octa_static_salt"))
	return hex.EncodeToString(hash[:])
}

// ParseInt safely parses a string to int with bounds checking.
// Usage: ParseInt("500", 256, 16, 2048) -> Returns 500
// Usage: ParseInt("abc", 256, 16, 2048) -> Returns 256 (Default)
// Usage: ParseInt("9999", 256, 16, 2048) -> Returns 2048 (Max)
func ParseInt(value string, def int, min int, max int) int {
	if value == "" {
		return def
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
}

// IsValidKeyFormat checks if the string contains only allowed characters.
// Allowed: a-z, A-Z, 0-9, -, _, /, @
// Performance: O(n) - No Regex overhead.
func IsValidKeyFormat(k string) bool {
	if k == "" {
		return false
	}

	for _, r := range k {
		if (r >= 'a' && r <= 'z') || // Lowercase
			(r >= 'A' && r <= 'Z') || // Uppercase
			(r >= '0' && r <= '9') || // Number
			r == '-' || r == '_' ||
			r == '/' || r == '@' {
			continue
		}

		return false
	}
	return true
}

func IsAllowedOrigin(origin string) bool {
	allowedPatterns := config.AppConfig.Security.CorsOrigins

	if origin != "" {
		cleanOrigin := getCleanOrigin(origin)

		for _, pattern := range allowedPatterns {
			if MatchOrigin(cleanOrigin, pattern) {
				return true
			}
		}
	}

	return false
}

func getCleanOrigin(originURL string) string {

	u, err := url.Parse(originURL)
	if err != nil {
		return originURL
	}

	if u.Scheme != "" && u.Host != "" {
		return u.Scheme + "://" + u.Host
	}

	return originURL
}

func MatchOrigin(origin, pattern string) bool {
	// Pattern “*” accepts everything
	if pattern == "*" {
		return true
	}

	// Exact Match
	if origin == pattern {
		return true
	}

	// “**.example.com” (Main Domain + Subdomains)
	if strings.Contains(pattern, "**.") {
		base := strings.Replace(pattern, "**.", "", 1) // "https://**.example.com" -> "https://example.com"

		// Is it the main domain?
		if origin == base {
			return true
		}

		// Is it a subdomain? (https://api.example.com)
		// Remove the protocol from the base: “example.com”
		domainPart := removeProtocol(base)

		if strings.HasSuffix(origin, "."+domainPart) {
			return true
		}
	}

	// 3. “*.example.com” (Subdomains Only)
	if strings.Contains(pattern, "*.") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix := parts[0] // "https://"
			suffix := parts[1] // ".example.com"

			if strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix) {

				middle := origin[len(prefix) : len(origin)-len(suffix)]

				if !strings.Contains(middle, "/") {
					return true
				}
			}
		}
	}

	return false
}

func removeProtocol(urlStr string) string {
	urlStr = strings.TrimPrefix(urlStr, "https://")
	return strings.TrimPrefix(urlStr, "http://")
}
