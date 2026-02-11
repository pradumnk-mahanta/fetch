package models

import "time"

type LocalDownloadInstance struct {
	ID                   string    `json:"id"`
	Protocol             string    `json:"protocol"`
	Provider             string    `json:"provider"`
	DownloadName         string    `json:"download_name"`
	OriginalDownloadURL  string    `json:"original_download_url"`
	OriginalDownloadFile []byte    `json:"original_download_file"`
	Category             string    `json:"category"`
	Status               string    `json:"status"`
	ExternalIDProvider   string    `json:"external_id_provider"`
	ExternalIDDownloader string    `json:"external_id_downloader"`
	AddedAt              time.Time `json:"added_at"`
}
