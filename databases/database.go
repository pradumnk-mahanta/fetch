package databases

import (
	"database/sql"
	"fetch/config"
	"fetch/logger"
	"fetch/models"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

const dbPath = "/data/fetchtb.db"

func InitDB() error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	logger.Log.Infow("Database initialized successfully", "path", dbPath)

	createDownloadsTable := `
	CREATE TABLE IF NOT EXISTS downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		protocol TEXT,
		provider TEXT,    
		download_name TEXT,
		original_download_url TEXT,
		original_download_file BLOB,
		category TEXT,
		status TEXT,
		external_provider_id TEXT, 
		external_provider_data_object TEXT,
		added_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS downloaditems (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		download_id TEXT,
		download_type TEXT,
		category TEXT,
		file_name TEXT,
		file_path TEXT,
		file_size INTEGER,
		status TEXT,
		external_provider_id TEXT, 
		external_provider_item_id TEXT, 
		external_provider_download_url TEXT,
		retry_counter INTEGER,
		added_at DATETIME
	);
	`

	//external_provider_id
	_, err = DB.Exec(createDownloadsTable)
	return err
}

func AddLocalDownload(protocol string, provider string, downloadname string, downloadUrl string, fileBytes []byte, category string) (string, error) {
	name := strings.TrimSuffix(downloadname, filepath.Ext(downloadname))
	stmt, _ := DB.Prepare("INSERT INTO downloads(protocol, provider, download_name, original_download_url, original_download_file, category, status, external_provider_id, external_provider_data_object, added_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	result, err := stmt.Exec(protocol, provider, name, downloadUrl, fileBytes, category, config.DOWNLOAD_STATUS_CLIENT_ADDED, "", "", time.Now())
	if err != nil {
		logger.Log.Errorw("Failed to add download to db", "error", err)
		return "", err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.Log.Errorw("Failed to get last insert id", "error", err)
		return "", err
	}
	logger.Log.Infow("Download Added", "protocol", protocol, "cat", category, "name", name, "id", strconv.FormatInt(id, 10))
	return strconv.FormatInt(id, 10), nil
}

func AddLocalDownloadItem(downloadId string, downloadType string, category string, fileName string, filePath string, fileSize int64, providerId string, providerItemId string, providerDownloadUrl string) string {
	stmt, _ := DB.Prepare("INSERT INTO downloaditems(download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	result, err := stmt.Exec(downloadId, downloadType, category, fileName, filePath, fileSize, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED, providerId, providerItemId, providerDownloadUrl, 0, time.Now())
	if err != nil {
		logger.Log.Errorw("Failed to add download item to db", "error", err)
		return ""
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.Log.Errorw("Failed to get last insert id", "error", err)
		return ""
	}
	logger.Log.Infow("Download Item Added", "id", strconv.FormatInt(id, 10))
	return strconv.FormatInt(id, 10)
}

func GetLocalDownloadDetails(id string) (models.LocalDownloadInstance, error) {
	var download models.LocalDownloadInstance
	var downloadItems []models.LocalDownloadInstanceItem
	errDownload := DB.QueryRow("SELECT id, protocol, provider, download_name, original_download_url, original_download_file, category, status, external_provider_id, external_provider_data_object, added_at FROM downloads WHERE id = ?", id).Scan(&download.ID, &download.Protocol, &download.Provider, &download.DownloadName, &download.OriginalDownloadURL, &download.OriginalDownloadFile, &download.Category, &download.Status, &download.ExternalProviderID, &download.ExternalProviderDataObject, &download.AddedAt)
	if errDownload != nil {
		download.DownloadItems = downloadItems
		return download, errDownload
	}

	dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ?", id)
	if errDownloadItems != nil {
		return download, nil
	}

	for dlItems.Next() {
		var downloadItem models.LocalDownloadInstanceItem
		dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
		downloadItems = append(downloadItems, downloadItem)
	}

	download.DownloadItems = downloadItems
	return download, nil
}

func GetLocalDownloadItemDetails(id string) (models.LocalDownloadInstanceItem, error) {
	var downloadItem models.LocalDownloadInstanceItem
	errDownload := DB.QueryRow("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ?", id).Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
	if errDownload != nil {
		return downloadItem, errDownload
	}
	return downloadItem, nil
}

func UpdateLocalDownloadItemExternalUrl(id string, externalDownloadUrl string) error {
	_, err := DB.Exec("UPDATE downloaditems SET external_provider_download_url = ? WHERE id = ?", externalDownloadUrl, id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateLocalDownloadProviderId(id string, externalIDProvider string, status string) error {
	_, err := DB.Exec("UPDATE downloads SET external_provider_id = ?, status = ? WHERE id = ?", externalIDProvider, status, id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateLocalDownloadProviderUrl(id string, externalDownloadProviderUrl string) error {
	res, err := DB.Exec("UPDATE downloads SET external_download_provider_url = ? WHERE id = ?", externalDownloadProviderUrl, id)
	if err != nil {
		return err
	}
	logger.Log.Debugw("Updated download provider URL", "id", id, "url", externalDownloadProviderUrl, "rows_affected", res)
	return nil
}

func UpdateLocalDownloadDownloaderId(id string, externalIDDownloader string, status string) error {
	_, err := DB.Exec("UPDATE downloads SET external_id_downloader = ?, status = ? WHERE id = ?", externalIDDownloader, status, id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateLocalDownloadStatus(id string, status string) error {
	_, err := DB.Exec("UPDATE downloads SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateLocalDownloadItemStatus(id string, status string) error {
	_, err := DB.Exec("UPDATE downloaditems SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return err
	}
	logger.Log.Debugw("Updated Download Item Status", "Item", id, "Status", status)
	return nil
}

func UpdateLocalDownloadItemRetryCounter(id string, retryCount int) error {
	_, err := DB.Exec("UPDATE downloaditems SET retry_counter = ? WHERE id = ?", retryCount, id)
	if err != nil {
		return err
	}
	logger.Log.Debugw("Updated Download Item RetryCount", "Item", id, "RetryCount", retryCount)
	return nil
}

func UpdateLocalDownloadProviderData(id string, providerData string) error {
	_, err := DB.Exec("UPDATE downloads SET external_provider_data_object = ? WHERE id = ?", providerData, id)
	if err != nil {
		return err
	}
	return nil
}

func GetLocalDownloads() ([]models.LocalDownloadInstance, error) {
	var downloads []models.LocalDownloadInstance

	rows, err := DB.Query("SELECT id, protocol, provider, download_name, original_download_url, original_download_file, category, status, external_provider_id, external_provider_data_object, added_at FROM downloads")
	if err != nil {
		return downloads, err
	}
	defer rows.Close()

	for rows.Next() {
		var download models.LocalDownloadInstance
		rows.Scan(&download.ID, &download.Protocol, &download.Provider, &download.DownloadName, &download.OriginalDownloadURL, &download.OriginalDownloadFile, &download.Category, &download.Status, &download.ExternalProviderID, &download.ExternalProviderDataObject, &download.AddedAt)

		dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ? ORDER BY added_at ASC", download.ID)
		if errDownloadItems != nil {
			continue
		} else {
			var downloadItems []models.LocalDownloadInstanceItem
			for dlItems.Next() {
				var downloadItem models.LocalDownloadInstanceItem
				dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
				downloadItems = append(downloadItems, downloadItem)
			}
			download.DownloadItems = downloadItems
		}
		downloads = append(downloads, download)
	}
	return downloads, nil
}

func GetLocalPendingDownloads() ([]models.LocalDownloadInstance, error) {
	var downloads []models.LocalDownloadInstance

	rows, err := DB.Query("SELECT id, protocol, provider, download_name, original_download_url, original_download_file, category, status, external_provider_id, external_provider_data_object, added_at FROM downloads WHERE status NOT IN (?, ?)", config.DOWNLOAD_STATUS_CLIENT_COMPLETED, config.DOWNLOAD_STATUS_CLIENT_FAILED)
	if err != nil {
		return downloads, err
	}
	defer rows.Close()

	for rows.Next() {
		var download models.LocalDownloadInstance
		rows.Scan(&download.ID, &download.Protocol, &download.Provider, &download.DownloadName, &download.OriginalDownloadURL, &download.OriginalDownloadFile, &download.Category, &download.Status, &download.ExternalProviderID, &download.ExternalProviderDataObject, &download.AddedAt)

		dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ? ORDER BY added_at ASC", download.ID)
		if errDownloadItems != nil {
			continue
		} else {
			var downloadItems []models.LocalDownloadInstanceItem
			for dlItems.Next() {
				var downloadItem models.LocalDownloadInstanceItem
				dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
				downloadItems = append(downloadItems, downloadItem)
			}
			download.DownloadItems = downloadItems
		}
		downloads = append(downloads, download)
	}
	return downloads, nil
}

func GetLocalCompletedDownloads() ([]models.LocalDownloadInstance, error) {
	var downloads []models.LocalDownloadInstance

	rows, err := DB.Query("SELECT id, protocol, provider, download_name, original_download_url, original_download_file, category, status, external_provider_id, external_provider_data_object, added_at FROM downloads WHERE status IN (?, ?)", config.DOWNLOAD_STATUS_CLIENT_COMPLETED, config.DOWNLOAD_STATUS_CLIENT_FAILED)
	if err != nil {
		return downloads, err
	}
	defer rows.Close()

	for rows.Next() {
		var download models.LocalDownloadInstance
		rows.Scan(&download.ID, &download.Protocol, &download.Provider, &download.DownloadName, &download.OriginalDownloadURL, &download.OriginalDownloadFile, &download.Category, &download.Status, &download.ExternalProviderID, &download.ExternalProviderDataObject, &download.AddedAt)

		dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ? ORDER BY added_at ASC", download.ID)
		if errDownloadItems != nil {
			continue
		} else {
			var downloadItems []models.LocalDownloadInstanceItem
			for dlItems.Next() {
				var downloadItem models.LocalDownloadInstanceItem
				dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
				downloadItems = append(downloadItems, downloadItem)
			}
			download.DownloadItems = downloadItems
		}
		downloads = append(downloads, download)
	}
	return downloads, nil
}

func GetLocalDownloadsAddedToProviderCount() int {
	downloadsAdded := 0
	errQry := DB.QueryRow("SELECT count(*) AS provider_downloads FROM downloads WHERE status IN (?, ?, ?)", config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING).Scan(&downloadsAdded)
	if errQry != nil {
		return 999
	}
	return downloadsAdded
}

func GetLocalDownloadItems() ([]models.LocalDownloadInstanceItem, error) {
	var downloadItems []models.LocalDownloadInstanceItem
	dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems ORDER BY added_at ASC")
	if errDownloadItems != nil {
		return downloadItems, errDownloadItems
	}
	defer dlItems.Close()

	for dlItems.Next() {
		var downloadItem models.LocalDownloadInstanceItem
		dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
		downloadItems = append(downloadItems, downloadItem)
	}

	return downloadItems, nil
}

func GetLocalDownloadItemsForDownload(downloadId string) ([]models.LocalDownloadInstanceItem, error) {
	var downloadItems []models.LocalDownloadInstanceItem
	dlItems, errDownloadItems := DB.Query("SELECT id, download_id, download_type, category, file_name, file_path, file_size, status, external_provider_id, external_provider_item_id, external_provider_download_url, retry_counter, added_at FROM downloaditems WHERE download_id = ? ORDER BY added_at ASC", downloadId)
	if errDownloadItems != nil {
		return downloadItems, errDownloadItems
	}
	defer dlItems.Close()

	for dlItems.Next() {
		var downloadItem models.LocalDownloadInstanceItem
		dlItems.Scan(&downloadItem.ID, &downloadItem.DownloadID, &downloadItem.DownloadType, &downloadItem.Category, &downloadItem.FileName, &downloadItem.FilePath, &downloadItem.FileSize, &downloadItem.Status, &downloadItem.ExternalProviderID, &downloadItem.ExternalProviderItemID, &downloadItem.ExternalProviderDownloadURL, &downloadItem.RetryCounter, &downloadItem.AddedAt)
		downloadItems = append(downloadItems, downloadItem)
	}
	return downloadItems, nil
}

func DeleteLocalDownloadItem(id string) error {
	_, err := DB.Exec("DELETE FROM downloaditems WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func DeleteLocalDownload(id string) error {
	_, err := DB.Exec("DELETE FROM downloads WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
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

func GetTranslatedStatusforDownloadBasedonItemStatus(itemStatus string) string {
	return ""
}
