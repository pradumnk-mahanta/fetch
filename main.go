package main

import (
	"fetchtb/services"
	"fetchtb/utils"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	utils.InitLogger()
	defer utils.Sync()
	utils.Logger.Info("Initializing FetchTB System...")

	// Testing Parts

	JdClient, err := services.NewJDClient()
	if err != nil {
		utils.Logger.Fatalw("Failed to initialize JDownloader client", "error", err)
	}

	downloads, err := JdClient.CheckPackageStatus()

	utils.Logger.Infow("Current Downloads", "count", len(downloads))

	//Testing Parts

	//if err := database.InitDB(); err != nil {
	//	utils.Logger.Fatalw("Database initialization failed", "error", err)
	//}
	//utils.Logger.Info("Database loaded successfully", "path", "./data/fetchtb.db")

	//http.HandleFunc("/sabnzbd/api", handlers.SabHandler)
	//http.HandleFunc("/qbittorrent/api", handlers.QBitHandler)

	// port := ":9090"
	// utils.Logger.Infow("Server starting",
	// 	"port", port,
	// 	"sabnzbd_url", "http://localhost"+port+"/sabnzbd/api",
	// 	"qbit_url", "http://localhost"+port+"/api/v2/",
	// )

	//if err := http.ListenAndServe(port, nil); err != nil {
	//	utils.Logger.Fatalw("Server crashed", "error", err)
	//}
}
