package main

import (
	"fetch/config"
	"fetch/databases"
	"fetch/handlers"
	"fetch/services"
	"log/slog"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
	}

	slog.Info("Initializing FetchTB System...")

	services.InitGDLService()

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
