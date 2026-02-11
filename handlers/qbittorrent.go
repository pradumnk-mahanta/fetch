package handlers

import (
	"log/slog" // Updated import
	"net/http"
	"strings"
)

func QBittorrentHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Log the exact path requested
	slog.Info("qBittorrent Request",
		"method", r.Method,
		"path", path,
	)

	if strings.Contains(path, "/auth/login") {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "FetchTB_Session", Path: "/"})
		w.Write([]byte("Ok."))
		return
	}
}
