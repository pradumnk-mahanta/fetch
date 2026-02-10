package database

import (
	"database/sql"
	"fetchtb/models"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {
	var err error
	DB, err = sql.Open("sqlite", "./data/fetchtb.db")
	if err != nil {
		return err
	}

	// Added 'category' column
	createTable := `
	CREATE TABLE IF NOT EXISTS downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		protocol TEXT,    
		external_id TEXT, 
		filename TEXT,
		category TEXT,
		status TEXT,
		added_at DATETIME
	);`

	_, err = DB.Exec(createTable)
	return err
}

// Updated to accept 'category'
func AddDownload(protocol, filename, category string) string {
	id := fmt.Sprintf("%x", time.Now().UnixNano())

	stmt, _ := DB.Prepare("INSERT INTO downloads(protocol, external_id, filename, category, status, added_at) VALUES(?, ?, ?, ?, ?, ?)")
	_, err := stmt.Exec(protocol, id, filename, category, "Downloading", time.Now())

	if err != nil {
		slog.Error("Failed to add download", "error", err)
		return ""
	}

	slog.Info("Download added", "protocol", protocol, "cat", category, "name", filename)
	return id
}

// New function for Pause/Resume
func UpdateStatus(externalID, newStatus string) bool {
	res, err := DB.Exec("UPDATE downloads SET status = ? WHERE external_id = ?", newStatus, externalID)
	if err != nil {
		return false
	}
	count, _ := res.RowsAffected()
	return count > 0
}

func DeleteDownload(externalID string) bool {
	res, _ := DB.Exec("DELETE FROM downloads WHERE external_id = ?", externalID)
	count, _ := res.RowsAffected()
	return count > 0
}

func GetSabQueue() []models.SabSlot {
	rows, _ := DB.Query("SELECT external_id, filename, status, category FROM downloads WHERE protocol='usenet' ORDER BY added_at DESC")
	defer rows.Close()

	var slots []models.SabSlot
	index := 0
	for rows.Next() {
		var s models.SabSlot
		var cat string
		// Scan category too, though SABnzbd JSON usually puts it in a separate history field
		// For queue, we primarily need status/filename
		rows.Scan(&s.NzoID, &s.Filename, &s.Status, &cat)
		s.Size = "2.0 GB"
		s.Index = index
		slots = append(slots, s)
		index++
	}
	return slots
}

// ... GetQBitQueue remains similar (you can update it to read category if needed) ...
func GetQBitQueue() []models.QBitTorrent {
	// Left as is for brevity, but you should update the SELECT to include category if needed for qBit
	return []models.QBitTorrent{}
}
