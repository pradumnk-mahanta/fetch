package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var AppConfig *Config

const configPath = "/data/config.json"

type Config struct {
	AppLogLevel                    string `json:"APPLICATION_LOG_LEVEL"`
	AppAuthUsername                string `json:"APPLICATION_AUTH_USERNAME"`
	AppAuthPassword                string `json:"APPLICATION_AUTH_PASSWORD"`
	AppUsenetDownloadProvider      string `json:"APPLICATION_USENET_DOWNLOAD_PROVIDER"`
	AppMaxDownloadSendToProvider   int    `json:"APPLICATION_MAX_DOWNLOAD_SEND_TO_PROVIDER"`
	SabCategories                  string `json:"SABNZBD_CATEGORIES"`
	DownloaderMaxParallelDownloads int    `json:"DOWNLOADER_MAX_PARALLEL_DOWNLOADS"`
	DownloaderMaxRetryDownloads    int    `json:"DOWNLOADER_MAX_RETRY_DOWNLOADS"`
	ProviderTBAPIKey               string `json:"PROVIDER_TB_CONFIG_API_KEY"`
	ProviderTBPreferZippedFolder   bool   `json:"PROVIDER_TB_CONFIG_PREFER_ZIPPED_FOLDER"`
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

	AppConfig = &cfg
	return nil
}

func SaveConfig() error {
	data, err := json.MarshalIndent(AppConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func CreateDefaultConfig() error {
	//Default Values
	AppConfig = &Config{
		AppLogLevel:                    "INFO",
		AppAuthUsername:                "",
		AppAuthPassword:                "",
		AppUsenetDownloadProvider:      "",
		AppMaxDownloadSendToProvider:   2,
		SabCategories:                  "sonarr,radarr",
		DownloaderMaxParallelDownloads: 2,
		DownloaderMaxRetryDownloads:    2,
		ProviderTBAPIKey:               "",
		ProviderTBPreferZippedFolder:   false,
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(AppConfig, "", "    ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write default config: %w", err)
	}

	fmt.Printf("Default config created at %s, please add the respective required details and restart the application", configPath)
	return ReadConfig()
}

const (
	TB_API_BASE_URL           = "https://api.torbox.app/v1/api"
	APPLICATION_DOWNLOAD_ROOT = "/downloads"
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
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY       = "Retry"
	DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED   = "Completed"
)

const (
	DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE    = "Full Archive for Download"
	DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE = "Individual File for Download"
)

func GenerateSessionHash() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
