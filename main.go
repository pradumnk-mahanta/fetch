package main

import (
	"context"
	"fetch/config"
	"fetch/databases"
	"fetch/handlers"
	"fetch/logger"
	"fetch/scheduler"
	"fetch/services"

	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	logger.InitLogger()
	defer logger.Sync()

	logger.Log.Infow("Initializing FetchTB System...")
	logger.Log.Debugw("Initializing FetchTB System...")

	logger.Log.Infow("Initializing Downloader!")
	services.InitGDLService()
	logger.Log.Infow("Downloader Initialized!")

	logger.Log.Infow("Initializing Database!")
	if err := databases.InitDB(); err != nil {
		logger.Log.Errorw("Database initialization failed", "error", err)
	}
	logger.Log.Infow("Database initialized successfully", "path", "./data/fetchtb.db")

	logger.Log.Infow("Initializing Scheduler!")
	scheduler := &scheduler.Scheduler{}
	scheduler.Start(context.Background())
	logger.Log.Infow("Scheduler Initialized!")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/fetch/api", handlers.CommonHandler)

	port := ":" + config.APPLICATION_API_PORT.GetValue()
	logger.Log.Infow("Server Starting on", "port", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Log.Errorw("Server crashed", "error", err)
		scheduler.Stop()
	}
}
