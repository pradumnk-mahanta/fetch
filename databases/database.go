package databases

import (
	"encoding/json"
	"errors"
	"fetch/config"
	"fetch/logger"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

const dbPath = "/data/fetch.db"

type LocalDownloadsInstanceItem struct {
	ID                          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DownloadID                  uint      `gorm:"index;column:download_id" json:"download_id"`
	DownloadType                string    `gorm:"column:download_type" json:"download_type"`
	Category                    string    `gorm:"column:category" json:"category"`
	FileName                    string    `gorm:"column:file_name" json:"file_name"`
	FilePath                    string    `gorm:"column:file_path" json:"file_path"`
	FileSize                    int64     `gorm:"column:file_size" json:"file_size"`
	Status                      string    `gorm:"column:status" json:"status"`
	ExternalProviderID          string    `gorm:"column:external_provider_id" json:"external_provider_id"`
	ExternalProviderItemID      string    `gorm:"column:external_provider_item_id" json:"external_provider_item_id"`
	ExternalProviderDownloadURL string    `gorm:"column:external_provider_download_url" json:"external_provider_download_url"`
	RetryCounter                int       `gorm:"column:retry_counter;default:0" json:"retry_counter"`
	AddedAt                     time.Time `gorm:"column:added_at" json:"added_at"`
}

func (i *LocalDownloadsInstanceItem) ToJSON() string {
	b, _ := json.Marshal(i)
	return string(b)
}

func (i *LocalDownloadsInstanceItem) Add() error {
	result := DB.Create(i)
	return result.Error
}

func (i *LocalDownloadsInstanceItem) IDString() string {
	return strconv.FormatUint(uint64(i.ID), 10)
}

func (i *LocalDownloadsInstanceItem) DownloadIDString() string {
	return strconv.FormatUint(uint64(i.DownloadID), 10)
}

type LocalDownloadsInstance struct {
	ID                         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Protocol                   string    `gorm:"column:protocol" json:"protocol"`
	Provider                   string    `gorm:"column:provider" json:"provider"`
	DownloadName               string    `gorm:"column:download_name" json:"download_name"`
	OriginalDownloadUrl        string    `gorm:"column:original_download_url" json:"original_download_url"`
	OriginalDownloadFile       []byte    `gorm:"column:original_download_file;type:blob" json:"-"`
	OriginalDownloadReference  string    `gorm:"column:original_download_reference;type:blob" json:"original_download_reference"` //hash store
	Category                   string    `gorm:"column:category" json:"category"`
	Status                     string    `gorm:"column:status" json:"status"`
	ExternalProviderID         string    `gorm:"column:external_provider_id" json:"external_provider_id"`
	ExternalProviderDataObject string    `gorm:"column:external_provider_data_object" json:"-"`
	AddedAt                    time.Time `gorm:"column:added_at" json:"added_at"`
	CompletedAt                time.Time `gorm:"column:completed_at" json:"completed_at"`

	DownloadItems []LocalDownloadsInstanceItem `gorm:"foreignKey:DownloadID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"download_items,omitempty"`
}

func (d *LocalDownloadsInstance) ToJSON() string {
	if d.DownloadItems == nil {
		d.DownloadItems = []LocalDownloadsInstanceItem{}
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func (d *LocalDownloadsInstance) Add() error {
	result := DB.Create(d)
	return result.Error
}

func (d *LocalDownloadsInstance) IDString() string {
	return strconv.FormatUint(uint64(d.ID), 10)
}

func LocalDownloadsInstancesToJSONArray(downloads []LocalDownloadsInstance) string {
	if downloads == nil {
		downloads = []LocalDownloadsInstance{}
	}
	for i := range downloads {
		if downloads[i].DownloadItems == nil {
			downloads[i].DownloadItems = []LocalDownloadsInstanceItem{}
		}
	}
	b, _ := json.Marshal(downloads)
	return string(b)
}

func LocalDownloadsInstanceItemsToJSONArray(items []LocalDownloadsInstanceItem) string {
	if items == nil {
		items = []LocalDownloadsInstanceItem{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	logger.Log.Infow("Database initialized successfully", "path", dbPath)
	if err := DB.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return err
	}

	if err := DB.Exec("PRAGMA busy_timeout = 5000;").Error; err != nil {
		return err
	}

	DB.Exec("PRAGMA foreign_keys = ON;")

	err = DB.AutoMigrate(
		&LocalDownloadsInstance{},
		&LocalDownloadsInstanceItem{},
	)

	if err != nil {
		return err
	}

	return nil
}

func AddLocalDownload(download LocalDownloadsInstance) (string, error) {

	dl := download
	if err := dl.Add(); err != nil {
		logger.Log.Errorw("Failed to add download to db", "error", err)
		return "", err
	}

	logger.Log.Infow("Download Added", "protocol", dl.Protocol, "cat", dl.Category, "name", dl.DownloadName, "id", dl.IDString())
	return dl.IDString(), nil
}

func AddLocalDownloadItem(downloadId string, downloadType string, category string, fileName string, filePath string, fileSize int64, providerId string, providerItemId string, providerDownloadUrl string) (string, error) {

	dlID, err := strconv.ParseUint(downloadId, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID format", "id", dlID, "error", err)
		return "", err
	}

	item := &LocalDownloadsInstanceItem{
		DownloadID:                  uint(dlID),
		DownloadType:                downloadType,
		Category:                    category,
		FileName:                    fileName,
		FilePath:                    filePath,
		FileSize:                    fileSize,
		Status:                      config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED,
		ExternalProviderID:          providerId,
		ExternalProviderItemID:      providerItemId,
		ExternalProviderDownloadURL: providerDownloadUrl,
		RetryCounter:                0,
		AddedAt:                     time.Now(),
	}

	if err := item.Add(); err != nil {
		logger.Log.Errorw("Failed to add download item to db", "error", err)
		return "", err
	}

	logger.Log.Infow("Download Item Added", "id", item.IDString())
	return item.IDString(), nil
}

func GetLocalDownloadDetails(id string) (LocalDownloadsInstance, error) {

	var download LocalDownloadsInstance
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID format", "id", id, "error", err)
		return download, err
	}

	result := DB.Preload("DownloadItems").First(&download, uint(uID))

	if result.Error != nil {
		return download, result.Error
	}

	if download.DownloadItems == nil {
		download.DownloadItems = []LocalDownloadsInstanceItem{}
	}

	return download, nil
}

func GetLocalDownloadsByReference(reference string) *LocalDownloadsInstance {
	var download LocalDownloadsInstance

	err := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Where("protocol = ? AND original_download_reference = ?", config.ProtocolTorrent, reference).
		First(&download).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		logger.Log.Errorw("Database error while fetching by reference", "reference", reference, "error", err)
		return nil
	}

	return &download
}

func GetLocalDownloadsByReferences(references string) []LocalDownloadsInstance {

	var downloads []LocalDownloadsInstance

	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Model(&LocalDownloadsInstance{}).Where("protocol = ?", config.ProtocolTorrent)

	if references != "" {
		hashes := strings.Split(references, "|")
		query.Where("original_download_reference IN ?", hashes)
	}

	errFind := query.Find(&downloads).Error
	if errFind != nil {
		logger.Log.Errorw("Database error while fetching by references", "references", references, "error", errFind)
	}

	return downloads
}

func GetLocalDownloadsByProtocol(protocol string) []LocalDownloadsInstance {

	var downloads []LocalDownloadsInstance
	result := DB.Preload("DownloadItems").Model(&LocalDownloadsInstance{}).
		Where("protocol = ?", protocol).
		Find(&downloads)

	if result.Error != nil {
		logger.Log.Errorw("Database error while fetching by references", "protocol", protocol, "error", result.Error)
	}

	return downloads
}

func GetLocalDownloadItemDetails(id string) (LocalDownloadsInstanceItem, error) {
	var downloadItem LocalDownloadsInstanceItem

	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid Item ID format", "id", id, "error", err)
		return downloadItem, err
	}

	result := DB.First(&downloadItem, uint(uID))

	if result.Error != nil {
		logger.Log.Errorw("Failed to fetch download item", "id", id, "error", result.Error)
		return downloadItem, result.Error
	}

	return downloadItem, nil
}

func UpdateLocalDownloadItemExternalUrl(id string, externalDownloadUrl string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID format", "id", id, "error", err)
		return err
	}

	result := DB.Model(&LocalDownloadsInstanceItem{}).
		Where("id = ?", uint(uID)).
		Update("external_provider_download_url", externalDownloadUrl)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update external URL", "id", id, "error", result.Error)
		return result.Error
	}

	return nil
}

func UpdateLocalDownloadProviderId(id string, externalIDProvider string, status string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for provider update", "id", id, "error", err)
		return err
	}

	result := DB.Model(&LocalDownloadsInstance{}).
		Where("id = ?", uint(uID)).
		Updates(map[string]interface{}{
			"external_provider_id": externalIDProvider,
			"status":               status,
		})

	if result.Error != nil {
		logger.Log.Errorw("Failed to update download provider info", "id", id, "error", result.Error)
		return result.Error
	}

	return nil
}

func UpdateLocalDownloadStatus(id string, status string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for status update", "id", id, "error", err)
		return err
	}

	updateData := map[string]interface{}{
		"status": status,
	}

	if status == config.DOWNLOAD_STATUS_CLIENT_COMPLETED || status == config.DOWNLOAD_STATUS_CLIENT_FAILED {
		now := time.Now()
		updateData["completed_at"] = &now
	} else {
		updateData["completed_at"] = nil
	}

	result := DB.Model(&LocalDownloadsInstance{}).
		Where("id = ?", uint(uID)).
		Updates(updateData)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update download status", "id", id, "error", result.Error)
		return result.Error
	}

	return nil
}

func UpdateLocalDownloadItemStatus(id string, status string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for item status update", "id", id, "error", err)
		return err
	}

	result := DB.Model(&LocalDownloadsInstanceItem{}).
		Where("id = ?", uint(uID)).
		Update("status", status)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update download item status", "id", id, "error", result.Error)
		return result.Error
	}

	logger.Log.Debugw("Updated Download Item Status", "Item", id, "Status", status)
	return nil
}

func UpdateLocalDownloadItemRetryCounter(id string, retryCount int) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for retry counter update", "id", id, "error", err)
		return err
	}

	result := DB.Model(&LocalDownloadsInstanceItem{}).
		Where("id = ?", uint(uID)).
		Update("retry_counter", retryCount)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update download item retry count", "id", id, "error", result.Error)
		return result.Error
	}

	logger.Log.Debugw("Updated Download Item RetryCount", "Item", id, "RetryCount", retryCount)
	return nil
}

func GetLocalDownloads() ([]LocalDownloadsInstance, error) {
	var downloads []LocalDownloadsInstance

	logger.Log.Debugw("Fetching all local downloads from database")

	result := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("added_at ASC")
	}).Find(&downloads)

	if result.Error != nil {
		logger.Log.Errorw("Failed to retrieve downloads", "error", result.Error)
		return nil, result.Error
	}

	for i := range downloads {
		if downloads[i].DownloadItems == nil {
			downloads[i].DownloadItems = []LocalDownloadsInstanceItem{}
		}
	}

	logger.Log.Debugw("Successfully retrieved downloads", "count", len(downloads))
	return downloads, nil
}

func GetLocalPendingDownloads(protocol string) ([]LocalDownloadsInstance, error) {
	var downloads []LocalDownloadsInstance

	logger.Log.Debugw("Fetching pending downloads from DB", "protocol", protocol)

	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("added_at ASC")
	}).Where("status NOT IN ?", []string{
		config.DOWNLOAD_STATUS_CLIENT_COMPLETED,
		config.DOWNLOAD_STATUS_CLIENT_FAILED,
	})

	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}

	err := query.Find(&downloads).Error

	if err != nil {
		logger.Log.Errorw("Failed to fetch pending downloads", "protocol", protocol, "error", err)
		return nil, err
	}

	for i := range downloads {
		if downloads[i].DownloadItems == nil {
			downloads[i].DownloadItems = []LocalDownloadsInstanceItem{}
		}
	}

	logger.Log.Debugw("Pending downloads retrieved", "protocol", protocol, "count", len(downloads))
	return downloads, nil
}

func GetLocalCompletedDownloads(protocol string) ([]LocalDownloadsInstance, error) {
	var downloads []LocalDownloadsInstance

	logger.Log.Debugw("Fetching completed/failed downloads from DB")

	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("added_at ASC")
	}).Where("status IN ?", []string{
		config.DOWNLOAD_STATUS_CLIENT_COMPLETED,
		config.DOWNLOAD_STATUS_CLIENT_FAILED,
	})

	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}

	err := query.Find(&downloads).Error

	if err != nil {
		logger.Log.Errorw("Failed to fetch completed downloads", "error", err)
		return nil, err
	}

	for i := range downloads {
		if downloads[i].DownloadItems == nil {
			downloads[i].DownloadItems = []LocalDownloadsInstanceItem{}
		}
	}

	logger.Log.Debugw("Completed downloads retrieved", "count", len(downloads))
	return downloads, nil
}

func UpdateLocalDownloadProviderData(id string, providerData string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for provider data update", "id", id, "error", err)
		return err
	}

	result := DB.Model(&LocalDownloadsInstance{}).
		Where("id = ?", uint(uID)).
		Update("external_provider_data_object", providerData)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update provider data object", "id", id, "error", result.Error)
		return result.Error
	}

	logger.Log.Debugw("Successfully updated provider data", "id", id)
	return nil
}

func GetLocalDownloadsAddedToProviderCount() int {
	var count int64

	targetStatuses := []string{
		config.DOWNLOAD_STATUS_PROVIDER_ADDED,
		config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING,
		config.DOWNLOAD_STATUS_PROVIDER_PROCESSING,
	}

	logger.Log.Debugw("Counting downloads by provider status", "target_statuses", targetStatuses)

	err := DB.Model(&LocalDownloadsInstance{}).
		Where("status IN ?", targetStatuses).
		Count(&count).Error

	if err != nil {
		logger.Log.Errorw("Database error while counting provider downloads", "error", err)
		return 999
	}

	logger.Log.Debugw("Provider download count retrieved", "count", count)
	return int(count)
}

func GetLocalDownloadItems() ([]LocalDownloadsInstanceItem, error) {
	var items []LocalDownloadsInstanceItem

	logger.Log.Debugw("Fetching all local download items from DB")

	result := DB.Order("added_at ASC").Find(&items)

	if result.Error != nil {
		logger.Log.Errorw("Failed to fetch all download items", "error", result.Error)
		return nil, result.Error
	}

	if items == nil {
		items = []LocalDownloadsInstanceItem{}
	}

	logger.Log.Debugw("Successfully retrieved all download items", "count", len(items))
	return items, nil
}

func GetLocalDownloadItemsForDownload(downloadId string) ([]LocalDownloadsInstanceItem, error) {
	var items []LocalDownloadsInstanceItem

	uID, err := strconv.ParseUint(downloadId, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid downloadId for items fetch", "id", downloadId, "error", err)
		return nil, err
	}

	logger.Log.Debugw("Fetching items for specific download", "download_id", uID)

	result := DB.Where("download_id = ?", uint(uID)).
		Order("added_at ASC").
		Find(&items)

	if result.Error != nil {
		logger.Log.Errorw("Failed to fetch items for download", "download_id", uID, "error", result.Error)
		return nil, result.Error
	}

	if items == nil {
		items = []LocalDownloadsInstanceItem{}
	}

	logger.Log.Debugw("Retrieved items for download", "download_id", uID, "count", len(items))
	return items, nil
}

func DeleteLocalDownloadItem(id string) error {
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for item deletion", "id", id, "error", err)
		return err
	}

	result := DB.Delete(&LocalDownloadsInstanceItem{}, uint(uID))

	if result.Error != nil {
		logger.Log.Errorw("Failed to delete download item", "id", id, "error", result.Error)
		return result.Error
	}

	logger.Log.Debugw("Deleted Download Item", "id", id, "rows_affected", result.RowsAffected)
	return nil
}

func DeleteLocalDownload(id string) error {

	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for download deletion", "id", id, "error", err)
		return err
	}

	result := DB.Delete(&LocalDownloadsInstance{}, uint(uID))

	if result.Error != nil {
		logger.Log.Errorw("Failed to delete download", "id", id, "error", result.Error)
		return result.Error
	}

	logger.Log.Debugw("Deleted Local Download and associated items",
		"id", id,
		"rows_affected", result.RowsAffected,
	)
	return nil
}
