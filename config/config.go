package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var AppConfig *Config

var version = "dev"

const configPath = "/data/config.json"
const (
	ApplicationDownloadRoot = "/downloads"
	ProtocolTorrent         = "torrent"
	ProtocolUsenet          = "usenet"
)

const (
	DownloaderIdInternal      = "internal"
	DownloaderIdSymlink       = "symlink"
	DownloaderIdDoNotDownload = "donotdownload"
	DownloaderIdStrmLink      = "strmlink"
)

var SupportedDownloaders = []DownloaderInfo{
	{
		ID:          DownloaderIdInternal,
		Name:        "Internal Downloader",
		Description: "All the files will be downloaded to host.",
	},
	{
		ID:          DownloaderIdSymlink,
		Name:        "Create Symlink",
		Description: "All the files will be symlinked to the provided mount path. Keep the mountpath in all the applications same. If mounting to /mnt/provoder in fetch, mount at the same location in Emby/Jellyfin/Plex. Please be advised, using this method relies on the media being available on the provider. If it is deleted from the Provider (goes out of cache), links will not resolve to anything. ",
	},
	{
		ID:          DownloaderIdDoNotDownload,
		Name:        "Do Not Download",
		Description: "Downloads will not be downloaded to host.",
	},
	{
		ID:          DownloaderIdStrmLink,
		Name:        "Create Strm Files",
		Description: "Strm Files are created for the downloads. Not dependent on mount paths. Please be advised, using this method relies on the media being available on the provider. If it is deleted from the Provider (goes out of cache), links will not resolve to anything. This method relies on your API Key to remain consistent. In case you change your API Keys, you will have to change the api keys in all your strm files untill an automated process is created.",
	},
}

var SupportedDebridProviders = []ProviderInfo{
	{
		DebridID:          "torbox",
		DebridName:        "Torbox",
		DebridAPIEndpoint: "https://api.torbox.app/v1/api",
	},
}

var SupportedUsenetProviders = []ProviderInfo{
	{
		UsenetID:          "torbox",
		UsenetName:        "Torbox",
		UsenetAPIEndpoint: "https://api.torbox.app/v1/api",
	},
}

type Config struct {
	Version                          string           `json:"version"`
	ApplicationLogLevel              string           `json:"application_log_level"`
	ApplicationSupportedLogLevel     []string         `json:"application_supported_log_level"`
	ApplicationAuthUsername          string           `json:"application_auth_username"`
	ApplicationAuthPassword          string           `json:"application_auth_password"`
	ApplicationCategories            string           `json:"application_categories"`
	DownloaderMaxDownloadsConcurrent int              `json:"downloader_max_downloads_concurrent"`
	DownloaderMaxDownloadsRetry      int              `json:"downloader_max_downloads_retry"`
	ConfiguredDownloaders            *DownloaderInfo  `json:"configured_downloaders"`
	SupportedDownloaders             []DownloaderInfo `json:"supported_downloaders"`
	SupportedDebridProviders         []ProviderInfo   `json:"supported_debrid_providers"`
	ConfiguredDebridProviders        []DebridConfig   `json:"configured_debrid_providers"`
	SupportedUsenetProviders         []ProviderInfo   `json:"supported_usenet_providers"`
	ConfiguredUsenetProviders        []UsenetConfig   `json:"configured_usenet_providers"`
}

type DownloaderInfo struct {
	ID          string `json:"downloader_id,omitempty"`
	Name        string `json:"downloader_name,omitempty"`
	Description string `json:"downloader_description,omitempty"`
}

type ProviderInfo struct {
	DebridID          string `json:"debrid_provider_id,omitempty"`
	UsenetID          string `json:"usenet_provider_id,omitempty"`
	DebridName        string `json:"debrid_provider_name,omitempty"`
	UsenetName        string `json:"usenet_provider_name,omitempty"`
	DebridAPIEndpoint string `json:"debrid_provider_api_endpoint,omitempty"`
	UsenetAPIEndpoint string `json:"usenet_provider_api_endpoint,omitempty"`
}

type DebridConfig struct {
	Priority           int    `json:"priority"`
	ID                 string `json:"debrid_provider_id"`
	Name               string `json:"debrid_provider_name"`
	APIEndpoint        string `json:"debrid_provider_api_endpoint"`
	APIKey             string `json:"debrid_provider_api_key"`
	PreferZippedFolder bool   `json:"debrid_provider_prefer_zipped_folder"`
	MaxSend            int    `json:"debrid_provider_max_send"`
	MountPoint         string `json:"usenet_provider_rclone_mount"`
}

type UsenetConfig struct {
	Priority           int    `json:"priority"`
	ID                 string `json:"usenet_provider_id"`
	Name               string `json:"usenet_provider_name"`
	APIEndpoint        string `json:"usenet_provider_api_endpoint"`
	APIKey             string `json:"usenet_provider_api_key"`
	PreferZippedFolder bool   `json:"usenet_provider_prefer_zipped_folder"`
	MaxSend            int    `json:"usenet_provider_max_send"`
	MountPoint         string `json:"usenet_provider_rclone_mount"`
}

func LoadConfig() error {
	if _, err := os.Stat(configPath); os.IsExist(err) || err == nil {
		return ReadConfig()
	}
	return CreateDefaultConfig()
}

func ReadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("Failed to parse config JSON: %w", err)
	}

	newVersion := GetVersion()
	if cfg.Version != newVersion {
		fmt.Printf("Updating app version from %s to %s\n", cfg.Version, newVersion)
		cfg.Version = newVersion
	}

	cfg.SupportedDebridProviders = SupportedDebridProviders
	cfg.SupportedUsenetProviders = SupportedUsenetProviders
	cfg.SupportedDownloaders = SupportedDownloaders
	if cfg.ConfiguredDownloaders == nil {
		cfg.ConfiguredDownloaders = &cfg.SupportedDownloaders[0]
	}

	AppConfig = &cfg
	SaveConfig()
	return nil
}

func SaveConfig() error {
	data, err := json.MarshalIndent(AppConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to marshal config to JSON: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write config file: %w", err)
	}

	return nil
}

func CreateDefaultConfig() error {
	AppConfig = &Config{
		Version:                          GetVersion(),
		ApplicationSupportedLogLevel:     []string{"INFO", "DEBUG"},
		ApplicationLogLevel:              "INFO",
		ApplicationAuthUsername:          "",
		ApplicationAuthPassword:          "",
		ApplicationCategories:            "sonarr,radarr",
		SupportedDownloaders:             SupportedDownloaders,
		ConfiguredDownloaders:            &SupportedDownloaders[0],
		DownloaderMaxDownloadsConcurrent: 2,
		DownloaderMaxDownloadsRetry:      2,
		SupportedDebridProviders:         SupportedDebridProviders,
		ConfiguredDebridProviders:        []DebridConfig{},
		SupportedUsenetProviders:         SupportedUsenetProviders,
		ConfiguredUsenetProviders:        []UsenetConfig{},
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(AppConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	fmt.Printf("New format config created at %s. Please update your details and restart.\n", configPath)

	return ReadConfig()
}

func GetVersion() string {
	return version
}

func GetUsenetProvider() UsenetConfig {
	if len(AppConfig.ConfiguredUsenetProviders) == 0 {
		return UsenetConfig{}
	} else {
		return AppConfig.ConfiguredUsenetProviders[0]
	}
}

func GetTorrentsProvider() DebridConfig {
	if len(AppConfig.ConfiguredDebridProviders) == 0 {
		return DebridConfig{}
	} else {
		return AppConfig.ConfiguredDebridProviders[0]
	}
}

func GetMaxSendToProvider() int {
	var usenetMax int = 1
	var torrentsMax int = 1
	if len(AppConfig.ConfiguredUsenetProviders) > 0 {
		usenetMax = AppConfig.ConfiguredUsenetProviders[0].MaxSend
	}
	if len(AppConfig.ConfiguredDebridProviders) > 0 {
		torrentsMax = AppConfig.ConfiguredDebridProviders[0].MaxSend
	}
	return min(usenetMax, torrentsMax)
}

const (
	DOWNLOAD_STATUS_CLIENT_ADDED                    = "Added"
	DOWNLOAD_STATUS_CLIENT_PROCESSING               = "Processing"
	DOWNLOAD_STATUS_CLIENT_DOWNLOADING              = "Downloading"
	DOWNLOAD_STATUS_CLIENT_FAILED                   = "Failed"
	DOWNLOAD_STATUS_CLIENT_COMPLETED                = "Completed"
	DOWNLOAD_STATUS_CLIENT_COMPLETED_NOT_DOWNLOADED = "Completed, Not Downloaded"
	DOWNLOAD_STATUS_PROVIDER_ADDED                  = "Added to Provider"
	DOWNLOAD_STATUS_PROVIDER_DOWNLOADING            = "Downloading on Provider"
	DOWNLOAD_STATUS_PROVIDER_PROCESSING             = "Processing on Provider"
	DOWNLOAD_STATUS_PROVIDER_FAILED                 = "Failed on Provider"
	DOWNLOAD_STATUS_PROVIDER_COMPLETED              = "Completed on Provider"
	DOWNLOAD_STATUS_DOWNLOADER_ADDED                = "Added to Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING          = "Downloading on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_PROCESSING           = "Processing on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_PAUSED               = "Paused on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_FAILED               = "Failed on Downloader"
	DOWNLOAD_STATUS_DOWNLOADER_COMPLETED            = "Completed on Downloader"
)

const (
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED       = "Added"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING  = "Processing"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING = "Downloading"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_PAUSED      = "Paused"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED      = "Failed"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY       = "Retry"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED   = "Completed"
)

const (
	DOWNLOAD_TYPE_FULL_ARCHIVE    = "Full Archive for Download"
	DOWNLOAD_TYPE_INDIVIDUAL_FILE = "Individual File for Download"
	DOWNLOAD_TYPE_CREATE_SYMLINK  = "Individual File Create Symlink"
	DOWNLOAD_TYPE_CREATE_STRM     = "Individual File Create Strm"
	DOWNLOAD_TYPE_DO_NOT_DOWNLOAD = "Do not Download Files"
)
