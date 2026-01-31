package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"os"

	"octa/internal/appinfo"
	"octa/internal/config"
	"octa/internal/database"
	"octa/internal/handlers"
	"octa/internal/middleware"
	"octa/pkg/cache"
	"octa/pkg/logger"
	"octa/pkg/utils"
)

type PageData struct {
	BaseURL string
}

func main() {

	utils.LoadEnv()

startupMessageActive := os.Getenv("STARTUP_LOG_ACTIVE")

if startupMessageActive != "false" {
    printAsciiLogo()
    printSignature()
}
	

	// Load Config & Env
	
	config.Load()

	// Connect DB
	database.InitDB()
	go database.StartCleaner()

	// App Uptime
	appinfo.StartTime = time.Now()

	// Cache
	appCache := cache.New()
	handlers.SetCache(appCache)

	if err := utils.InitFonts("fonts/Inter_28pt-SemiBold.ttf"); err != nil {
		// log.Printf("Warning: Font loading failed, using fallback. Error: %v", err)
		logger.LogWarn("Warning: Font loading failed, using fallback. Error: %v", err)
	}

	mux := http.NewServeMux()

	// LandingPage
	// REMOVED
	// if config.AppConfig.App.LandingPage {
	// 	mux.HandleFunc("GET /", handleIndex)
	// }

	// Public Avatar & Assets Routes
	mux.HandleFunc("GET /avatar/{seed}", handlers.ServeDirectAvatar)              // /avatar/octa
	mux.HandleFunc("GET /u/{key...}", handlers.ServeUserAvatar)                   // /u/admin
	mux.HandleFunc("GET /avatar/github/{username}", handlers.GithubAvatarHandler) // /avatar/github/octocat

	// Upload Routews
	mux.HandleFunc("POST /upload", handlers.UploadHandler)
	mux.HandleFunc("DELETE /upload/delete", handlers.DeleteAPIHandler)

	if config.AppConfig.Cache.Enabled {
		InitConsoleUI(mux)
	}

	finalHandler := middleware.RateLimitMiddleware(middleware.CorsMiddleware(middleware.LoggerMiddleware(mux)))

	// FOR BENCHMARK
	// finalHandler := middleware.CorsMiddleware(middleware.LoggerMiddleware(mux))

	port := config.AppConfig.Server.Port

	baseURL := config.AppConfig.GetBaseUrl()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      finalHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.LogServerStart(port, baseURL)
	log.Fatal(server.ListenAndServe())
}
