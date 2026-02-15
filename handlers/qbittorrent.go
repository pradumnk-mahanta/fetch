package handlers

import (
	"bytes"
	"encoding/json"
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/forest6511/gdl"
)

func QBittorrentHandler(w http.ResponseWriter, r *http.Request) {

	path := strings.TrimSuffix(r.URL.Path, "/")
	method := r.Method

	logger.Log.Infow("qBittorrent Request", "method", method, "path", path)

	switch path {
	case "/api/v2/auth/login":
		HandleQBLogin(w, r)
		return

	case "/api/v2/app/version":
		HandleQBVersion(w, r)
		return

	case "/api/v2/torrents/info":
		HandleQBTorrentsInfo(w, r)
		return

	case "/api/v2/torrents/add":
		HandleQBAddTorrent(w, r)
		return

	case "/api/v2/torrents/delete":
		HandleQBDelete(w, r)
		return

	case "/api/v2/sync/maindata":
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

	hashesParam := r.URL.Query().Get("hashes")

	response, err := models.BuildQBTorrentsInfo(hashesParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func HandleQBDelete(w http.ResponseWriter, r *http.Request) {
	if !ValidateQBSession(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	hashes := r.FormValue("hashes")
	deleteFiles := r.FormValue("deleteFiles")

	logger.Log.Debugw("Delete torrent", "hashes", hashes, "deleteFiles", deleteFiles)

	localDownloads := databases.GetLocalDownloadsByReference(hashes)
	for _, localDownload := range localDownloads {
		err := DeleteLocalDownload(localDownload.IDString())
		if err == nil {
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

	ridParam := r.URL.Query().Get("rid")

	data, err := models.BuildQBSyncMainData(ridParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
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

			ref, err := adapters.CreateDownload(config.ProtocolTorrent, fh.Filename, fileBytes, "", "", category)
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
