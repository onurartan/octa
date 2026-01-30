package middleware

import (
	"net/http"
	"sync"
	"time"

	"octa/internal/config"
	"octa/pkg/utils"

	"golang.org/x/time/rate"
)

// Configuration
const (
	// Rate Limit Rules
	DefaultRequests = 20 // Steady state rate (token refilling speed)

	BurstSize = 50 // Max burst capacity (bucket size) for traffic spikes

	// Garbage Collection
	VisitorTTL      = 5 * time.Minute // Time before an inactive IP is removed from memory
	CleanupInterval = 3 * time.Minute // Frequency of the cleanup routine
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

func init() {
	go startCleanupRoutine()
}

// startCleanupRoutine runs in the background to remove stale visitor entries,
// preventing memory leaks over time.
func startCleanupRoutine() {
	ticker := time.NewTicker(CleanupInterval)
	for range ticker.C {
		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > VisitorTTL {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		conf := config.AppConfig.Security.RateLimit

		windowDuration, _ := time.ParseDuration(conf.Window)
		if windowDuration == 0 {
			windowDuration = time.Second
		}

		request := conf.Requests

		if request == 0 {
			request = DefaultRequests
		}

		rps := float64(request) / windowDuration.Seconds()

		burst := conf.Burst
		if burst == 0 {
			burst = BurstSize
		}

		limiter := rate.NewLimiter(rate.Limit(rps), burst)

		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// RateLimitMiddleware enforces request quotas per IP address.
// Blocks excessive requests with a 429 JSON response.
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !config.AppConfig.Security.RateLimit.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip := utils.GetRealIP(r)
		limiter := getVisitor(ip)

		if !limiter.Allow() {
			utils.WriteError(
				w,
				http.StatusTooManyRequests,
				utils.ErrRequestRateLimitExceeded,
				"Too many requests. Please wait a moment.",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
