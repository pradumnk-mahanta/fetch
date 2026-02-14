package models

import (
	"fetch/config"
	"fetch/databases"
	"fmt"
	"path/filepath"
	"time"
)

type SabQueueResponse struct {
	Queue SabQueue `json:"queue"`
}

type SabQueue struct {
	Status    string         `json:"status"`
	Speed     string         `json:"speed"`
	Size      string         `json:"size"`
	SizeLeft  string         `json:"sizeleft"`
	NoOfSlots int            `json:"noofslots"`
	Slots     []SabQueueItem `json:"slots"`
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
	Percentage int64   `json:"percentage"`
	Speed      float64 `json:"speed"`
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

func BuildSabQueueOutput(
	downloads []databases.LocalDownloadsInstance,
	downloaderItems []GDLDownload,
) SabQueueResponse {

	downloaderMap := make(map[string]GDLDownload, len(downloaderItems))
	for _, d := range downloaderItems {
		downloaderMap[d.ID] = d
	}

	slots := make([]SabQueueItem, 0, len(downloads))

	var globalTotalSize int64
	var globalTotalDownloaded int64
	var globalSpeed float64

	for _, download := range downloads {

		var totalSize int64
		var totalDownloaded int64
		var totalSpeed float64

		for _, item := range download.DownloadItems {

			totalSize += item.FileSize

			if runtime, exists := downloaderMap[item.IDString()]; exists {
				totalDownloaded += int64(runtime.BytesDownloaded)
				totalSpeed += runtime.AverageSpeed
				continue
			}

			if item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED {
				totalDownloaded += item.FileSize
			}
		}

		if totalDownloaded > totalSize {
			totalDownloaded = totalSize
		}

		sizeLeft := totalSize - totalDownloaded
		if sizeLeft < 0 {
			sizeLeft = 0
		}

		var percentage int64
		if totalSize > 0 {
			percentage = int64((float64(totalDownloaded) / float64(totalSize)) * 100)
		}

		slots = append(slots, SabQueueItem{
			NzoID:      download.IDString(),
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

		globalTotalSize += totalSize
		globalTotalDownloaded += totalDownloaded
		globalSpeed += totalSpeed
	}

	globalSizeLeft := globalTotalSize - globalTotalDownloaded
	if globalSizeLeft < 0 {
		globalSizeLeft = 0
	}

	queueStatus := "Idle"
	if globalSpeed > 0 {
		queueStatus = "Downloading"
	}

	return SabQueueResponse{
		Queue: SabQueue{
			Status:    queueStatus,
			Speed:     formatSpeed(globalSpeed),
			Size:      formatBytes(globalTotalSize),
			SizeLeft:  formatBytes(globalSizeLeft),
			NoOfSlots: len(slots),
			Slots:     slots,
		},
	}
}

func formatBytes(b int64) string {
	gb := float64(b) / 1024 / 1024 / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

func formatSpeed(speed float64) string {
	mb := speed / 1024
	return fmt.Sprintf("%.2f MB/s", mb)
}

func TransaltedClientStatusforSABNzbd(status string) string {
	switch status {
	case config.DOWNLOAD_STATUS_CLIENT_ADDED:
		return "Queued"
	case config.DOWNLOAD_STATUS_CLIENT_PROCESSING:
		return "Fetching"
	case config.DOWNLOAD_STATUS_CLIENT_FAILED:
		return "Failed"
	case config.DOWNLOAD_STATUS_CLIENT_COMPLETED:
		return "Completed"
	default:
		return "Downloading"
	}
}

type SabHistoryResponse struct {
	History SabHistory `json:"history"`
}

type SabHistory struct {
	NoOfSlots         int              `json:"noofslots"`
	PPSlots           int              `json:"ppslots"`
	DaySize           string           `json:"day_size"`
	WeekSize          string           `json:"week_size"`
	MonthSize         string           `json:"month_size"`
	TotalSize         string           `json:"total_size"`
	LastHistoryUpdate int64            `json:"last_history_update"`
	Slots             []SabHistoryItem `json:"slots"`
}

type SabHistoryItem struct {
	ActionLine   string      `json:"action_line"`
	DuplicateKey string      `json:"duplicate_key"`
	Meta         interface{} `json:"meta"`
	FailMessage  string      `json:"fail_message"`
	Loaded       bool        `json:"loaded"`
	Size         string      `json:"size"`
	Category     string      `json:"category"`
	PP           string      `json:"pp"`
	Retry        int         `json:"retry"`
	Script       string      `json:"script"`
	NzbName      string      `json:"nzb_name"`
	DownloadTime int         `json:"download_time"`
	Storage      string      `json:"storage"`
	HasRating    bool        `json:"has_rating"`
	Status       string      `json:"status"`
	ScriptLine   string      `json:"script_line"`
	Completed    int64       `json:"completed"`
	TimeAdded    int64       `json:"time_added"`
	NzoID        string      `json:"nzo_id"`
	Downloaded   int64       `json:"downloaded"`
	Report       string      `json:"report"`
	Password     string      `json:"password"`
	Path         string      `json:"path"`
	PostProcTime int         `json:"postproc_time"`
	Name         string      `json:"name"`
	URL          string      `json:"url"`
	MD5Sum       string      `json:"md5sum"`
	Archive      bool        `json:"archive"`
	Bytes        int64       `json:"bytes"`
	URLInfo      string      `json:"url_info"`
	StageLog     []StageLog  `json:"stage_log"`
}

type StageLog struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

func BuildSabHistoryResponse(
	downloads []databases.LocalDownloadsInstance,
) SabHistoryResponse {

	slots := make([]SabHistoryItem, 0)
	var totalBytes int64
	now := time.Now().Unix()

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

		root := config.APPLICATION_DOWNLOAD_ROOT
		storagePath := filepath.Join(root, download.Category, download.DownloadName)

		completedUnix := download.CompletedAt.Unix()
		addedUnix := download.AddedAt.Unix()

		totalBytes += totalSize

		slots = append(slots, SabHistoryItem{
			ActionLine:   "",
			DuplicateKey: "",
			Meta:         nil,
			FailMessage:  "",
			Loaded:       false,
			Size:         formatBytesHistory(totalSize),
			Category:     download.Category,
			PP:           "D",
			Retry:        0,
			Script:       "None",
			NzbName:      download.DownloadName + ".nzb",
			DownloadTime: int(download.CompletedAt.Sub(download.AddedAt).Seconds()),
			Storage:      storagePath,
			HasRating:    false,
			Status:       status,
			ScriptLine:   "",
			Completed:    completedUnix,
			TimeAdded:    addedUnix,
			NzoID:        download.IDString(),
			Downloaded:   downloaded,
			Report:       "",
			Password:     "",
			Path:         storagePath,
			PostProcTime: 0,
			Name:         download.DownloadName,
			URL:          download.DownloadName + ".nzb",
			MD5Sum:       "",
			Archive:      false,
			Bytes:        totalSize,
			URLInfo:      "",
			StageLog:     []StageLog{},
		})
	}

	return SabHistoryResponse{
		History: SabHistory{
			NoOfSlots:         len(slots),
			PPSlots:           0,
			DaySize:           formatBytesHistory(totalBytes),
			WeekSize:          formatBytesHistory(totalBytes),
			MonthSize:         formatBytesHistory(totalBytes),
			TotalSize:         formatBytesHistory(totalBytes),
			LastHistoryUpdate: now,
			Slots:             slots,
		},
	}
}

func formatBytesHistory(b int64) string {
	gb := float64(b) / 1024 / 1024 / 1024
	return fmt.Sprintf("%.1f G", gb)
}
