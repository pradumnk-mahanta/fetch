package models

import (
	"encoding/json"
	"time"
)

type LocalDownloadInstance struct {
	ID                         string                      `json:"id"`
	Protocol                   string                      `json:"protocol"`
	Provider                   string                      `json:"provider"`
	DownloadName               string                      `json:"download_name"`
	OriginalDownloadURL        string                      `json:"original_download_url"`
	OriginalDownloadFile       []byte                      `json:"-"`
	Category                   string                      `json:"category"`
	Status                     string                      `json:"status"`
	ExternalProviderID         string                      `json:"external_provider_id"`
	ExternalProviderDataObject string                      `json:"external_provider_data_object"`
	AddedAt                    time.Time                   `json:"added_at"`
	DownloadItems              []LocalDownloadInstanceItem `json:"download_items"`
}

func (l *LocalDownloadInstance) ToJSON() (string, error) {
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type LocalDownloadInstances []LocalDownloadInstance

func (l LocalDownloadInstances) ToJSON() (string, error) {
	if l == nil {
		l = make(LocalDownloadInstances, 0)
	}
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type LocalDownloadInstanceItem struct {
	ID                          string    `json:"id"`
	DownloadID                  string    `json:"download_id"`
	DownloadType                string    `json:"download_type"`
	FileName                    string    `json:"file_name"`
	FilePath                    string    `json:"file_path"`
	FileSize                    int64     `json:"file_size"`
	Status                      string    `json:"status"`
	ExternalProviderID          string    `json:"external_provider_id"`
	ExternalProviderDownloadURL string    `json:"external_provider_download_url"`
	AddedAt                     time.Time `json:"added_at"`
}
