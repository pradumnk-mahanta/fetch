package config

import "os"

type EnvKey string

func (key EnvKey) GetValue() string {
	return os.Getenv(string(key))
}

const (
	APPLICATION_API_PORT       EnvKey = "APPLICATION_API_PORT"
	SABNZBD_API_KEY            EnvKey = "SABNZBD_API_KEY"
	QBITTORRENT_USERNAME       EnvKey = "QBITTORRENT_USERNAME"
	QBITTORRENT_PASSWORD       EnvKey = "QBITTORRENT_PASSWORD"
	JD_EMAIL                   EnvKey = "JD_EMAIL"
	JD_PASSWORD                EnvKey = "JD_PASSWORD"
	JD_DEVICE_NAME             EnvKey = "JD_DEVICE_NAME"
	JD_DOWNLOAD_ROOT           EnvKey = "JD_DOWNLOAD_ROOT"
	TB_API_KEY                 EnvKey = "TB_API_KEY"
	TORRENTS_DOWNLOAD_PROVIDER EnvKey = "TORRENTS_DOWNLOAD_PROVIDER"
	USENET_DOWNLOAD_PROVIDER   EnvKey = "USENET_DOWNLOAD_PROVIDER"
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
