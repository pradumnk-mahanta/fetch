package handlers

import (
	"context"
	"encoding/json"
	"fetch/adapters"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/forest6511/gdl"
)

func SABNzbdHandler(writer http.ResponseWriter, request *http.Request) {

	writer.Header().Set("Content-Type", "application/json")
	query := request.URL.Query()

	user := request.FormValue("ma_username")
	pass := request.FormValue("ma_password")

	logger.Log.Debugw("Received Credentials", "user", user, "pass", "Not Logged")

	if user != config.AppConfig.ApplicationAuthUsername || pass != config.AppConfig.ApplicationAuthPassword {
		writer.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(writer).Encode(GetSABNzbdError("Credentials Incorrect!"))
		return
	}

	mode := query.Get("mode")
	name := query.Get("name")
	nzo_id := query.Get("value")
	category := query.Get("category")

	if category == "" {
		category = query.Get("cat")
		if category == "" {
			category = "default"
		}
	}
	logger.Log.Infow("Received SABNzbd API Request", "mode", mode, "name", name, "nzo_id", nzo_id, "category", category)

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
		switch name {
		case "delete":
			err := DeleteLocalDownload(nzo_id)
			if err != nil {
				logger.Log.Debugw("Unable to delete Local Download")
			}
			HandleSabHistory(writer)
		default:
			HandleSabHistory(writer)
		}

	case "addurl":
		nzbURL := query.Get("name")
		if nzbURL == "" {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(GetSABNzbdError("Missing NZB URL"))
			return
		}

		fileBytes, fileStats, err := gdl.DownloadToMemory(request.Context(), nzbURL)

		fileName := query.Get("nzbname")
		if fileName == "" {
			if fileStats.Filename != "" {
				fileName = fileStats.Filename
			} else {
				fileInfo, errFile := gdl.GetFileInfo(context.Background(), nzbURL)
				if errFile != nil {
					logger.Log.Errorw("Unable to retrieve file metadata:", errFile.Error())
					writer.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(writer).Encode(GetSABNzbdError("Missing NZB Name"))
					return
				}
				fileName = fileInfo.Filename
			}
		}

		id, err := adapters.CreateDownload(config.ProtocolUsenet, fileName, fileBytes, "", "", category)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(GetSABNzbdError("Failed to create download"))
			logger.Log.Errorw("Failed to create download", "error", err)
			return
		}
		RespondAdd(writer, id)

	case "addfile":
		err := request.ParseMultipartForm(32 << 20)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(GetSABNzbdError("Failed to parse multipart form"))
			logger.Log.Errorw("Failed to parse multipart form", "error", err)
			return
		}

		formCat := request.FormValue("cat")
		if formCat != "" {
			category = formCat
		}

		var file multipart.File
		var header multipart.FileHeader

		fromNameKey, headerNameKey, errName := request.FormFile("name")
		if errName != nil {
			logger.Log.Warnw("No nzbfile found in request for Key name, Searching with nzbfile key", "error", err)
			fromNzbFileKey, headerNzbFileKey, errNzbFile := request.FormFile("nzbfile")
			if errNzbFile != nil {
				writer.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(writer).Encode(GetSABNzbdError("Failed to retrieve nzb file form request"))
				logger.Log.Warnw("No nzbfile found in request for Key nzbfile", "error", err)
				return
			}
			header = *headerNzbFileKey
			file = fromNzbFileKey
			defer fromNzbFileKey.Close()
		} else {
			header = *headerNameKey
			file = fromNameKey
			defer fromNameKey.Close()
		}

		logger.Log.Infow("Received nzbfile", "filename", header.Filename, "size", header.Size)

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(GetSABNzbdError("Failed to read download file"))
			logger.Log.Errorw("Failed to read download file", "error", err)
			return
		}

		id, err := adapters.CreateDownload(config.ProtocolUsenet, adapters.GetSanitizedPath(header.Filename), fileBytes, "", "", category)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(GetSABNzbdError("Failed to create download"))
			logger.Log.Errorw("Failed to create download", "error", err)
			return
		}
		RespondAdd(writer, id)

	case "get_config":
		HandleConfig(writer, request)

	case "version":
		writer.Write([]byte(`{"version": "5.0.0"}`))

	case "fullstatus", "status":
		HandleFullStatus(writer, request)

	default:
		HandleSabQueue(writer)
	}
}

func HandleNzbDownload() {

}

func HandleSabQueue(writer http.ResponseWriter) {
	var sabQueueResponse models.SabQueueResponse
	var downloads []databases.LocalDownloadsInstance
	localDownloads, localDownloadsError := databases.GetLocalPendingDownloads(config.ProtocolUsenet)
	if localDownloadsError != nil {
		logger.Log.Debugw("Unable to get Local Download Items. Defaulting to Empty Queue")

	}
	downloads = localDownloads

	downloaderDownloads := services.GetGDLService().Status()
	sabQueueResponse = models.BuildSabQueueOutput(downloads, downloaderDownloads)
	json.NewEncoder(writer).Encode(sabQueueResponse)
}

func HandleSabHistory(writer http.ResponseWriter) {
	var downloads []databases.LocalDownloadsInstance
	localDownloads, localDownloadsError := databases.GetLocalCompletedDownloads(config.ProtocolUsenet)
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

func GetSABNzbdError(message string) map[string]interface{} {
	return map[string]interface{}{
		"status": false,
		"error":  message,
	}
}

func HandleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]interface{}{
		"config": map[string]interface{}{
			"misc": map[string]interface{}{
				"host":          "0.0.0.0",
				"port":          "9090",
				"api_key":       "",
				"download_dir":  filepath.Join(config.ApplicationDownloadRoot),
				"complete_dir":  filepath.Join(config.ApplicationDownloadRoot),
				"max_art_tries": 3,
				"enable_https":  false,
				"refresh_rate":  1,
				"direct_unpack": true,
				"pre_check":     true,
				"flat_unpack":   0,
			},

			"categories": GetConfigCategories(),

			"servers": []map[string]interface{}{
				{
					"name":        "Server 1",
					"host":        "",
					"port":        563,
					"connections": 50,
					"ssl":         1,
					"enable":      1,
				},
			},
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func GetConfigCategories() []map[string]interface{} {
	appConfigCategoriesString := config.AppConfig.ApplicationCategories
	if appConfigCategoriesString == "" {
		appConfigCategoriesString = "*"
	}

	categories := strings.Split(appConfigCategoriesString, ",")

	var result []map[string]interface{}
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category == "" {
			continue
		}
		result = append(result, map[string]interface{}{
			"name":   category,
			"pp":     "3",
			"script": "Default",
			"dir":    filepath.Join(config.ApplicationDownloadRoot, category),
		})
	}
	return result
}

func HandleFullStatus(w http.ResponseWriter, r *http.Request) {
	var sabFullHistoryResponse models.SabFullStatusResponse
	var downloads []databases.LocalDownloadsInstance
	localDownloads, localDownloadsError := databases.GetLocalPendingDownloads(config.ProtocolUsenet)
	if localDownloadsError != nil {
		logger.Log.Debugw("Unable to get Local Download Items. Defaulting to Empty Queue")

	}
	downloads = localDownloads

	downloaderDownloads := services.GetGDLService().Status()
	sabFullHistoryResponse = models.BuildSabFullStatusOutput(downloads, downloaderDownloads)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sabFullHistoryResponse)
}
