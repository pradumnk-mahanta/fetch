package handlers

import (
	"encoding/json"
	"fetch/adapters"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"net/http"
)

const (
	protocol = "usenet"
)

func SABNzbdHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	query := request.URL.Query()
	mode := query.Get("mode")
	name := query.Get("name")
	//nzo_id := query.Get("value") // nzo_id for operations
	category := query.Get("cat")

	if category == "" {
		category = "default"
	}

	switch mode {
	case "queue":
		switch name {
		case "delete":
			//database.DeleteDownload(nzo_id)
			handleSabQueue(writer)
		case "pause":
			//database.UpdateStatus(nzo_id, "Paused")
			handleSabQueue(writer)
		case "resume":
			//database.UpdateStatus(nzo_id, "Downloading")
			handleSabQueue(writer)
		default:
			handleSabQueue(writer)
		}

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
		respondAdd(writer, id) //Handle return gracefully

	case "version":
		writer.Write([]byte(`{"version": "4.2.0"}`))

	default:
		handleSabQueue(writer)
	}
}

func handleNzbDownload() {

}

func handleSabQueue(writer http.ResponseWriter) {
	resp := models.SabQueueResponse{
		Queue: models.SabQueueData{
			Status:   "Downloading",
			Speed:    "25 MB/s",
			Size:     "100 GB",
			SizeLeft: "50 GB",
			Version:  "4.2.0",
			Slots:    databases.GetSABNzbdQueue(),
		},
	}
	json.NewEncoder(writer).Encode(resp)
}

func respondAdd(w http.ResponseWriter, id string) {
	json.NewEncoder(w).Encode(models.SabAddResponse{
		Status: true,
		NzoIDs: []string{id},
	})
}
