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
)

func main() {
	config.LoadConfig()
	if err := config.LoadConfig(); err != nil {
		panic("Unable to find config file. Please add a config file and restart the application!")
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

	logger.Log.Infow("Initializing Scheduler!")
	scheduler := &scheduler.Scheduler{}
	scheduler.Start(context.Background())
	logger.Log.Infow("Scheduler Initialized!")

	http.HandleFunc("/sabnzbd/api", handlers.SABNzbdHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBittorrentHandler)
	http.HandleFunc("/fetch/api", handlers.CommonHandler)

	port := ":" + config.AppConfig.AppAPIPort
	logger.Log.Infow("Server Starting on", "port", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Log.Errorw("Server crashed", "error", err)
		scheduler.Stop()
	}
}
