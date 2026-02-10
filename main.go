package main

import (
	"fetchtb/database"
	"fetchtb/handlers"
	"log/slog" // The new standard library
	"net/http"
	"os"
)

func main() {
	// Setup structured logger (Text format is best for CLI, JSON for cloud)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Initializing FetchTB System...")

	// 1. Initialize DB
	if err := database.InitDB(); err != nil {
		slog.Error("Database initialization failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database loaded successfully", "path", "./fetchtb.db")

	// 2. Register Routes
	http.HandleFunc("/sabnzbd/api", handlers.SabHandler)
	http.HandleFunc("/api", handlers.SabHandler)
	http.HandleFunc("/api/v2/", handlers.QBitHandler)

	// 3. Start Server
	port := ":8080"
	slog.Info("Server starting",
		"port", port,
		"sabnzbd_url", "http://localhost"+port+"/sabnzbd/api",
		"qbit_url", "http://localhost"+port+"/api/v2/",
	)

	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server crashed", "error", err)
		os.Exit(1)
	}
}
