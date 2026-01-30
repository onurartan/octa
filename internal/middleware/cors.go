package middleware

import (
	"net/http"
	"octa/pkg/utils"
)

// CorsMiddleware handles Cross-Origin Resource Sharing with Wildcard Subdomain support.
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestOrigin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")
origin := requestOrigin

if origin == ""{
	origin = referer
}

		isAllowed := utils.IsAllowedOrigin(origin)

		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Secret-Key, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

