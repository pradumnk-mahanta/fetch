package handlers

import (
	"fetchtb/adapters"
	"fetchtb/models"
	"fetchtb/utils"
	"net/http"
)

func TasksHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	query := request.URL.Query()
	task := query.Get("task")

	switch task {
	case "list_downloads":
		var downloads models.LocalDownloadInstances
		downloads, err := adapters.ListDownloads()
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
		}

		utils.Logger.Debugw("List downloads result", "result", downloads)

		writer.WriteHeader(http.StatusOK)
		jsonResult, err := downloads.ToJSON()
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte(`{"error": "Failed to convert result to JSON"}`))
			return
		}
		writer.Write([]byte(jsonResult))
	case "update_downloads":
		result, err := adapters.UpdateDownloads()
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte(`{"error": "Failed to update downloads"}`))
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte(`{"result": "` + result + `"}`))
	default:
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`{"error": "Invalid task"}`))
	}
}
