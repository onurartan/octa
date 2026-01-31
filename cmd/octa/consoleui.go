package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"octa/internal/config"
	"octa/internal/handlers"
	"octa/pkg/logger"
	// "octa/pkg/utils"

	"octa"
)

func InitConsoleUI(serve *http.ServeMux) {

	staticContent, _ := fs.Sub(octa.WebAssets, "web/static")
	fileServer := http.FileServer(http.FS(staticContent))

	// SERVE Static files
	serve.HandleFunc("GET /console/static/", func(w http.ResponseWriter, r *http.Request) {

		if !handlers.IsAuthenticated(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/console/static/js/login") && !handlers.IsAuthenticated(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		http.StripPrefix("/console/static/", fileServer).ServeHTTP(w, r)
	})

	// AUTHENTICATION ROUTES
	serve.HandleFunc("GET /console/login", handleLoginPage)
	serve.HandleFunc("POST /console/api/login", handlers.LoginRateLimitMiddleware(handlers.LoginHandler))
	serve.HandleFunc("POST /console/api/logout", handlers.LogoutHandler)

	// ADMIN DASHBOARD
	serve.HandleFunc("GET /console", handlers.AuthMiddleware(handleDashboard))

	// ADMIN API ROUTES

	// GET stats
	serve.HandleFunc("GET /console/api/stats", handlers.AuthMiddleware(handlers.GetStats))

	// GET Assets
	serve.HandleFunc("GET /console/api/assets", handlers.AuthMiddleware(handlers.ListAssets))

	// GET backup sqlite database
	serve.HandleFunc("GET /console/api/backup", handlers.AuthMiddleware(handlers.BackupHandler))

	// DELETE assets
	serve.HandleFunc("DELETE /console/api/assets/{id}", handlers.AuthMiddleware(handlers.DeleteAssetHandler))

	// PUT update asset keys
	serve.HandleFunc("PUT /console/api/assets/{id}", handlers.AuthMiddleware(handlers.UpdateAssetKeys))
}

// landing page
// REMOVED
// func handleIndex(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Path != "/" {
// 		utils.WriteError(w, http.StatusNotFound, utils.ErrRequestNotFound, "Page not found.")
// 		return
// 	}

// 	tmpl, err := template.ParseFiles("web/index.html")
// 	if err != nil {
// 		http.Error(w, "Service Unavailable", http.StatusInternalServerError)
// 		return
// 	}

// 	baseURL := config.AppConfig.GetBaseUrl()
// 	data := PageData{BaseURL: baseURL}

// 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
// 	tmpl.Execute(w, data)
// }





func handleLoginPage(w http.ResponseWriter, r *http.Request) {

	// expectedToken := utils.GenerateSessionHash(
	// 	config.AppConfig.ConsoleUI.User.Username,
	// 	config.AppConfig.ConsoleUI.User.Password,
	// )

	//  c, err := r.Cookie("auth_token"); err == nil && c.Value == expectedToken

	if handlers.IsAuthenticated(r) {
		http.Redirect(w, r, "/console", http.StatusSeeOther)
		return
	}

	renderTemplate(w, "web/login.html")
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "web/dashboard.html")
}

func renderTemplate(w http.ResponseWriter, path string) {
	tmpl, err := template.ParseFS(octa.WebAssets, path)
	if err != nil {
		
		logger.LogError("Template Error: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}


	baseURL := config.AppConfig.GetBaseUrl()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, map[string]string{"BaseURL": baseURL})
}
