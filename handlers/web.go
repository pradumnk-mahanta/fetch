package handlers

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fetch/config"
	"fetch/logger"
	"net/http"
	"time"
)

//go:embed web/*
var staticFiles embed.FS

var activeSessions = make(map[string]time.Time)

const sessionCookieName = "fetch_session"

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

	if user == config.AppConfig.ApplicationAuthUsername && pass == config.AppConfig.ApplicationAuthPassword {
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

	if config.AppConfig.ApplicationAuthUsername == "" {
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
		if r.URL.Path == "/internal/api/stats" {
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

	case "/internal/api/stats":
		HandleDownlaodsList(w)

	case "/settings":
		content, _ := staticFiles.ReadFile("web/settings.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
		return

	case "/add":
		content, _ := staticFiles.ReadFile("web/add.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
		return

	case "/internal/api/config":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.AppConfig)

	case "/internal/api/config/update":
		HandleConfigUpdate(w, r)

	case "/internal/api/delete":
		HandleDownloadDelete(w, r)

	case "/internal/api/retry":
		HandleDownloadRetry(w, r)

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
	if config.AppConfig.ApplicationAuthUsername != "" {
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

	config.AppConfig.ApplicationAuthUsername = u
	config.AppConfig.ApplicationAuthPassword = p

	if err := config.SaveConfig(); err != nil {
		logger.Log.Errorw("Failed to save credentials to config file", "error", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	logger.Log.Infow("Admin user registered and app secured", "user", u)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HandleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		logger.Log.Errorw("Failed to decode config update", "error", err)
		http.Error(w, "Invalid configuration format", http.StatusBadRequest)
		return
	}

	*config.AppConfig = newConfig

	logger.InitLogger()

	if err := config.SaveConfig(); err != nil {
		logger.Log.Errorw("Failed to save config to disk", "error", err)
		http.Error(w, "Internal server error saving file", http.StatusInternalServerError)
		return
	}

	logger.Log.Infow("Configuration updated and saved via Settings page")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

func HandleDownloadDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	err := DeleteLocalDownload(id)
	if err != nil {
		logger.Log.Errorw("Failed to delete download", "id", id, "error", err)
		http.Error(w, "Deletion failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "deleted"}`))
}

func HandleDownloadRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	err := RetryLocalDownload(id)
	if err != nil {
		logger.Log.Errorw("Failed to update download", "id", id, "error", err)
		http.Error(w, "Retry Failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}

func HandleDownloadAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//ADDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}
