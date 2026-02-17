package handlers

import (
	"bytes"
	"encoding/json"
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/forest6511/gdl"
)

func QBittorrentHandler(w http.ResponseWriter, r *http.Request) {

	path := strings.TrimSuffix(r.URL.Path, "/")
	method := r.Method

	logger.Log.Infow("qBittorrent Request", "method", method, "path", path)

	switch path {
	case "/qbittorrent/api/v2/auth/login", "/api/v2/auth/login":
		HandleQBLogin(w, r)
		return

	case "/qbittorrent/api/v2/app/webapiVersion", "/api/v2/app/webapiVersion":
		HandleQBVersion(w, r)
		return

	case "/qbittorrent/api/v2/app/preferences", "/api/v2/app/preferences":
		HandleQBPreferences(w, r)
		return

	case "/qbittorrent/api/v2/torrents/categories", "/api/v2/torrents/categories":
		HandleQBTorrentCategories(w, r)
		return

	case "/qbittorrent/api/v2/torrents/info", "/api/v2/torrents/info":
		HandleQBTorrentsInfo(w, r)
		return

	case "/qbittorrent/api/v2/torrents/files", "/api/v2/torrents/files":
		HandleQBTorrentFiles(w, r)
		return

	case "/qbittorrent/api/v2/torrents/properties", "/api/v2/torrents/properties":
		HandleQBTorrentProperties(w, r)
		return

	case "/qbittorrent/api/v2/torrents/add", "/api/v2/torrents/add":
		HandleQBAddTorrent(w, r)
		return

	case "/qbittorrent/api/v2/torrents/delete", "/api/v2/torrents/delete":
		HandleQBDelete(w, r)
		return

	case "/qbittorrent/api/v2/sync/maindata", "/api/v2/sync/maindata":
		HandleQBSyncMainData(w, r)
		return

	default:
		logger.Log.Warnw("Unhandled qBittorrent endpoint", "path", path)
		http.NotFound(w, r)
	}
}

func HandleQBLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	logger.Log.Debugw("qB Login Attempt",
		"username", username,
	)

	if username != config.AppConfig.ApplicationAuthUsername || password != config.AppConfig.ApplicationAuthPassword {
		logger.Log.Warnw("qB Login Failed", "username", username)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Fails."))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "SID",
		Value:    GenerateSessionHash(),
		Path:     "/",
		HttpOnly: true,
	})

	logger.Log.Infow("qB Login Success", "username", username)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok."))
}

func ValidateQBSession(r *http.Request) bool {
	cookie, err := r.Cookie("SID")
	if err != nil {
		return false
	}
	return cookie.Value != ""
}

func HandleQBVersion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("5.0.0"))
}

func HandleQBTorrentsInfo(w http.ResponseWriter, r *http.Request) {
	if !ValidateQBSession(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	category := r.URL.Query().Get("category")
	downloads := databases.GetLocalDownloadsByFilter(databases.LocalDownloadsInstance{
		Category: category,
		Protocol: config.ProtocolTorrent,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetTorrentsInfoList(downloads))
}

func HandleQBDelete(w http.ResponseWriter, r *http.Request) {
	if !ValidateQBSession(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	hashes := r.FormValue("hashes")
	deleteFiles := r.FormValue("deleteFiles")

	logger.Log.Debugw("Delete torrent", "hashes", hashes, "deleteFiles", deleteFiles)

	localDownloads := databases.GetLocalDownloadsByReferences(hashes)
	for _, localDownload := range localDownloads {
		err := DeleteLocalDownload(localDownload.IDString())
		if err != nil {
			logger.Log.Debugw("Unable to delete Local Download", "id", localDownload.IDString())
		}
	}
	w.WriteHeader(http.StatusOK)
}

func HandleQBSyncMainData(w http.ResponseWriter, r *http.Request) {
	if !ValidateQBSession(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	downloads := databases.GetLocalDownloadsByFilter(databases.LocalDownloadsInstance{
		Protocol: config.ProtocolTorrent,
	})

	torrentInfo := GetTorrentsInfoList(downloads)
	var torrents map[string]models.QBTorrentInfo = make(map[string]models.QBTorrentInfo)

	for _, torrent := range torrentInfo {
		torrents[torrent.Hash] = torrent
	}

	mainData := models.QBSyncMainInfo{
		Rid:             time.Now().Unix(),
		FullUpdate:      true,
		Torrents:        torrents,
		TorrentsRemoved: []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(mainData)
}

func HandleQBAddTorrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !ValidateQBSession(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to parse form: %v"}`, err), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["torrents"]
	urlsField := r.FormValue("urls")

	if len(files) == 0 && strings.TrimSpace(urlsField) == "" {
		http.Error(w, `{"error":"No torrents or urls provided"}`, http.StatusBadRequest)
		return
	}

	category := r.FormValue("category")
	if category == "" {
		category = "default"
	}

	var added []map[string]string
	var errors []string

	if len(files) > 0 {
		for _, fh := range files {
			file, err := fh.Open()
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: Failed to open file", fh.Filename))
				continue
			}

			fileBytes, err := io.ReadAll(file)
			defer file.Close()
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: Failed to read file", fh.Filename))
				continue
			}

			fileName, infoHash, infoError := GetTorrentInfo(fileBytes, "")
			if infoError != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", "File Info Parse Error", infoError))
				continue
			}

			logger.Log.Infow("Received Torrent File", "fileName", fileName, "infoHash", infoHash)
			ref, err := adapters.CreateDownload(config.ProtocolTorrent, fh.Filename, fileBytes, "", infoHash, category)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fh.Filename, err))
				continue
			}

			added = append(added, map[string]string{
				"name": fh.Filename,
				"hash": ref,
			})
		}
	}

	if urlsField != "" {
		urls := strings.Split(urlsField, "\n")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			if strings.HasPrefix(url, "magnet:") {

				fileName, infoHash, infoError := GetTorrentInfo([]byte{}, url)
				if infoError != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", url, infoError))
					continue
				}

				ref, errCreate := adapters.CreateDownload(config.ProtocolTorrent, fileName, []byte{}, url, infoHash, category)
				if errCreate != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", url, errCreate))
					continue
				}

				added = append(added, map[string]string{
					"name": fileName,
					"hash": ref,
				})

			} else if strings.HasSuffix(strings.ToLower(url), ".torrent") {
				fileBytes, fileStats, err := gdl.DownloadToMemory(r.Context(), url)

				fileName, infoHash, infoError := GetTorrentInfo(fileBytes, "")
				if infoError != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", url, infoError))
					continue
				}

				ref, err := adapters.CreateDownload(config.ProtocolTorrent, fileName, fileBytes, "", infoHash, category)
				if err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", url, err))
					continue
				}
				added = append(added, map[string]string{
					"name": fileStats.Filename,
					"hash": ref,
				})
			} else {
				errors = append(errors, fmt.Sprintf("%s: invalid URL or magnet", url))
				continue
			}
		}
	}

	resp := map[string]interface{}{
		"success": len(errors) == 0,
		"added":   added,
		"errors":  errors,
	}

	json.NewEncoder(w).Encode(resp)
}

func GetTorrentInfo(torrentBytes []byte, magnetLink string) (name string, infoHash string, err error) {
	if len(torrentBytes) > 0 {
		mi, err := metainfo.Load(bytes.NewReader(torrentBytes))
		if err != nil {
			return "", "", fmt.Errorf("failed to parse torrent file: %w", err)
		}

		info, err := mi.UnmarshalInfo()
		if err != nil {
			return "", "", fmt.Errorf("failed to unmarshal torrent info: %w", err)
		}

		infoHash = mi.HashInfoBytes().HexString()
		name = info.Name
		return name, infoHash, nil
	}

	if magnetLink != "" {
		u, err := url.Parse(magnetLink)
		if err != nil {
			return "", "", fmt.Errorf("invalid magnet link: %w", err)
		}

		xts := u.Query()["xt"]
		for _, xt := range xts {
			if strings.HasPrefix(xt, "urn:btih:") {
				infoHash = strings.TrimPrefix(xt, "urn:btih:")
				break
			}
		}

		if infoHash == "" {
			return "", "", fmt.Errorf("magnet link missing info hash")
		}

		name = u.Query().Get("dn")
		if name == "" {
			name = infoHash
		}

		return name, infoHash, nil
	}

	return "", "", fmt.Errorf("no torrent file or magnet link provided")
}

func HandleQBPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	prefs := map[string]interface{}{
		"locale":                                 "en",
		"create_subfolder_enabled":               true,
		"start_paused_enabled":                   false,
		"auto_delete_mode":                       0,
		"preallocate_all":                        false,
		"incomplete_files_ext":                   false,
		"auto_tmm_enabled":                       false,
		"torrent_changed_tmm_enabled":            false,
		"save_path_changed_tmm_enabled":          false,
		"category_changed_tmm_enabled":           false,
		"save_path":                              config.ApplicationDownloadRoot,
		"temp_path_enabled":                      false,
		"temp_path":                              "",
		"scan_dirs":                              map[string]interface{}{},
		"export_dir":                             "",
		"export_dir_fin":                         "",
		"mail_notification_enabled":              false,
		"upnp":                                   true,
		"dht":                                    true,
		"pex":                                    true,
		"lsd":                                    true,
		"encryption":                             0,
		"anonymous_mode":                         false,
		"proxy_type":                             -1,
		"proxy_ip":                               "",
		"proxy_port":                             0,
		"proxy_peer_connections":                 false,
		"proxy_auth_enabled":                     false,
		"proxy_username":                         "",
		"proxy_password":                         "",
		"max_connec":                             -1,
		"max_connec_per_torrent":                 -1,
		"max_uploads":                            -1,
		"max_uploads_per_torrent":                -1,
		"download_limit":                         0,
		"upload_limit":                           0,
		"max_active_downloads":                   -1,
		"max_active_torrents":                    -1,
		"max_active_uploads":                     -1,
		"web_ui_domain_list":                     "",
		"web_ui_address":                         "0.0.0.0",
		"web_ui_port":                            9090,
		"web_ui_upnp":                            false,
		"web_ui_username":                        "",
		"web_ui_password":                        "",
		"web_ui_csrf_protection_enabled":         true,
		"web_ui_clickjacking_protection_enabled": true,
		"web_ui_secure_cookie_enabled":           false,
		"web_ui_max_auth_fail_count":             5,
		"web_ui_ban_duration":                    3600,
		"web_ui_session_timeout":                 3600,
		"web_ui_host_header_validation_enabled":  true,
	}
	json.NewEncoder(w).Encode(prefs)
}

func HandleQBTorrentCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	categories := make(map[string]map[string]string)

	csv := strings.TrimSpace(config.AppConfig.ApplicationCategories)
	if csv != "" {
		for _, cat := range strings.Split(csv, ",") {
			categoryName := strings.TrimSpace(cat)
			if categoryName == "" {
				continue
			}

			categories[categoryName] = map[string]string{
				"name":     categoryName,
				"savePath": filepath.Join(config.ApplicationDownloadRoot, categoryName),
			}
		}
	}

	json.NewEncoder(w).Encode(categories)
}

func HandleQBTorrentFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	hash := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("hash")))
	if hash == "" {
		http.Error(w, `{"error":"Missing hash parameter"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	download := databases.GetLocalDownloadByFilter(databases.LocalDownloadsInstance{
		Protocol:                  config.ProtocolTorrent,
		OriginalDownloadReference: hash,
	})

	if download == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	var files []map[string]interface{}
	if len(download.OriginalDownloadFile) > 0 {
		miFile, errFile := metainfo.Load(bytes.NewReader(download.OriginalDownloadFile))
		if errFile != nil {
			logger.Log.Errorw("Failed to load torrent metainfo", "hash", hash, "error", errFile)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}

		info, err := miFile.UnmarshalInfo()
		if err != nil {
			logger.Log.Errorw("Failed to unmarshal torrent info", "hash", hash, "error", err)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		for _, file := range info.Files {
			fileName := strings.Join(file.Path, "/")
			files = append(files, map[string]interface{}{
				"name": fileName,
			})
		}
		if files == nil {
			files = make([]map[string]interface{}, 0)
		}
	} else {
		files = make([]map[string]interface{}, 0)
	}

	json.NewEncoder(w).Encode(files)
}

func HandleQBTorrentProperties(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	hash := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("hash")))
	if hash == "" {
		http.Error(w, `{"error":"Missing hash parameter"}`, http.StatusBadRequest)
		return
	}

	download := databases.GetLocalDownloadByFilter(databases.LocalDownloadsInstance{
		Protocol: config.ProtocolTorrent,
	})
	if download == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	var completedTime *int64
	if download.Status == config.DOWNLOAD_STATUS_CLIENT_COMPLETED {
		ts := download.CompletedAt.Unix()
		completedTime = &ts
	} else {
		completedTime = nil
	}

	response := map[string]interface{}{
		"save_path":                filepath.Join(config.ApplicationDownloadRoot, download.Category),
		"creation_date":            download.AddedAt.Unix(),
		"piece_size":               0,
		"comment":                  "",
		"total_wasted":             0,
		"total_uploaded":           0,
		"total_uploaded_session":   0,
		"total_downloaded":         0,
		"total_downloaded_session": 0,
		"up_limit":                 0,
		"dl_limit":                 0,
		"time_elapsed":             0,
		"seeding_time":             0,
		"nb_connections":           0,
		"nb_connections_limit":     -1,
		"share_ratio":              2,
		"addition_date":            download.AddedAt.Unix(),
		"completion_date":          completedTime,
		"created_by":               "",
		"private":                  false,
	}

	if len(download.OriginalDownloadFile) > 0 {
		mi, err := metainfo.Load(bytes.NewReader(download.OriginalDownloadFile))
		if err == nil {
			info, err := mi.UnmarshalInfo()
			if err == nil {
				response["piece_size"] = info.PieceLength
				response["creation_date"] = mi.CreationDate
				response["comment"] = mi.Comment
				response["created_by"] = mi.CreatedBy
				response["private"] = info.Private
			}
		}
	}

	json.NewEncoder(w).Encode(response)
}

func GetTopLevelDir(path string) string {
	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		path = string(os.PathSeparator) + path
	}

	trimmed := strings.TrimPrefix(path, string(os.PathSeparator))
	parts := strings.Split(trimmed, string(os.PathSeparator))

	if len(parts) > 0 {
		return string(os.PathSeparator) + parts[0]
	}

	return string(os.PathSeparator)
}

func GetTorrentsInfoList(downloads []databases.LocalDownloadsInstance) []models.QBTorrentInfo {
	liveDownloads := services.GetGDLService().Status()

	var result []models.QBTorrentInfo
	for _, download := range downloads {

		var savePath = filepath.Join(config.ApplicationDownloadRoot, download.Category)
		var contentPath = savePath
		var totalSize int64 = 0
		var completedSize int64 = 0

		var completedTime *int64
		if download.Status == config.DOWNLOAD_STATUS_CLIENT_COMPLETED {
			ts := download.CompletedAt.Unix()
			completedTime = &ts
		} else {
			completedTime = nil
		}

		switch download.Status {
		case config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING,
			config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING,
			config.DOWNLOAD_STATUS_PROVIDER_COMPLETED,
			config.DOWNLOAD_STATUS_DOWNLOADER_ADDED,
			config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING,
			config.DOWNLOAD_STATUS_DOWNLOADER_FAILED,
			config.DOWNLOAD_STATUS_DOWNLOADER_PAUSED,
			config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING,
			config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED:
			if download.DownloadType == config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE {
				contentPath = strings.TrimSuffix(filepath.Join(savePath, download.DownloadItems[0].FilePath), filepath.Ext(download.DownloadItems[0].FilePath))
			} else {
				contentPath = filepath.Join(savePath, models.GetTopFolderFromPath(download.DownloadItems[0].FilePath))
			}

			for _, item := range download.DownloadItems {
				totalSize += item.FileSize
				if strings.ToLower(item.Status) == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED {
					completedSize += item.FileSize
					continue
				}

				for _, status := range liveDownloads {
					if status.ID == item.IDString() {
						partialBytes := float64(item.FileSize) * (status.Percentage / 100)
						completedSize += int64(partialBytes)
						break
					}
				}
			}
		default:

		}

		progress := float64(0)
		if totalSize > 0 {
			progress = float64(completedSize) / float64(totalSize)
		}

		torrent := models.QBTorrentInfo{
			Hash:         download.OriginalDownloadReference,
			Name:         download.DownloadName,
			Size:         totalSize,
			Progress:     progress,
			State:        models.TranslateLocalDownloadStatusToQBStatus(download.Status),
			DlSpeed:      0,
			UpSpeed:      0,
			Eta:          0,
			Category:     download.Category,
			SavePath:     savePath,
			AddedOn:      download.AddedAt.Unix(),
			CompletionOn: completedTime,
			ContentPath:  contentPath + "/",
		}

		result = append(result, torrent)
	}

	if result == nil {
		result = make([]models.QBTorrentInfo, 0)
	}

	return result
}
