package main

import (
	"fetchtb/handlers"
	"net/http"
)

func main() {
	http.HandleFunc("/sabnzbd/api", handlers.SabHandler)
	http.HandleFunc("/qbittorrent/api", handlers.QBitHandler)
}
