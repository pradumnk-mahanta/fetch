package models

import (
	"fetch/config"
	"fetch/databases"
	"path/filepath"
	"strings"
	"time"
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
	// Remaining States - pausedDL, stalledDL // Map Later
}

func BuildQBTorrentsInfo(hashesParam string) ([]QBTorrentInfo, error) {

	downloads := databases.GetLocalDownloadsByReferences(hashesParam)
	var result []QBTorrentInfo
	for _, d := range downloads {

		var totalSize int64
		var completedSize int64

		for _, item := range d.DownloadItems {
			totalSize += item.FileSize
			if strings.ToLower(item.Status) == "completed" {
				completedSize += item.FileSize
			}
		}

		progress := float64(0)
		if totalSize > 0 {
			progress = float64(completedSize) / float64(totalSize)
		}

		torrent := QBTorrentInfo{
			Hash:         d.OriginalDownloadReference,
			Name:         d.DownloadName,
			Size:         totalSize,
			Progress:     progress,
			State:        TranslateLocalDownloadStatusToQBStatus(d.Status),
			DlSpeed:      0,
			UpSpeed:      0,
			Eta:          0,
			Category:     d.Category,
			SavePath:     config.ApplicationDownloadRoot + "/" + d.Category,
			AddedOn:      d.AddedAt.Unix(),
			CompletionOn: d.CompletedAt.Unix(),
		}

		result = append(result, torrent)
	}

	if result == nil {
		result = make([]QBTorrentInfo, 0)
	}

	return result, nil
}

func BuildQBSyncMainData(ridParam string) (*QBSyncMainInfo, error) {

	downloads := databases.GetLocalDownloadsByProtocol(config.ProtocolTorrent)
	torrents := make(map[string]QBTorrentInfo)

	for _, download := range downloads {

		savePath := config.ApplicationDownloadRoot + "/" + download.Category
		if len(download.DownloadItems) > 0 {
			if download.DownloadItems[0].DownloadType == config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE {
				savePath = strings.TrimSuffix(savePath+"/"+download.DownloadItems[0].FilePath, filepath.Ext(download.DownloadItems[0].FilePath))
			} else {
				savePath = filepath.Dir(savePath + "/" + download.DownloadItems[0].FilePath)
			}
		}

		var totalSize int64
		var completedSize int64
		for _, item := range download.DownloadItems {
			totalSize += item.FileSize
			if strings.ToLower(item.Status) == "completed" {
				completedSize += item.FileSize
			}
		}

		progress := float64(0)
		if totalSize > 0 {
			progress = float64(completedSize) / float64(totalSize)
		}

		hash := download.OriginalDownloadReference
		torrents[hash] = QBTorrentInfo{
			Hash:         hash,
			Name:         download.DownloadName,
			Size:         totalSize,
			Progress:     progress,
			State:        TranslateLocalDownloadStatusToQBStatus(download.Status),
			DlSpeed:      0,
			UpSpeed:      0,
			Eta:          0,
			Category:     download.Category,
			SavePath:     savePath,
			AddedOn:      download.AddedAt.Unix(),
			CompletionOn: download.CompletedAt.Unix(),
		}
	}

	return &QBSyncMainInfo{
		Rid:             time.Now().Unix(),
		FullUpdate:      true,
		Torrents:        torrents,
		TorrentsRemoved: []string{},
	}, nil
}
