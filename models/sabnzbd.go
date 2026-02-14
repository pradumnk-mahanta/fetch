package models

import (
	"fetch/config"
	"path/filepath"
	"time"
)

type SabQueueResponse struct {
	Queue SabQueueData `json:"queue"`
}

type SabQueueData struct {
	Status   string    `json:"status"`
	Speed    string    `json:"speed"`
	Size     string    `json:"size"`
	SizeLeft string    `json:"sizeleft"`
	Version  string    `json:"version"`
	Slots    []SabSlot `json:"slots"`
}

type SabSlot struct {
	NzoID    string `json:"nzo_id"`
	Filename string `json:"filename"`
	Size     string `json:"size"`
	Status   string `json:"status"`
	Index    int    `json:"index"`
}

type SabAddResponse struct {
	Status bool     `json:"status"`
	NzoIDs []string `json:"nzo_ids"`
}

type Download struct {
	ID            string
	DownloadName  string
	Status        string
	Category      string
	DownloadItems []LocalDownloadInstanceItem `json:"download_items"`
}

type SabQueueItem struct {
	NzoID      string  `json:"nzo_id"`
	Filename   string  `json:"filename"`
	Status     string  `json:"status"`
	Category   string  `json:"cat"`
	Size       int64   `json:"size"`
	SizeLeft   int64   `json:"sizeleft"`
	MB         float64 `json:"mb"`
	MBLeft     float64 `json:"mbleft"`
	Percentage float64 `json:"percentage"`
	Speed      float64 `json:"speed"` // KB/s
}

func BuildSabQueueOutput(
	downloads []LocalDownloadInstance,
	downloaderItems []GDLDownload,
) []SabQueueItem {

	downloaderMap := make(map[string]GDLDownload)
	for _, d := range downloaderItems {
		downloaderMap[d.ID] = d
	}

	var result []SabQueueItem

	for _, download := range downloads {

		var totalSize int64
		var totalDownloaded int64
		var totalSpeed float64

		for _, item := range download.DownloadItems {

			totalSize += item.FileSize

			if runtime, exists := downloaderMap[item.ID]; exists {
				totalDownloaded += int64(runtime.BytesDownloaded)
				totalSpeed += runtime.AverageSpeed
				continue
			}

			if item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED {
				totalDownloaded += item.FileSize
			}
		}

		sizeLeft := totalSize - totalDownloaded

		var percentage float64
		if totalSize > 0 {
			percentage = (float64(totalDownloaded) / float64(totalSize)) * 100
		}

		result = append(result, SabQueueItem{
			NzoID:      download.ID,
			Filename:   download.DownloadName,
			Status:     TransaltedClientStatusforSABNzbd(download.Status),
			Category:   download.Category,
			Size:       totalSize,
			SizeLeft:   sizeLeft,
			MB:         float64(totalSize) / 1024 / 1024,
			MBLeft:     float64(sizeLeft) / 1024 / 1024,
			Percentage: percentage,
			Speed:      totalSpeed,
		})
	}

	if result != nil {
		return result
	}

	return make([]SabQueueItem, 0)
}

func TransaltedClientStatusforSABNzbd(status string) string {
	switch status {
	case config.DOWNLOAD_STATUS_CLIENT_ADDED:
		return "Queued"
	case config.DOWNLOAD_STATUS_CLIENT_DOWNLOADING:
		return "Downloading"
	case config.DOWNLOAD_STATUS_CLIENT_PROCESSING:
		return "Fetching"
	case config.DOWNLOAD_STATUS_CLIENT_FAILED:
		return "Failed"
	case config.DOWNLOAD_STATUS_CLIENT_COMPLETED:
		return "Completed"
	default:
		return "Queued"
	}
}

type SabHistoryResponse struct {
	History SabHistory `json:"history"`
}

type SabHistory struct {
	Slots []SabHistoryItem `json:"slots"`
}

type SabHistoryItem struct {
	NzoID       string `json:"nzo_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Category    string `json:"cat"`
	Size        int64  `json:"size"`
	Downloaded  int64  `json:"downloaded"`
	Storage     string `json:"storage"`
	Completed   string `json:"completed"`
	FailMessage string `json:"fail_message,omitempty"`
}

func BuildSabHistoryResponse(
	downloads []LocalDownloadInstance,
) SabHistoryResponse {

	var slots []SabHistoryItem

	for _, download := range downloads {

		status := TransaltedClientStatusforSABNzbd(download.Status)

		if status != "Completed" && status != "Failed" {
			continue
		}

		var totalSize int64
		var downloaded int64

		for _, item := range download.DownloadItems {
			totalSize += item.FileSize

			if item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED {
				downloaded += item.FileSize
			}
		}

		root := config.APPLICATION_DOWNLOAD_ROOT.GetValue()
		storagePath := filepath.Join(root, download.Category, download.DownloadName)

		slots = append(slots, SabHistoryItem{
			NzoID:      download.ID,
			Name:       download.DownloadName,
			Status:     status,
			Category:   download.Category,
			Size:       totalSize,
			Downloaded: downloaded,
			Storage:    storagePath,
			Completed:  download.AddedAt.Format(time.RFC3339),
		})
	}

	if slots == nil {
		slots = make([]SabHistoryItem, 0)
	}

	return SabHistoryResponse{
		History: SabHistory{
			Slots: slots,
		},
	}
}
