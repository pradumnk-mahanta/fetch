package main

import (
	"fetchtb/adapters"
	"fetchtb/config"
	"fetchtb/handlers"
	"fetchtb/utils"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		utils.Logger.Errorw("Error loading .env file", "error", err)
	}

	utils.InitLogger()
	defer utils.Sync()
	utils.Logger.Infow("Initializing FetchTB System...")

	// Testing Parts

	// JdClient, err := services.NewJDClient()
	// if err != nil {
	// 	utils.Logger.Fatalw("Failed to initialize JDownloader client", "error", err)
	// }

	// downloads, err := JdClient.CheckPackageStatus()

	// utils.Logger.Infow("Current Downloads", "count", len(downloads))

	//Testing Parts

	if err := adapters.InitDB(); err != nil {
		utils.Logger.Fatalw("Database initialization failed", "error", err)
	}
	utils.Logger.Infow("Database loaded successfully", "path", "./data/fetchtb.db")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/tasks/api", handlers.TasksHandler)

	port := config.APPLICATION_API_PORT.GetValue()
	utils.Logger.Infow("Server Starting",
		"port", port,
		"sabnzbd_url", "http://localhost"+port+"/sabnzbd/api",
		"qbit_url", "http://localhost"+port+"/qbittorrent/api",
	)

	if err := http.ListenAndServe(port, nil); err != nil {
		utils.Logger.Fatalw("Server crashed", "error", err)
	}
}
