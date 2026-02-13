package main

import (
	"fetch/config"
	"fetch/databases"
	"fetch/handlers"
	"fetch/logger"
	"fetch/services"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	logger.InitLogger()
	defer logger.Sync()

	err := godotenv.Load()
	if err != nil {
		logger.Log.Errorw("Error loading .env file", "error", err)
	}

	logger.Log.Infow("Initializing FetchTB System...")

	services.InitGDLService()

	if err := databases.InitDB(); err != nil {
		logger.Log.Errorw("Database initialization failed", "error", err)
	}
	logger.Log.Infow("Database loaded successfully", "path", "./data/fetchtb.db")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/fetch/api", handlers.CommonHandler)

	port := config.APPLICATION_API_PORT.GetValue()
	logger.Log.Infow("Server Starting",
		"port", port,
		"sabnzbd_url", "http://localhost"+port+"/sabnzbd/api",
		"qbit_url", "http://localhost"+port+"/qbittorrent/api",
	)

	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Log.Errorw("Server crashed", "error", err)
	}
}
