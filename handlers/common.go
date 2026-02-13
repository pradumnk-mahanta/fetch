package handlers

import (
	"fetch/adapters"
	"fetch/logger"
	"fetch/models"
	"net/http"
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
			result, err := adapters.ProcessDownloads()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to update downloads"}`))
				return
			}
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": "` + result + `"}`))
		case "list":
			var downloads models.LocalDownloadInstances
			downloads, err := adapters.LocalDownloadList()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
			}

			logger.Log.Debugw("List downloads result", "result", downloads)

			writer.WriteHeader(http.StatusOK)
			jsonResult, err := downloads.ToJSON()
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to convert result to JSON"}`))
				return
			}
			writer.Write([]byte(jsonResult))
		case "update_status":
			id := query.Get("id")
			status := query.Get("status")
			err := adapters.LocalDownloadUpdateStatus(id, status)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte(`{"error": "Failed to update download status in database"}`))
				return
			}

			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`{"result": "Downloader Download status updated to ` + status + ` with ID ` + id + `"}`))
		default:
			return
		}

	case "download_items":
		switch task {
		case "process":
			result, err := adapters.ProcessDownloads()
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
		// case "delete":
		// 	id := query.Get("id")
		// 	deleted, err := adapters.DownloaderDeleteDownload(id)
		// 	if err != nil {
		// 		writer.WriteHeader(http.StatusInternalServerError)
		// 		writer.Write([]byte(`{"error": "Failed to delete download from downloader"}`))
		// 		return
		// 	}

		// 	if !deleted {
		// 		writer.WriteHeader(http.StatusNotFound)
		// 		writer.Write([]byte(`{"error": "Download not found in downloader"}`))
		// 		return
		// 	}

		// 	writer.WriteHeader(http.StatusOK)
		// 	writer.Write([]byte(`{"result": "Downloader Download deleted with ID ` + id + `"}`))
		default:
			return
		}

	case "downloader_resume_download", "downloader_retry_download":
		id := query.Get("id")
		// deleted, err := adapters.DownloaderResumeDownload(id)
		// if err != nil {
		// 	writer.WriteHeader(http.StatusInternalServerError)
		// 	writer.Write([]byte(`{"error": "Failed to resume download in downloader"}`))
		// 	return
		// }

		// if !deleted {
		// 	writer.WriteHeader(http.StatusNotFound)
		// 	writer.Write([]byte(`{"error": "Download not found in downloader"}`))
		// 	return
		// }

		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(`{"result": "Downloader Download resumed with ID ` + id + `"}`))

	default:
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`{"error": "Invalid task"}`))
	}
}
