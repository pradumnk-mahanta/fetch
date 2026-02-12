package main

import (
	"fetchtb/config"
	"fetchtb/databases"
	"fetchtb/handlers"
	"fetchtb/utils"
	"log/slog"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
	}

	utils.InitLogger()
	defer utils.Sync()
	slog.Info("Initializing FetchTB System...")

	// Testing Parts

	// jdClient, err := services.NewJDClient()
	// if err != nil {
	// 	utils.Logger.Fatalw("Failed to initialize JDownloader client", "error", err)
	// }

	// jdAddLinkError := jdClient.AddLink("https://store-041.wnam.tb-cdn.io/dld/09e76728-c8ef-4217-b778-a032a3d94fd6?token=abed62c9-6a99-470a-b282-d75c47b2d3ef", "Test Package", "prowlarr")
	// if jdAddLinkError != nil {
	// 	utils.Logger.Errorw("Failed to add download to JDownloader", "error", jdAddLinkError)
	// }
	// utils.Logger.Infow("Current Downloads", "count", len(downloads))

	//Testing Parts

	if err := databases.InitDB(); err != nil {
		slog.Error("Database initialization failed", "error", err)
	}
	slog.Info("Database loaded successfully", "path", "./data/fetchtb.db")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/tasks/api", handlers.TasksHandler)

	port := config.APPLICATION_API_PORT.GetValue()
	slog.Info("Server Starting",
		"port", port,
		"sabnzbd_url", "http://localhost"+port+"/sabnzbd/api",
		"qbit_url", "http://localhost"+port+"/qbittorrent/api",
	)

	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server crashed", "error", err)
	}
}
