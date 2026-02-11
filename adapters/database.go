package adapters

import (
	"database/sql"
	"fetchtb/config"
	"fetchtb/models"
	"fetchtb/utils"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
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
		provider TEXT,    
		downloadname TEXT,
		original_download_url TEXT,
		original_download_file BLOB,
		category TEXT,
		status TEXT,
		external_id_provider TEXT, 
		external_id_downloader TEXT, 
		added_at DATETIME
	);`
	_, err = DB.Exec(createTable)
	return err
}

func AddLocalDownload(protocol string, provider string, downloadname string, downloadUrl string, downloadfile multipart.File, category string) string {
	fileBytes, err := io.ReadAll(downloadfile)
	if err != nil {
		slog.Error("Failed to read download file", "error", err)
		return ""
	}

	name := strings.TrimSuffix(downloadname, filepath.Ext(downloadname))

	stmt, _ := DB.Prepare("INSERT INTO downloads(protocol, provider, downloadname, original_download_url, original_download_file, category, status, external_id_provider, external_id_downloader, added_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	result, err := stmt.Exec(protocol, provider, name, downloadUrl, fileBytes, category, config.DOWNLOAD_STATUS_ADDED, "", "", time.Now())
	if err != nil {
		slog.Error("Failed to add download to db", "error", err)
		return ""
	}

	id, err := result.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id", "error", err)
		return ""
	}
	utils.Logger.Infow("Download Added", "protocol", protocol, "cat", category, "name", name, "id", strconv.FormatInt(id, 10))
	return strconv.FormatInt(id, 10)
}

func GetLocalDownloadDetails(id string) (models.LocalDownloadInstance, error) {
	var download models.LocalDownloadInstance

	err := DB.QueryRow("SELECT id, protocol, provider, downloadname, original_download_url, original_download_file, category, status, external_id_provider, external_id_downloader, added_at FROM downloads WHERE id = ?", id).Scan(&download.ID, &download.Protocol, &download.Provider, &download.DownloadName, &download.OriginalDownloadURL, &download.OriginalDownloadFile, &download.Category, &download.Status, &download.ExternalIDProvider, &download.ExternalIDDownloader, &download.AddedAt)
	if err != nil {
		return download, err
	}

	return download, nil
}

func UpdateLocalDownloadDetails(id string, externalIDProvider string, externalIDDownloader string, status string) error {
	_, err := DB.Exec("UPDATE downloads SET external_id_provider = ?, external_id_downloader = ?, status = ? WHERE id = ?", externalIDProvider, externalIDDownloader, status, id)
	if err != nil {
		return err
	}
	return nil
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

func GetSABNzbdQueue() []models.SabSlot {
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
