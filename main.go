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
	configLoadError := config.LoadConfig()
	if configLoadError != nil {
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

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/app/logo.png")
	})

	http.HandleFunc("/api", handlers.SABNzbdHandler)
	http.HandleFunc("/api/v2/", handlers.QBittorrentHandler)
	http.HandleFunc("/qbittorrent/api/v2/", handlers.QBittorrentHandler)
	http.HandleFunc("/sabnzbd/api", handlers.CommonHandler)
	http.HandleFunc("/fetch/api", handlers.CommonHandler)

	http.HandleFunc("/login", handlers.WebHandlerLogin)
	http.HandleFunc("/internal/api", handlers.WebProtectedHandler)
	http.HandleFunc("/settings", handlers.WebProtectedHandler)
	http.HandleFunc("/register", handlers.WebHandlerRegister)
	http.HandleFunc("/", handlers.WebProtectedHandler)

	port := ":9090"
	logger.Log.Infow("Server Starting on", "port", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		logger.Log.Errorw("Server crashed", "error", err)
		scheduler.Stop()
	}
}
