package utils

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)


// NormalizeKey cleans a key/slug for storage.
// - trims spaces
// - removes leading/trailing slashes
// - collapses multiple slashes
func NormalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, "/")

	for strings.Contains(key, "//") {
		key = strings.ReplaceAll(key, "//", "/")
	}

	return key
}

func GetRealIP(r *http.Request) string {

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func ConvertToInt(s string, objName string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number(%s): %s", objName, s)
	}
	return result, nil
}
