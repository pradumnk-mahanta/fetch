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

var SupportedDownloaders = []DownloaderInfo{
	{
		ID:   "gdl",
		Name: "Internal Downloader",
	},
	{
		ID:   "symlink",
		Name: "Create Symlink (WIP)",
	},
	{
		ID:   "dnd",
		Name: "Do Not Download (WIP)",
	},
}

var SupportedDebridProviders = []ProviderInfo{
	{
		ID:          "torbox",
		Name:        "Torbox",
		APIEndpoint: "https://api.torbox.app/v1/api",
	},
}

var SupportedUsenetProviders = []ProviderInfo{
	{
		ID:          "torbox",
		Name:        "Torbox",
		APIEndpoint: "https://api.torbox.app/v1/api",
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
	ID   string `json:"downloader_id,omitempty"`
	Name string `json:"downloader_name,omitempty"`
}

type ProviderInfo struct {
	ID             string `json:"debrid_provider_id,omitempty"`
	UsenetID       string `json:"usenet_provider_id,omitempty"`
	Name           string `json:"debrid_provider_name,omitempty"`
	UsenetName     string `json:"usenet_provider_name,omitempty"`
	APIEndpoint    string `json:"debrid_provider_api_endpoint,omitempty"`
	UsenetEndpoint string `json:"usenet_provider_api_endpoint,omitempty"`
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

	cfg.SupportedDownloaders = SupportedDownloaders
	if cfg.ConfiguredDownloaders != nil {
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
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY       = "Retry"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED   = "Completed"
)

const (
	DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE    = "Full Archive for Download"
	DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE = "Individual File for Download"
)
