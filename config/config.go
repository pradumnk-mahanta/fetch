package config

import (
	"os"
	"strconv"
)

type EnvKey string

func (key EnvKey) GetValue() string {
	return os.Getenv(string(key))
}

func (key EnvKey) GetBoolValue() bool {
	return os.Getenv(string(key)) == "true"
}

func (key EnvKey) GetIntValue() int {
	var intValue, error = strconv.Atoi(os.Getenv(string(key)))
	if error != nil {
		return 1
	}
	return intValue
}

const (
	APPLICATION_API_PORT                 EnvKey = "APPLICATION_API_PORT"
	APPLICATION_DOWNLOAD_ROOT            EnvKey = "APPLICATION_DOWNLOAD_ROOT"
	APPLICATION_USENET_DOWNLOAD_PROVIDER EnvKey = "APPLICATION_USENET_DOWNLOAD_PROVIDER"
	SABNZBD_API_KEY                      EnvKey = "SABNZBD_API_KEY"
	QBITTORRENT_USERNAME                 EnvKey = "QBITTORRENT_USERNAME"
	QBITTORRENT_PASSWORD                 EnvKey = "QBITTORRENT_PASSWORD"
	TB_CONFIG_API_KEY                    EnvKey = "TB_CONFIG_API_KEY"
	TB_CONFIG_PREFER_ZIPPED_FOLDER       EnvKey = "TB_CONFIG_PREFER_ZIPPED_FOLDER"
	TORRENTS_DOWNLOAD_PROVIDER           EnvKey = "TORRENTS_DOWNLOAD_PROVIDER"
	DOWNLOADER_MAX_PARALLEL_DOWNLOADS    EnvKey = "DOWNLOADER_MAX_PARALLEL_DOWNLOADS"
)

const (
	TB_API_BASE_URL = "https://api.torbox.app/v1/api"
)

const (
	DOWNLOAD_STATUS_CLIENT_ADDED           = "Added"
	DOWNLOAD_STATUS_CLIENT_PROCESSING      = "Processing"
	DOWNLOAD_STATUS_CLIENT_DOWNLOADING     = "Downloading"
	DOWNLOAD_STATUS_CLIENT_FAILED          = "Failed"
	DOWNLOAD_STATUS_CLIENT_COMPLETED       = "Completed"
	DOWNLOAD_STATUS_PROVIDER_ADDED         = "Added to Provider"
	DOWNLOAD_STATUS_PROVIDER_DOWNLOADING   = "Downloading on Provider"
	DOWNLOAD_STATUS_PROVIDER_PROCESSING    = "Processing on Provider"
	DOWNLOAD_STATUS_PROVIDER_FAILED        = "Failed on Provider"
	DOWNLOAD_STATUS_PROVIDER_COMPLETED     = "Completed on Provider"
	DOWNLOAD_STATUS_DOWNLOADER_ADDED       = "Added to Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING = "Downloading on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_PROCESSING  = "Processing on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_PAUSED      = "Paused on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_FAILED      = "Failed on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_COMPLETED   = "Completed on Downloader"
)

const (
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED       = "Added"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING  = "Processing"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING = "Downloading"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_PAUSED      = "Paused"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED      = "Failed"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED   = "Completed"
)

const (
	DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE    = "Full Archive for Download"
	DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE = "Individual File for Download"
)
