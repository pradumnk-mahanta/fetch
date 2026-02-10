package handlers

import (
	"encoding/json"
	"fetchtb/database"
	"fetchtb/models"
	"log/slog"
	"net/http"
)

func SabHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	mode := query.Get("mode")
	name := query.Get("name")   // 'delete', 'pause', 'resume' or filename for addurl
	value := query.Get("value") // nzo_id for operations
	cat := query.Get("cat")     // Category from URL

	// Fallback: if cat is empty, check "cat" parameter again (sometimes sent differently)
	if cat == "" {
		cat = "default"
	}

	switch mode {
	case "queue":
		// Handle queue mutations inside the queue mode (SABnzbd quirk)
		if name == "delete" {
			database.DeleteDownload(value)
			handleSabQueue(w)
		} else if name == "pause" {
			database.UpdateStatus(value, "Paused")
			handleSabQueue(w)
		} else if name == "resume" {
			database.UpdateStatus(value, "Downloading")
			handleSabQueue(w)
		} else {
			handleSabQueue(w)
		}

	case "addurl":
		filename := name
		if filename == "" {
			filename = "Unknown_NZB"
		}

		newID := database.AddDownload("usenet", filename, cat)
		respondAdd(w, newID)

	case "addfile":
		// Handle Multipart Upload
		err := r.ParseMultipartForm(32 << 20) // 32MB max
		if err != nil {
			slog.Error("Failed to parse multipart form", "error", err)
			return
		}

		// Get category from form data
		formCat := r.FormValue("cat")
		if formCat != "" {
			cat = formCat
		}

		// Get the file
		file, header, err := r.FormFile("nzbfile")
		if err != nil {
			slog.Error("No nzbfile found in request")
			return
		}
		defer file.Close()

		newID := database.AddDownload("usenet", header.Filename, cat)
		respondAdd(w, newID)

	case "version":
		w.Write([]byte(`{"version": "4.2.0"}`))

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

func respondAdd(w http.ResponseWriter, id string) {
	json.NewEncoder(w).Encode(models.SabAddResponse{
		Status: true,
		NzoIDs: []string{id},
	})
}
