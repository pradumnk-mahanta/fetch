package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fetch/config"
	"fetch/databases"
	"fetch/handlers"
	"fetch/logger"
	"fetch/scheduler"
	"fetch/services"
	"time"

	"net/http"
)

//go:embed web/*
var staticFiles embed.FS

var activeSessions = make(map[string]time.Time)

const sessionCookieName = "fetch_session"

func main() {
	config.LoadConfig()
	if err := config.LoadConfig(); err != nil {
		panic("Unable to find config file. Please add a config file and restart the application!")
	}

	logger.InitLogger()
	defer logger.Sync()

	logger.Log.Infow("Initializing FetchTB System...")
	logger.Log.Debugw("Initializing FetchTB System...")

	logger.Log.Infow("Initializing Downloader!")
	services.InitGDLService()
	logger.Log.Infow("Downloader Initialized!")

	logger.Log.Infow("Initializing Database!")
	if err := databases.InitDB(); err != nil {
		logger.Log.Errorw("Database initialization failed", "error", err)
	}

	logger.Log.Infow("Initializing Scheduler!")
	scheduler := &scheduler.Scheduler{}
	scheduler.Start(context.Background())
	logger.Log.Infow("Scheduler Initialized!")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/fetch/api", handlers.CommonHandler)

	http.HandleFunc("/login", WebHandlerLogin)
	http.HandleFunc("/", WebProtectedHandler)
	http.HandleFunc("/api/internal/stats", WebProtectedHandler)
	http.HandleFunc("/register", WebHandlerRegister)

	port := ":" + config.AppConfig.AppAPIPort
	logger.Log.Infow("Server Starting on", "port", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Log.Errorw("Server crashed", "error", err)
		scheduler.Stop()
	}
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	expiry, exists := activeSessions[cookie.Value]
	if !exists || time.Now().After(expiry) {
		return false
	}
	return true
}

func WebHandlerLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := r.FormValue("username")
	pass := r.FormValue("password")

	if user == config.AppConfig.AppAuthUsername && pass == config.AppConfig.AppAuthPassword {
		hash := GenerateSessionHash()
		activeSessions[hash] = time.Now().Add(24 * time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    hash,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/?error=auth_failed", http.StatusSeeOther)
	}
}

func GenerateSessionHash() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func WebProtectedHandler(w http.ResponseWriter, r *http.Request) {

	if config.AppConfig.AppAuthUsername == "" {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		content, _ := staticFiles.ReadFile("web/register.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
		return
	}

	if !isAuthenticated(r) {
		if r.URL.Path == "/api/internal/stats" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		content, _ := staticFiles.ReadFile("web/login.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
		return
	}

	switch r.URL.Path {
	case "/":
		content, err := staticFiles.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)

	case "/api/internal/stats":
		handlers.HandleDownlaodsList(w)

	case "/logout":
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			delete(activeSessions, cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		logger.Log.Debugw("User logged out successfully")
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.NotFound(w, r)
	}
}

func WebHandlerRegister(w http.ResponseWriter, r *http.Request) {
	if config.AppConfig.AppAuthUsername != "" {
		http.Error(w, "Initial setup already completed.", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	u := r.FormValue("username")
	p := r.FormValue("password")

	if u == "" || p == "" {
		http.Redirect(w, r, "/?error=empty_fields", http.StatusSeeOther)
		return
	}

	config.AppConfig.AppAuthUsername = u
	config.AppConfig.AppAuthPassword = p

	if err := config.SaveConfig(); err != nil {
		logger.Log.Errorw("Failed to save credentials to config file", "error", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	logger.Log.Infow("Admin user registered and app secured", "user", u)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
