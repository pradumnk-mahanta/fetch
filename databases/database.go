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
	Protocol                    string    `gorm:"column:protocol" json:"protocol"`
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
	logger.Log.Debugw("Adding LocalDownloadsInstanceItem to DB", "download_id", i.DownloadID)
	result := DB.Create(i)
	if result.Error != nil {
		logger.Log.Errorw("Failed to add LocalDownloadsInstanceItem", "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully added LocalDownloadsInstanceItem", "id", i.ID)
	return nil
}

func (i *LocalDownloadsInstanceItem) Delete() error {
	logger.Log.Debugw("Deleting LocalDownloadsInstanceItem", "id", i.ID)
	errTx := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&LocalDownloadsInstanceItem{}, i.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			logger.Log.Warnw("No LocalDownloadsInstanceItem found to delete", "id", i.ID)
			return nil
		}
		return nil
	})
	if errTx != nil {
		logger.Log.Errorw("Failed to delete LocalDownloadsInstanceItem", "id", i.ID, "error", errTx)
		return errTx
	}
	logger.Log.Infow("Successfully deleted LocalDownloadsInstanceItem", "id", i.ID)
	return nil
}

func (i *LocalDownloadsInstanceItem) IDString() string {
	return strconv.FormatUint(uint64(i.ID), 10)
}

func (i *LocalDownloadsInstanceItem) DownloadIDString() string {
	return strconv.FormatUint(uint64(i.DownloadID), 10)
}

func LocalDownloadsInstanceItemsToJSONArray(items []LocalDownloadsInstanceItem) string {
	if items == nil {
		items = []LocalDownloadsInstanceItem{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

type LocalDownloadsInstance struct {
	ID                         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Protocol                   string    `gorm:"column:protocol" json:"protocol"`
	Provider                   string    `gorm:"column:provider" json:"provider"`
	DownloadName               string    `gorm:"column:download_name" json:"download_name"`
	DownloadType               string    `gorm:"column:download_type:default:'Individual File for Download'" json:"download_type"`
	OriginalDownloadUrl        string    `gorm:"column:original_download_url" json:"original_download_url"`
	OriginalDownloadFile       []byte    `gorm:"column:original_download_file;type:blob" json:"-"`
	OriginalDownloadReference  string    `gorm:"column:original_download_reference;type:string" json:"original_download_reference"` //hash store
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
	logger.Log.Debugw("Adding LocalDownloadsInstance to DB", "download_name", d.DownloadName)
	result := DB.Create(d)
	if result.Error != nil {
		logger.Log.Errorw("Failed to add LocalDownloadsInstance", "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully added LocalDownloadsInstance", "id", d.ID)
	return nil
}

func (i *LocalDownloadsInstance) Delete() error {
	logger.Log.Debugw("Deleting LocalDownloadsInstance", "id", i.ID)
	errTx := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&LocalDownloadsInstance{}, i.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			logger.Log.Warnw("No LocalDownloadsInstance found to delete", "id", i.ID)
			return nil
		}
		return nil
	})
	if errTx != nil {
		logger.Log.Errorw("Failed to delete LocalDownloadsInstance", "id", i.ID, "error", errTx)
		return errTx
	}
	logger.Log.Infow("Successfully deleted LocalDownloadsInstance", "id", i.ID)
	return nil
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

func InitDB() error {
	var err error
	logger.Log.Debugw("Initializing database", "path", dbPath)
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.Log.Errorw("Failed to open database", "path", dbPath, "error", err)
		return err
	}

	if err := DB.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		logger.Log.Errorw("Failed to set PRAGMA journal_mode", "error", err)
		return err
	}

	if err := DB.Exec("PRAGMA busy_timeout = 5000;").Error; err != nil {
		logger.Log.Errorw("Failed to set PRAGMA busy_timeout", "error", err)
		return err
	}

	if err := DB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		logger.Log.Errorw("Failed to set PRAGMA foreign_keys", "error", err)
		return err
	}

	err = DB.AutoMigrate(
		&LocalDownloadsInstance{},
		&LocalDownloadsInstanceItem{},
		&LocalArchivedDownloadsInstance{},
		&LocalArchivedDownloadsInstanceItem{},
	)

	if err != nil {
		logger.Log.Errorw("Failed to auto-migrate database schema", "error", err)
		return err
	}

	logger.Log.Infow("Database initialized successfully", "path", dbPath)
	return nil
}

func AddLocalDownload(download LocalDownloadsInstance) (string, error) {
	dl := download
	logger.Log.Debugw("Adding local download", "protocol", dl.Protocol, "category", dl.Category)
	if err := dl.Add(); err != nil {
		logger.Log.Errorw("Failed to add download to db", "error", err)
		return "", err
	}
	logger.Log.Infow("Download Added", "protocol", dl.Protocol, "cat", dl.Category, "name", dl.DownloadName, "id", dl.IDString())
	return dl.IDString(), nil
}

func AddLocalDownloadItem(localDownloadsInstanceItem LocalDownloadsInstanceItem) (string, error) {
	item := &localDownloadsInstanceItem
	logger.Log.Debugw("Adding local download item", "file_name", item.FileName)
	if err := item.Add(); err != nil {
		logger.Log.Errorw("Failed to add download item to db", "error", err)
		return "", err
	}
	logger.Log.Infow("Download Item Added", "id", item.IDString())
	return item.IDString(), nil
}

func GetLocalDownloadDetails(id string) (LocalDownloadsInstance, error) {
	var download LocalDownloadsInstance
	logger.Log.Debugw("Fetching local download details", "id", id)
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID format", "id", id, "error", err)
		return download, err
	}
	result := DB.Preload("DownloadItems").First(&download, uint(uID))
	if result.Error != nil {
		logger.Log.Errorw("Failed to fetch local download details", "id", id, "error", result.Error)
		return download, result.Error
	}
	if download.DownloadItems == nil {
		download.DownloadItems = []LocalDownloadsInstanceItem{}
	}
	logger.Log.Infow("Successfully fetched local download details", "id", id)
	return download, nil
}

func GetLocalDownloadsByReference(reference string) *LocalDownloadsInstance {
	var download LocalDownloadsInstance
	logger.Log.Debugw("Fetching local download by reference", "reference", reference)
	err := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Where("protocol = ? AND original_download_reference = ?", config.ProtocolTorrent, reference).
		First(&download).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.Debugw("Local download not found by reference", "reference", reference)
			return nil
		}
		logger.Log.Errorw("Database error while fetching by reference", "reference", reference, "error", err)
		return nil
	}
	logger.Log.Infow("Successfully fetched local download by reference", "reference", reference)
	return &download
}

func GetLocalDownloadsByReferences(references string) []LocalDownloadsInstance {
	var downloads []LocalDownloadsInstance
	logger.Log.Debugw("Fetching local downloads by references", "references", references)
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
		return downloads
	}
	logger.Log.Infow("Successfully fetched local downloads by references", "count", len(downloads))
	return downloads
}

func GetLocalDownloadsByFilter(downloadFilters LocalDownloadsInstance) []LocalDownloadsInstance {
	var downloads []LocalDownloadsInstance
	logger.Log.Debugw("Getting downloads by Filter", "downloadFilters", downloadFilters)
	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Model(&downloadFilters)

	errFind := query.Where(&downloadFilters).Find(&downloads).Error
	if errFind != nil {
		logger.Log.Errorw("Database error while fetching by filter", "filters", downloadFilters, "error", errFind)
		return downloads
	}
	logger.Log.Infow("Successfully fetched local downloads by filter", "count", len(downloads))
	return downloads
}

func GetLocalDownloadByFilter(downloadFilters LocalDownloadsInstance) *LocalDownloadsInstance {
	var download LocalDownloadsInstance
	logger.Log.Debugw("Fetching local download by filter", "downloadFilters", downloadFilters)
	err := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Where(&downloadFilters).First(&download).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.Debugw("Local download not found by filter", "downloadFilters", downloadFilters)
			return nil
		}
		logger.Log.Errorw("Database error while fetching by filter", "filters", downloadFilters, "error", err)
		return nil
	}
	logger.Log.Infow("Successfully fetched local download by filter", "id", download.ID)
	return &download
}

func GetLocalDownloadItemDetails(id string) (LocalDownloadsInstanceItem, error) {
	var downloadItem LocalDownloadsInstanceItem
	logger.Log.Debugw("Fetching local download item details", "id", id)

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

	logger.Log.Infow("Successfully fetched local download item details", "id", id)
	return downloadItem, nil
}

func UpdateLocalDownload(localDownload LocalDownloadsInstance) error {
	logger.Log.Debugw("Updating local download", "id", localDownload.ID)
	result := DB.Model(&LocalDownloadsInstance{}).
		Where("id = ?", localDownload.ID).
		Updates(localDownload)
	if result.Error != nil {
		logger.Log.Errorw("Failed to update download provider info", "id", localDownload.ID, "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully updated local download", "id", localDownload.ID)
	return nil
}

func UpdateLocalDownloadItem(localDownloadItem LocalDownloadsInstanceItem) error {
	logger.Log.Debugw("Updating local download item", "id", localDownloadItem.ID)
	result := DB.Model(&LocalDownloadsInstanceItem{}).
		Where("id = ?", localDownloadItem.ID).
		Updates(localDownloadItem)
	if result.Error != nil {
		logger.Log.Errorw("Failed to update download provider info", "id", localDownloadItem.ID, "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully updated local download item", "id", localDownloadItem.ID)
	return nil
}

func UpdateLocalDownloadItemStatus(id string, status string) error {
	logger.Log.Debugw("Updating local download item status", "id", id, "status", status)
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

	logger.Log.Infow("Successfully updated download item status", "Item", id, "Status", status)
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

	logger.Log.Infow("Successfully retrieved downloads", "count", len(downloads))
	return downloads, nil
}

func GetLocalPendingDownloads(protocol string) ([]LocalDownloadsInstance, error) {
	logger.Log.Debugw("Fetching pending downloads from DB", "protocol", protocol)
	var downloads []LocalDownloadsInstance
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
	logger.Log.Infow("Successfully retrieved pending downloads", "protocol", protocol, "count", len(downloads))
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

	logger.Log.Infow("Successfully retrieved completed downloads", "count", len(downloads))
	return downloads, nil
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
	logger.Log.Infow("Successfully retrieved all download items", "count", len(items))
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

	logger.Log.Infow("Successfully retrieved items for download", "download_id", uID, "count", len(items))
	return items, nil
}

func DeleteLocalDownloadItem(id string) error {
	logger.Log.Debugw("Deleting local download item", "id", id)
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
	logger.Log.Infow("Successfully deleted download item", "id", id, "rows_affected", result.RowsAffected)
	return nil
}

func DeleteLocalDownload(id string) error {
	logger.Log.Debugw("Deleting local download", "id", id)
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for download deletion", "id", id, "error", err)
		return err
	}
	errTx := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("download_id = ?", uint(uID)).Delete(&LocalDownloadsInstanceItem{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&LocalDownloadsInstance{}, uint(uID)).Error; err != nil {
			return err
		}
		return nil
	})
	if errTx != nil {
		logger.Log.Errorw("Failed to delete local download and items", "id", id, "error", errTx)
		return errTx
	}
	logger.Log.Infow("Successfully deleted local download and associated items", "id", id)
	return nil
}

type LocalArchivedDownloadsInstance struct {
	ID                         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Protocol                   string    `gorm:"column:protocol" json:"protocol"`
	Provider                   string    `gorm:"column:provider" json:"provider"`
	DownloadName               string    `gorm:"column:download_name" json:"download_name"`
	DownloadType               string    `gorm:"column:download_type:default:'Individual File for Download'" json:"download_type"`
	OriginalDownloadUrl        string    `gorm:"column:original_download_url" json:"original_download_url"`
	OriginalDownloadFile       []byte    `gorm:"column:original_download_file;type:blob" json:"-"`
	OriginalDownloadReference  string    `gorm:"column:original_download_reference;type:blob" json:"original_download_reference"` //hash store
	Category                   string    `gorm:"column:category" json:"category"`
	Status                     string    `gorm:"column:status" json:"status"`
	ExternalProviderID         string    `gorm:"column:external_provider_id" json:"external_provider_id"`
	ExternalProviderDataObject string    `gorm:"column:external_provider_data_object" json:"-"`
	AddedAt                    time.Time `gorm:"column:added_at" json:"added_at"`
	CompletedAt                time.Time `gorm:"column:completed_at" json:"completed_at"`
	Refresh                    bool      `gorm:"column:refresh" json:"refresh"`
	LastRefreshAt              time.Time `gorm:"column:last_refresh_at" json:"last_refresh_at"`

	DownloadItems []LocalArchivedDownloadsInstanceItem `gorm:"foreignKey:DownloadID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"download_items,omitempty"`
}

func (d *LocalArchivedDownloadsInstance) ToJSON() string {
	if d.DownloadItems == nil {
		d.DownloadItems = []LocalArchivedDownloadsInstanceItem{}
	}
	b, _ := json.Marshal(d)
	return string(b)
}

func (d *LocalArchivedDownloadsInstance) Add() error {
	logger.Log.Debugw("Adding LocalArchivedDownloadsInstance to DB", "download_name", d.DownloadName)
	result := DB.Create(d)
	if result.Error != nil {
		logger.Log.Errorw("Failed to add LocalArchivedDownloadsInstance", "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully added LocalArchivedDownloadsInstance", "id", d.ID)
	return nil
}

func (i *LocalArchivedDownloadsInstance) Delete() error {
	logger.Log.Debugw("Deleting LocalArchivedDownloadsInstance", "id", i.ID)
	errTx := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&LocalArchivedDownloadsInstance{}, i.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			logger.Log.Warnw("No LocalArchivedDownloadsInstance found to delete", "id", i.ID)
			return nil
		}
		return nil
	})
	if errTx != nil {
		logger.Log.Errorw("Failed to delete LocalArchivedDownloadsInstance and items", "id", i.ID, "error", errTx)
		return errTx
	}
	logger.Log.Infow("Successfully deleted LocalArchivedDownloadsInstance and associated items", "id", i.ID)
	return nil
}

func (d *LocalArchivedDownloadsInstance) IDString() string {
	return strconv.FormatUint(uint64(d.ID), 10)
}

func LocalArchivedDownloadsInstancesToJSONArray(downloads []LocalArchivedDownloadsInstance) string {
	if downloads == nil {
		downloads = []LocalArchivedDownloadsInstance{}
	}
	for i := range downloads {
		if downloads[i].DownloadItems == nil {
			downloads[i].DownloadItems = []LocalArchivedDownloadsInstanceItem{}
		}
	}
	b, _ := json.Marshal(downloads)
	return string(b)
}

type LocalArchivedDownloadsInstanceItem struct {
	ID                          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DownloadID                  uint      `gorm:"index;column:download_id" json:"download_id"`
	DownloadType                string    `gorm:"column:download_type" json:"download_type"`
	Protocol                    string    `gorm:"column:protocol" json:"protocol"`
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

func (i *LocalArchivedDownloadsInstanceItem) ToJSON() string {
	b, _ := json.Marshal(i)
	return string(b)
}

func (i *LocalArchivedDownloadsInstanceItem) Add() error {
	logger.Log.Debugw("Adding LocalArchivedDownloadsInstanceItem to DB", "download_id", i.DownloadID)
	result := DB.Create(i)
	if result.Error != nil {
		logger.Log.Errorw("Failed to add LocalArchivedDownloadsInstanceItem", "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully added LocalArchivedDownloadsInstanceItem", "id", i.ID)
	return nil
}

func (i *LocalArchivedDownloadsInstanceItem) IDString() string {
	return strconv.FormatUint(uint64(i.ID), 10)
}

func (i *LocalArchivedDownloadsInstanceItem) DownloadIDString() string {
	return strconv.FormatUint(uint64(i.DownloadID), 10)
}

func LocalArchivedDownloadsInstanceItemsToJSONArray(items []LocalArchivedDownloadsInstanceItem) string {
	if items == nil {
		items = []LocalArchivedDownloadsInstanceItem{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func AddArchivedLocalDownload(download LocalArchivedDownloadsInstance) (string, error) {
	dl := download
	logger.Log.Debugw("Adding archived local download", "protocol", dl.Protocol, "category", dl.Category)
	if err := dl.Add(); err != nil {
		logger.Log.Errorw("Failed to add download to db", "error", err)
		return "", err
	}
	logger.Log.Infow("Archive Download Added", "protocol", dl.Protocol, "cat", dl.Category, "name", dl.DownloadName, "id", dl.IDString())
	return dl.IDString(), nil
}

func GetArchivedLocalDownloadsAll() []LocalArchivedDownloadsInstance {
	var downloads []LocalArchivedDownloadsInstance
	logger.Log.Debugw("Fetching all archived local downloads")
	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	})

	errFind := query.Find(&downloads).Error
	if errFind != nil {
		logger.Log.Errorw("Database error while fetching archived downloads", "error", errFind)
		return downloads
	}
	logger.Log.Infow("Successfully fetched all archived downloads", "count", len(downloads))
	return downloads
}

func GetArchivedLocalDownloadsByFilter(downloadFilters LocalArchivedDownloadsInstance) []LocalArchivedDownloadsInstance {
	var downloads []LocalArchivedDownloadsInstance
	logger.Log.Debugw("Getting archived downloads by Filter", "downloadFilters", downloadFilters)
	query := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).Model(&downloadFilters)

	errFind := query.Where(&downloadFilters).Find(&downloads).Error
	if errFind != nil {
		logger.Log.Errorw("Database error while fetching archived downloads by filter", "filters", downloadFilters, "error", errFind)
		return downloads
	}
	logger.Log.Infow("Successfully fetched archived downloads by filter", "count", len(downloads))
	return downloads
}

func GetArchivedLocalDownload(id string) (LocalArchivedDownloadsInstance, error) {
	var download LocalArchivedDownloadsInstance
	logger.Log.Debugw("Fetching archived local download", "id", id)
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID format", "id", id, "error", err)
		return LocalArchivedDownloadsInstance{}, err
	}
	result := DB.Preload("DownloadItems").First(&download, uint(uID))
	if result.Error != nil {
		logger.Log.Errorw("Failed to fetch archived download details", "id", id, "error", result.Error)
		return LocalArchivedDownloadsInstance{}, result.Error
	}
	if download.DownloadItems == nil {
		download.DownloadItems = []LocalArchivedDownloadsInstanceItem{}
	}
	logger.Log.Infow("Successfully fetched archived download details", "id", id)
	return download, nil
}

func DeleteLocalArchivedDownload(id string) error {
	logger.Log.Debugw("Deleting local archived download", "id", id)
	uID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorw("Invalid ID for download deletion", "id", id, "error", err)
		return err
	}
	errTx := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("download_id = ?", uint(uID)).Delete(&LocalArchivedDownloadsInstanceItem{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&LocalArchivedDownloadsInstance{}, uint(uID)).Error; err != nil {
			return err
		}
		return nil
	})
	if errTx != nil {
		logger.Log.Errorw("Failed to delete local archived download and items", "id", id, "error", errTx)
		return errTx
	}
	logger.Log.Infow("Successfully deleted local archived download and associated items", "id", id)
	return nil
}

func UpdateLocalArchivedDownload(localArchivedDownload LocalArchivedDownloadsInstance) error {
	logger.Log.Debugw("Updating local archived download", "id", localArchivedDownload.ID)
	result := DB.Model(&LocalArchivedDownloadsInstance{}).
		Where("id = ?", localArchivedDownload.ID).
		Updates(localArchivedDownload)
	if result.Error != nil {
		logger.Log.Errorw("Failed to update download provider info", "id", localArchivedDownload.ID, "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully updated local archived download", "id", localArchivedDownload.ID, "refresh", localArchivedDownload.Refresh)
	return nil
}

func UpdateLocalArchivedDownloadSelected(localArchivedDownload LocalArchivedDownloadsInstance) error {
	logger.Log.Debugw("Updating selected fields of local archived download", "id", localArchivedDownload.ID)
	result := DB.Model(&LocalArchivedDownloadsInstance{}).
		Where("id = ?", localArchivedDownload.ID).
		Select("refresh", "status", "last_refresh_at").
		Updates(localArchivedDownload)

	if result.Error != nil {
		logger.Log.Errorw("Failed to update selected fields of archived download", "id", localArchivedDownload.ID, "error", result.Error)
		return result.Error
	}
	logger.Log.Infow("Successfully updated selected fields of local archived download", "id", localArchivedDownload.ID)
	return nil
}

func GetArchivedLocalDownloadsOlderThan(olderThan time.Time) []LocalArchivedDownloadsInstance {
	var downloads []LocalArchivedDownloadsInstance
	logger.Log.Debugw("Getting archived downloads older than with refresh enabled", "olderThan", olderThan)

	err := DB.Preload("DownloadItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("file_size DESC")
	}).
		Model(&LocalArchivedDownloadsInstance{}).
		Where("last_refresh_at < ?", olderThan).
		Where("refresh = ?", true).
		Find(&downloads).Error

	if err != nil {
		logger.Log.Errorw("Database error while fetching archived downloads older than time", "olderThan", olderThan, "error", err)
		return downloads
	}

	logger.Log.Infow("Successfully fetched archived downloads older than time", "count", len(downloads))
	return downloads
}
