package models

import (
	"encoding/json"
	"time"
)

type LocalDownloadInstance struct {
	ID                          string    `json:"id"`
	Protocol                    string    `json:"protocol"`
	Provider                    string    `json:"provider"`
	DownloadName                string    `json:"download_name"`
	OriginalDownloadURL         string    `json:"original_download_url"`
	OriginalDownloadFile        []byte    `json:"-"`
	Category                    string    `json:"category"`
	Status                      string    `json:"status"`
	ExternalIDProvider          string    `json:"external_id_provider"`
	ExternalIDDownloader        string    `json:"external_id_downloader"`
	ExternalDownloadProviderURL string    `json:"external_download_provider_url"`
	AddedAt                     time.Time `json:"added_at"`
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
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
