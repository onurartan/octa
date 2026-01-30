package handlers

import (
	"crypto/subtle"
	"encoding/json"

	"strings"
	"time"

	"net/http"
	"sync"

	"octa/internal/config"
	"octa/pkg/utils"

	"golang.org/x/time/rate"
)

// Login RATE LIMITER (Brute Force Protection)
var loginVisitors = make(map[string]*rate.Limiter)
var loginMu sync.Mutex

// getLoginVisitor creates a strict rate limiter specifically for login endpoints.
// Limits: 1 request/sec, Burst: 10.
func getLoginVisitor(ip string) *rate.Limiter {
	loginMu.Lock()
	defer loginMu.Unlock()

	limiter, exists := loginVisitors[ip]
	if !exists {
		limiter = rate.NewLimiter(1, 10)
		loginVisitors[ip] = limiter
	}
	return limiter
}

// LoginRateLimitMiddleware enforces strict limits on authentication attempts.
func LoginRateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := utils.GetRealIP(r)

		limiter := getLoginVisitor(ip)
		if !limiter.Allow() {
			utils.WriteError(w, http.StatusTooManyRequests, utils.ErrAuthRateLimitExceed, "Too many login attempts. Please wait.")
			return
		}
		next(w, r)
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler validates credentials and sets a secure HTTP-only cookie.
// It uses constant-time comparison to prevent timing attacks.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds LoginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Invalid request body.")
		return
	}

	expectedUser := config.AppConfig.ConsoleUI.User.Username
	expectedPass := config.AppConfig.ConsoleUI.User.Password

	// Even if username is wrong, we check password to keep response time consistent.
	userMatch := subtle.ConstantTimeCompare([]byte(creds.Username), []byte(expectedUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(creds.Password), []byte(expectedPass)) == 1

	if !userMatch || !passMatch {
		// Artificial delay to slow down brute-force scripts
		time.Sleep(500 * time.Millisecond)
		utils.WriteError(w, http.StatusUnauthorized, utils.ErrAuthInvalid, "Incorrect username or password.")
		return
	}

	sessionToken := utils.GenerateSessionHash(expectedUser, expectedPass)

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,                            // JavaScript access forbidden (XSS protection)
		Secure:   r.TLS != nil,                    // True if using HTTPS
		SameSite: http.SameSiteLaxMode,            // CSRF
		Expires:  time.Now().Add(720 * time.Hour), // 30 Days
	})

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"action":  "logged_in",
		"message": "Login successful.",
	})
}

// LogoutHandler invalidates the authentication cookie.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0), 
		MaxAge:   -1,
	})

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"action":  "logged_out",
		"message": "Logged out successfully.",
	})
}

// AuthMiddleware protects routes by verifying the session cookie.
// It handles both API clients (401 JSON) and Browsers (Redirect).
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				utils.WriteError(w, http.StatusUnauthorized, utils.ErrAuthRequired, "Session expired or invalid.")
				return
			}

			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

// IsAuthenticated verifies the session cookie and returns true if valid.
func IsAuthenticated(r *http.Request) bool {
	c, err := r.Cookie("auth_token")
	if err != nil {
		return false
	}

	expectedToken := utils.GenerateSessionHash(
		config.AppConfig.ConsoleUI.User.Username,
		config.AppConfig.ConsoleUI.User.Password,
	)

	return c.Value == expectedToken
}
