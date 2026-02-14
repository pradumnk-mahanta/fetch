package handlers

import (
	"encoding/json"
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"net/http"
)

const (
	protocol = "usenet"
)

func SABNzbdHandler(writer http.ResponseWriter, request *http.Request) {

	writer.Header().Set("Content-Type", "application/json")
	query := request.URL.Query()

	apikey := query.Get("apikey")
	logger.Log.Debugw("Received API key", "apikey", query.Get("apikey"))

	if apikey != config.SABNZBD_API_KEY.GetValue() {
		writer.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(writer).Encode(map[string]interface{}{
			"status": false,
			"error":  "API Key Incorrect.",
		})
		return
	}

	mode := query.Get("mode")
	name := query.Get("name")
	nzo_id := query.Get("value") // nzo_id for operations
	category := query.Get("cat")

	if category == "" {
		category = "default"
	}

	switch mode {
	case "queue":
		switch name {
		case "delete":
			err := DeleteLocalDownload(nzo_id)
			if err == nil {
				logger.Log.Debugw("Unable to delete Local Download")
			}
			HandleSabQueue(writer)
		case "pause":
			HandleSabQueue(writer)
		case "resume":
			HandleSabQueue(writer)
		default:
			HandleSabQueue(writer)
		}

	case "history":
		HandleSabHistory(writer)

	case "addfile":
		err := request.ParseMultipartForm(32 << 20)
		if err != nil {
			logger.Log.Errorw("Failed to parse multipart form", "error", err)
			return
		}

		formCat := request.FormValue("cat")
		if formCat != "" {
			category = formCat
		}

		file, header, err := request.FormFile("nzbfile")
		if err != nil {
			logger.Log.Errorw("No nzbfile found in request", "error", err)
			return
		}
		defer file.Close()

		logger.Log.Infow("Received nzbfile", "filename", header.Filename, "size", header.Size)
		id, err := adapters.CreateDownload(protocol, header.Filename, file, "", category)
		if err != nil {
			logger.Log.Errorw("Failed to create download", "error", err)
			return
		}
		RespondAdd(writer, id)

	case "version":
		writer.Write([]byte(`{"version": "4.2.0"}`))

	default:
		HandleSabQueue(writer)
	}
}

func HandleNzbDownload() {

}

func HandleSabQueue(writer http.ResponseWriter) {
	var sabQueueItems []models.SabQueueItem
	var downloads []models.LocalDownloadInstance
	localDownloads, localDownloadsError := databases.GetLocalPendingDownloads()
	if localDownloadsError != nil {
		logger.Log.Debugw("Unable to get Local Download Items. Defaulting to Empty Queue")

	}
	downloads = localDownloads

	downloaderDownloads := services.GetGDLService().Status()
	sabQueueItems = models.BuildSabQueueOutput(downloads, downloaderDownloads)
	json.NewEncoder(writer).Encode(sabQueueItems)
}

func HandleSabHistory(writer http.ResponseWriter) {
	var downloads []models.LocalDownloadInstance
	localDownloads, localDownloadsError := databases.GetLocalCompletedDownloads()
	if localDownloadsError != nil {
		logger.Log.Debugw("Unable to get Local Download Items. Defaulting to Empty Queue")
	}
	downloads = localDownloads

	sabHistoryItems := models.BuildSabHistoryResponse(downloads)
	json.NewEncoder(writer).Encode(sabHistoryItems)
}

func RespondAdd(w http.ResponseWriter, id string) {
	json.NewEncoder(w).Encode(models.SabAddResponse{
		Status: true,
		NzoIDs: []string{id},
	})
}

func DeleteLocalDownload(downloadId string) error {
	localDownlaodItems, err := databases.GetLocalDownloadItemsForDownload(downloadId)
	if err != nil {
		return err
	}

	downloader := services.GetGDLService()
	for _, downloadItem := range localDownlaodItems {
		downloader.Delete(downloadItem.ID)
	}

	databases.DeleteLocalDownload(downloadId)
	return nil
}
