package models

import "encoding/json"

type GDLDownload struct {
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	OutputPath      string  `json:"output_path"`
	Status          string  `json:"status"`
	BytesDownloaded int64   `json:"bytes_downloaded"`
	TotalSize       int64   `json:"total_size,omitempty"`
	Percentage      float64 `json:"percentage"`
	AverageSpeed    string  `json:"average_speed,omitempty"`
}

func (l *GDLDownload) ToJSON() (string, error) {
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type GDLDownloads []GDLDownload

func (l GDLDownloads) ToJSON() (string, error) {
	if l == nil {
		l = make(GDLDownloads, 0)
	}
	bytes, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
