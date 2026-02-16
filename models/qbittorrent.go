package models

import (
	"fetch/config"
	"path/filepath"
	"strings"
)

type QBTorrentInfo struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	Progress     float64 `json:"progress"`
	State        string  `json:"state"`
	DlSpeed      int64   `json:"dlspeed"`
	UpSpeed      int64   `json:"upspeed"`
	Eta          int64   `json:"eta"`
	Category     string  `json:"category"`
	SavePath     string  `json:"save_path"`
	ContentPath  string  `json:"content_path"`
	AddedOn      int64   `json:"added_on"`
	CompletionOn int64   `json:"completion_on"`
}

type QBSyncMainInfo struct {
	Rid             int64                    `json:"rid"`
	FullUpdate      bool                     `json:"full_update"`
	Torrents        map[string]QBTorrentInfo `json:"torrents"`
	TorrentsRemoved []string                 `json:"torrents_removed"`
}

func TranslateLocalDownloadStatusToQBStatus(status string) string {
	switch status {
	case config.DOWNLOAD_STATUS_CLIENT_ADDED:
		return "queuedDL"
	case config.DOWNLOAD_STATUS_CLIENT_PROCESSING:
		return "downloading"
	case config.DOWNLOAD_STATUS_CLIENT_FAILED:
		return "error"
	case config.DOWNLOAD_STATUS_CLIENT_COMPLETED:
		return "uploading"
	default:
		return "downloading"
	}
}

func GetTopFolderFromPath(path string) string {
	cleanPath := filepath.ToSlash(path)
	parts := strings.Split(strings.TrimLeft(cleanPath, "/"), "/")
	if len(parts) > 1 {
		return "/" + parts[0]
	}
	return "/"
}
