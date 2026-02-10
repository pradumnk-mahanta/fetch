package handlers

import (
	"encoding/json"
	"fetchtb/database"
	"fetchtb/models"
	"log/slog" // Updated import
	"net/http"
)

func SabHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	mode := query.Get("mode")

	// Structured Request Log
	slog.Info("SABnzbd Request",
		"method", r.Method,
		"mode", mode,
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")

	switch mode {
	case "queue":
		handleSabQueue(w)
	case "addurl":
		name := query.Get("name")
		if name == "" {
			name = "Unknown_NZB"
		}

		newID := database.AddDownload("usenet", name)

		json.NewEncoder(w).Encode(models.SabAddResponse{
			Status: true,
			NzoIDs: []string{newID},
		})
	default:
		handleSabQueue(w)
	}
}

func handleSabQueue(w http.ResponseWriter) {
	resp := models.SabQueueResponse{
		Queue: models.SabQueueData{
			Status:   "Downloading",
			Speed:    "25 MB/s",
			Size:     "100 GB",
			SizeLeft: "50 GB",
			Version:  "4.2.0",
			Slots:    database.GetSabQueue(),
		},
	}
	json.NewEncoder(w).Encode(resp)
}
