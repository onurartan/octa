// Package utils provides common helper functions for string manipulation,
// data parsing, and system operations used across the application.
package utils

import (
	"octa/pkg/logger"
	"regexp"
	"strconv"
	"strings"
)

// sizeRegex matches a number followed optionally by a unit string.
// It allows flexible spacing between the number and the unit.
var sizeRegex = regexp.MustCompile(`^(\d+)\s*([a-zA-Z]*)$`)

// unitMultipliers maps data size units to their byte values using binary prefixes (IEC standard).
// 1 KB = 1024 Bytes, 1 MB = 1024 * 1024 Bytes, etc.
var unitMultipliers = map[string]int64{
	"":   1,       // Bytes (default)
	"B":  1,       // Bytes
	"KB": 1 << 10, // Kibibyte (1024)
	"MB": 1 << 20, // Mebibyte (1024^2)
	"GB": 1 << 30, // Gibibyte (1024^3)
	"TB": 1 << 40, // Tebibyte (1024^4)
	"PB": 1 << 50, // Pebibyte (1024^5)
}

// SizeToBytes parses a human-readable data size string into its integer byte representation.
// It supports binary prefixes (KB, MB, GB, TB, PB) where 1KB = 1024 Bytes.
//
// The input string is case-insensitive and tolerates whitespace (e.g., "5MB", "5 MB", "5mb").
//
// Parameters:
//   - sizeStr: The string representing the size (e.g., "5MB", "10GB").
//   - defaultValue: The fallback value returned if parsing fails or the unit is unsupported.
//
// Returns:
//
//	The size in bytes as int64. Returns defaultValue upon error.
func SizeToBytes(sizeStr string, defaultValue int64) int64 {
	rawStr := strings.TrimSpace(strings.ToUpper(sizeStr))
	if rawStr == "" {
		return defaultValue
	}

	matches := sizeRegex.FindStringSubmatch(rawStr)
	if len(matches) != 3 {
		logger.LogWarn(" Utils: Invalid size format '%s', using default.", sizeStr)
		return defaultValue
	}

	// Parse numeric value
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || value <= 0 {
		logger.LogWarn(" Utils: Invalid numeric value in '%s', using default.", sizeStr)
		return defaultValue
	}

	// Validate unit and apply multiplier
	unit := matches[2]
	multiplier, exists := unitMultipliers[unit]
	if !exists {
		logger.LogWarn(" Utils: Unsupported unit '%s' in '%s', using default.", unit, sizeStr)
		return defaultValue
	}

	return value * multiplier
}
