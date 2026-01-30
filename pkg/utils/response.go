package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	// Request Error Codes
	ErrRequestInvalid           = "request/invalid_parameters"
	ErrRequestBadRequest        = "request/bad_request"
	ErrRequestNotFound          = "request/not_found"
	ErrRequestMissingKey        = "request/missing_key"
	ErrRequestRateLimitExceeded = "request/rate_limit_exceeded"
	ErrRequestForbidden         = "request/forbidden"

	ErrRequestBodyTooLarge     = "request/body_too_large"
	ErrRequestUnSupportedMedia = "request/invalid_media"

	// Auth Error Codes
	ErrAuthRequired        = "auth/authentication_required"
	ErrAuthInvalid         = "auth/invalid_credentials"
	ErrAuthRateLimitExceed = "auth/rate_limit_exceeded"

	// Server Error Codes
	ErrServerInternal = "server/internal_error"
	ErrServerTimeout  = "server/timeout"

	// Validation & Resource Error Codes
	ErrValidationInvalidFormat = "validation/invalid_format"
	ErrResourceNotFound        = "resource/not_found"
	ErrResourceConflict        = "resource/conflict"

	// Others
	ErrImageGenerationFailed = "image/generation_failed"
	ErrImageProcessingFailed = "image/processing_failed"
	ErrUpstreamFailed        = "upstream/service_failed" // Github vs.

	ErrBackupConcurrencyLimit = "backup/concurrency_limit"
	ErrBackupForbiddenOrigin  = "backup/forbidden_origin"
)

var (
	ErrAssetNotFound = errors.New("asset not found")
)

type APIError struct {
	Code    string `json:"code"`    // e.g., "request/invalid_parameters"
	Message string `json:"message"` // User-friendly message
	Status  int    `json:"status"`  // HTTP Status Code
}

// WriteError sends a JSON formatted error response
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	fmt.Println(code, ": ", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Code:    code,
		Message: message,
		Status:  status,
	})
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
