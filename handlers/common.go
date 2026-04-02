package handlers

import (
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"fmt"
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
		case "list":
			HandleDownlaodsList(writer)
		case "archive":
			HandleArchiveList(writer)
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
		logger.Log.Debugw("Unable to list all combined result")
		combinedDownloads = make([]models.CombinedDownloadDetails, 0)
	}

	logger.Log.Debugw("Successfully fetched combined downloads!")

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(models.CombinedDownloadsToJSONArray(combinedDownloads)))
}

func HandleArchiveList(writer http.ResponseWriter) {
	var archivedDownloads []databases.LocalArchivedDownloadsInstance

	archivedDownloads = databases.GetLocalArchivedDownloadsByFilter(databases.LocalArchivedDownloadsInstance{})
	if archivedDownloads == nil {
		logger.Log.Debugw("No archived downloads to process.")
		archivedDownloads = make([]databases.LocalArchivedDownloadsInstance, 0)
	}
	logger.Log.Debugw("Successfully fetched archived downloads!")

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(databases.LocalArchivedDownloadsInstancesToJSONArray(archivedDownloads)))
}

func DeleteLocalDownload(downloadId string) error {
	localDownload := databases.GetLocalDownloadByFilter(databases.LocalDownloadsInstance{
		ID: databases.GetParsedUint(downloadId),
	})
	if localDownload == nil {
		err := fmt.Errorf("local download not found: id=%s", downloadId)
		logger.Log.Warnw("Local download not found", "id", downloadId)
		return err
	}

	downloader := services.GetGDLService()
	for _, downloadItem := range localDownload.DownloadItems {
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

	localDownload.Delete()
	return nil
}

func RetryLocalDownload(downloadId string) error {
	localDownlaodItems := databases.GetLocalDownloadByFilter(databases.LocalDownloadsInstance{
		ID: databases.GetParsedUint(downloadId),
	})
	if localDownlaodItems == nil {
		err := fmt.Errorf("local download not found: id=%s", downloadId)
		logger.Log.Warnw("Local download not found", "id", downloadId)
		return err
	}

	downloader := services.GetGDLService()
	for _, downloadItem := range localDownlaodItems.DownloadItems {
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

	databases.UpdateLocalDownload(databases.LocalDownloadsInstance{
		ID:     databases.GetParsedUint(downloadId),
		Status: config.DOWNLOAD_STATUS_CLIENT_ADDED,
	})

	return nil
}
