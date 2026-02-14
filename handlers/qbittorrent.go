package handlers

import (
	"fetch/logger"
	"net/http"
	"strings"
)

func QBittorrentHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	logger.Log.Infow("qBittorrent Request",
		"method", r.Method,
		"path", path,
	)

	if strings.Contains(path, "/auth/login") {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "FetchTB_Session", Path: "/"})
		w.Write([]byte("Ok."))
		return
	}
}
