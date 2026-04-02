package handlers

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/forest6511/gdl"
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

	case "/internal/api/archives":
		HandleArchiveList(w)

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

	case "/archive":
		content, _ := staticFiles.ReadFile("web/archive.html")
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

	case "/internal/api/add":
		HandleDownloadAdd(w, r)

	case "/internal/api/archive":
		HandleArchiveDownloadAction(w, r)

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

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to parse form: %v"}`, err), http.StatusBadRequest)
		return
	}

	protocol := strings.ToLower(r.FormValue("protocol"))
	downloadType := r.FormValue("downloadType")
	category := r.FormValue("category")
	inputType := r.FormValue("inputType")
	provider := r.FormValue("provider")

	if protocol == "" || downloadType == "" || category == "" || inputType == "" || provider == "" {
		http.Error(w, `{"error":"Missing Required Fields"}`, http.StatusBadRequest)
		return
	}

	var localDownload databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
		Protocol:      protocol,
		Provider:      provider,
		Category:      category,
		DownloadType:  GetTranslatedDownloadType(protocol, provider, downloadType),
		Status:        config.DOWNLOAD_STATUS_CLIENT_ADDED,
		AddedAt:       time.Now(),
		DownloadItems: []databases.LocalDownloadsInstanceItem{},
	}

	switch inputType {
	case "file":
		fileData, fileHeader, errFile := r.FormFile("fileUpload")
		if errFile != nil {
			http.Error(w, `{"error":"Unable to Parse File Data"}`, http.StatusBadRequest)
			logger.Log.Warnw("Unable to Parse File Data", "error", errFile)
			return
		}

		fileBytes, errFileBytes := io.ReadAll(fileData)
		if errFileBytes != nil {
			http.Error(w, `{"error":"Unable to Parse File Data"}`, http.StatusBadRequest)
			logger.Log.Warnw("Unable to Parse File Data", "error", errFile)
			return
		}

		localDownload.DownloadName = GetCleanName(strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename)))
		localDownload.OriginalDownloadFile = fileBytes
		localDownload.Add()

	case "url":
		urlInput := r.FormValue("url")
		if urlInput == "" {
			logger.Log.Errorw("Missing urlInput Fields")
			http.Error(w, `{"error":"Missing urlInput Fields"}`, http.StatusBadRequest)
			return
		}

		if strings.HasPrefix(urlInput, "magnet:") {
			fileName, infoHash, infoError := GetTorrentInfo([]byte{}, urlInput)
			if infoError != nil {
				logger.Log.Errorw("Unable to retrieve Magnet metadata:", infoError.Error())
				http.Error(w, `{"error":"Unable to retrieve Magnet Metadata"}`, http.StatusBadRequest)
				return
			}

			localDownload.DownloadName = GetCleanName(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
			localDownload.OriginalDownloadUrl = urlInput
			localDownload.OriginalDownloadReference = infoHash

		} else {
			fileInfo, errFile := gdl.GetFileInfo(context.Background(), urlInput)
			if errFile != nil {
				logger.Log.Errorw("Unable to retrieve file metadata:", errFile.Error())
				http.Error(w, `{"error":"Unable to retrieve file Metadata"}`, http.StatusBadRequest)
				return
			}

			fileBytes, fileStats, errFileBytes := gdl.DownloadToMemory(r.Context(), urlInput)
			if errFileBytes != nil {
				logger.Log.Errorw("Unable to retrieve file metadata:", errFileBytes.Error())
				http.Error(w, `{"error":"Unable to retrieve file Metadata"}`, http.StatusBadRequest)
				return
			}

			var fileName string
			if fileStats.Filename != "" {
				fileName = fileStats.Filename
			} else if fileInfo.Filename != "" {
				fileName = fileInfo.Filename
			} else {
				logger.Log.Errorw("Unable to retrieve file Name from Url")
				http.Error(w, `{"error":"Unable to retrieve file Name from Url"}`, http.StatusBadRequest)
				return
			}

			localDownload.DownloadName = GetCleanName(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
			localDownload.OriginalDownloadFile = fileBytes

		}
		localDownload.Add()

	default:
		http.Error(w, "Invalid Input", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "Added"}`))
}

func GetTranslatedDownloadType(protocol string, provider string, downloadType string) string {
	var preferZipped bool
	switch protocol {
	case config.ProtocolTorrent:
		for _, item := range config.AppConfig.ConfiguredDebridProviders {
			if item.ID == provider {
				preferZipped = item.PreferZippedFolder
				continue
			}
		}
	case config.ProtocolUsenet:
		for _, item := range config.AppConfig.ConfiguredUsenetProviders {
			if item.ID == provider {
				preferZipped = item.PreferZippedFolder
				continue
			}
		}
	}

	switch downloadType {
	case config.DownloaderIdInternal:
		if preferZipped {
			return config.DOWNLOAD_TYPE_FULL_ARCHIVE
		} else {
			return config.DOWNLOAD_TYPE_INDIVIDUAL_FILE
		}
	case config.DownloaderIdStrmLink:
		return config.DOWNLOAD_TYPE_CREATE_STRM

	case config.DownloaderIdSymlink:
		return config.DOWNLOAD_TYPE_CREATE_SYMLINK

	default:
		return config.DOWNLOAD_TYPE_DO_NOT_DOWNLOAD
	}
}

func GetCleanName(name string) string {
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", ".")
	}
	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		logger.Log.Debug("Unable to decode the file name, using as is.")
		return name
	}
	return decodedName
}

func HandleArchiveDownloadAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	action := r.URL.Query().Get("action")
	if id == "" {
		http.Error(w, "Missing Action", http.StatusBadRequest)
		return
	}

	archiveDownload := databases.GetLocalArchivedDownloadByFilter(databases.LocalArchivedDownloadsInstance{
		ID: databases.GetParsedUint(id),
	})
	if archiveDownload == nil {
		logger.Log.Errorw("Failed to fetch archive download", "id", id)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch action {
	case "download":
		var localDownload databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:                  archiveDownload.Protocol,
			Provider:                  archiveDownload.Provider,
			DownloadName:              archiveDownload.DownloadName,
			DownloadType:              config.DOWNLOAD_TYPE_FULL_ARCHIVE,
			OriginalDownloadUrl:       archiveDownload.OriginalDownloadUrl,
			OriginalDownloadFile:      archiveDownload.OriginalDownloadFile,
			OriginalDownloadReference: archiveDownload.OriginalDownloadReference,
			Category:                  archiveDownload.Category,
			Status:                    config.DOWNLOAD_STATUS_CLIENT_ADDED,
			AddedAt:                   time.Now(),
			DownloadItems:             []databases.LocalDownloadsInstanceItem{},
		}
		localDownload.Add()
	case "delete":
		archiveDownload.Delete()
	case "refresh":
		archiveDownload.LastRefreshAt = time.Now().Add(time.Hour * 24 * 30 * -1)
		databases.UpdateLocalArchivedDownloadSelected(*archiveDownload)
	case "togglerefresh":
		refreshEnabled := r.URL.Query().Get("enabled") == "true"
		archiveDownload.Refresh = refreshEnabled
		databases.UpdateLocalArchivedDownloadSelected(*archiveDownload)
	case "readd":
		var localDownload databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:                  archiveDownload.Protocol,
			Provider:                  archiveDownload.Provider,
			DownloadName:              archiveDownload.DownloadName,
			DownloadType:              archiveDownload.DownloadType,
			OriginalDownloadUrl:       archiveDownload.OriginalDownloadUrl,
			OriginalDownloadFile:      archiveDownload.OriginalDownloadFile,
			OriginalDownloadReference: archiveDownload.OriginalDownloadReference,
			Category:                  archiveDownload.Category,
			Status:                    config.DOWNLOAD_STATUS_CLIENT_ADDED,
			AddedAt:                   time.Now(),
			DownloadItems:             []databases.LocalDownloadsInstanceItem{},
		}
		localDownload.Add()
		archiveDownload.Delete()
	default:
		http.Error(w, `{"error":"Invalid Action"}`, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "actioned"}`))
}
