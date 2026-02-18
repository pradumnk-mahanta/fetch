package handlers

import (
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func CommonHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	query := request.URL.Query()
	domain := query.Get("domain")
	task := query.Get("task")

	switch domain {
	case "downloads":
		switch task {
		case "process":
			result, err := adapters.ProcessDownloadsQueue()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to update downloads"}`))
				return
			}
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": "` + result + `"}`))
		case "list":
			HandleDownlaodsList(writer)

		// case "update_status":
		// 	id := query.Get("id")
		// 	status := query.Get("status")
		// 	err := adapters.LocalDownloadUpdateStatus(id, status)
		// 	if err != nil {
		// 		writer.WriteHeader(http.StatusInternalServerError)
		// 		writer.Write([]byte(`{"error": "Failed to update download status in database"}`))
		// 		return
		// 	}
		// 	writer.WriteHeader(http.StatusOK)
		// 	writer.Write([]byte(`{"result": "Downloader Download status updated to ` + status + ` with ID ` + id + `"}`))
		default:
			return
		}

	case "download_items":
		switch task {
		case "process":
			result, err := adapters.ProcessDownloadItemsQueue()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to update downloads"}`))
				return
			}
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": "` + result + `"}`))
		default:
			return
		}

	case "downloader":
		switch task {
		case "list":
			var downloads models.GDLDownloads
			downloads, err := adapters.DownloaderListStatus()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to get downloader status"}`))
				return
			}

			jsonResult, err := downloads.ToJSON()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to convert downloader status to JSON"}`))
				return
			}

			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": ` + jsonResult + `}`))
		case "resume", "retry":
			id := query.Get("id")
			resumed, err := adapters.DownloaderResumeDownload(id)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to resume download in downloader"}`))
				return
			}
			if !resumed {
				writer.WriteHeader(http.StatusNotFound)
				writer.Write([]byte(`{"error": "Download Item not found in downloader"}`))
				return
			}
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": "Downloader Download resumed with ID ` + id + `"}`))
		default:
			return
		}

	default:
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`{"error": "Invalid task"}`))
	}
}

func HandleDownlaodsList(writer http.ResponseWriter) {
	var combinedDownloads []models.CombinedDownloadDetails
	gdlDownloads := services.GetGDLService().Status()

	combinedDownloads, combineError := models.GetCombinedDownloadDetails(gdlDownloads)
	if combineError != nil {
		logger.Log.Debugw("List downloads result", "result", combinedDownloads)
		combinedDownloads = make([]models.CombinedDownloadDetails, 0)
	}

	logger.Log.Debugw("List downloads result", "result", combinedDownloads)

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(models.CombinedDownloadsToJSONArray(combinedDownloads)))
}

func DeleteLocalDownload(downloadId string) error {
	localDownlaodItems, err := databases.GetLocalDownloadItemsForDownload(downloadId)
	if err != nil {
		return err
	}

	downloader := services.GetGDLService()
	for _, downloadItem := range localDownlaodItems {
		downloader.Delete(downloadItem.IDString())
		if downloadItem.FilePath != "" {
			pathOnDisk := config.ApplicationDownloadRoot + "/" + downloadItem.Category + "/" + downloadItem.FilePath
			if downloadItem.DownloadType == config.DOWNLOAD_TYPE_FULL_ARCHIVE {
				pathOnDisk = strings.TrimSuffix(pathOnDisk, filepath.Ext(downloadItem.FilePath))
			} else {
				pathOnDisk = filepath.Dir(pathOnDisk)
			}
			err := os.RemoveAll(pathOnDisk)
			if err != nil {
				if os.IsNotExist(err) {
					logger.Log.Debugw("Path not present on disk", "path", pathOnDisk)
				} else {
					logger.Log.Errorw("Failed to delete", "path", pathOnDisk, "error", err)
				}
			} else {
				logger.Log.Infow("Deleted", "path", pathOnDisk)
			}

		}
	}
	databases.DeleteLocalDownload(downloadId)
	return nil
}
