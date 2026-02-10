package database

import (
	"database/sql"
	"fetchtb/models"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite" // CHANGED: Replaced mattn/go-sqlite3
)

var DB *sql.DB

func InitDB() error {
	var err error
	DB, err = sql.Open("sqlite", "./fetchtb.db")
	if err != nil {
		return err
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		protocol TEXT,    
		external_id TEXT, 
		filename TEXT,
		status TEXT,
		added_at DATETIME
	);`

	_, err = DB.Exec(createTable)
	return err
}

func AddDownload(protocol, filename string) string {
	id := fmt.Sprintf("%x", time.Now().UnixNano())

	stmt, _ := DB.Prepare("INSERT INTO downloads(protocol, external_id, filename, status, added_at) VALUES(?, ?, ?, ?, ?)")
	_, err := stmt.Exec(protocol, id, filename, "downloading", time.Now())

	if err != nil {
		// Log structured error
		slog.Error("Failed to insert download",
			"protocol", protocol,
			"filename", filename,
			"error", err,
		)
		return ""
	}

	// Log success with context
	slog.Info("Download added",
		"protocol", protocol,
		"id", id,
		"filename", filename,
	)
	return id
}

// ... GetQueue and Delete functions remain similar, just remove old logs ...
func DeleteDownload(externalID string) bool {
	res, err := DB.Exec("DELETE FROM downloads WHERE external_id = ?", externalID)
	if err != nil {
		slog.Error("Failed to delete download",
			"external_id", externalID,
			"error", err,
		)
		return false
	}
	count, _ := res.RowsAffected()
	return count > 0
}

// --- SABnzbd Specific Fetcher ---

func GetSabQueue() []models.SabSlot {
	rows, err := DB.Query("SELECT external_id, filename, status FROM downloads WHERE protocol='usenet' ORDER BY added_at DESC")
	if err != nil {
		return []models.SabSlot{}
	}
	defer rows.Close()

	var slots []models.SabSlot
	index := 0
	for rows.Next() {
		var s models.SabSlot
		rows.Scan(&s.NzoID, &s.Filename, &s.Status)
		s.Size = "2.0 GB" // Mock size
		s.Index = index
		slots = append(slots, s)
		index++
	}
	return slots
}

// --- qBittorrent Specific Fetcher ---

func GetQBitQueue() []models.QBitTorrent {
	rows, err := DB.Query("SELECT external_id, filename, status FROM downloads WHERE protocol='torrent' ORDER BY added_at DESC")
	if err != nil {
		return []models.QBitTorrent{}
	}
	defer rows.Close()

	var torrents []models.QBitTorrent
	for rows.Next() {
		var t models.QBitTorrent
		var status string
		rows.Scan(&t.Hash, &t.Name, &status)

		t.State = status
		t.Size = 2000000000 // 2 GB in bytes
		t.Progress = 0.45   // 45%
		t.Dlspeed = 5000000 // 5 MB/s
		t.Category = "tv-sonarr"
		torrents = append(torrents, t)
	}
	return torrents
}
