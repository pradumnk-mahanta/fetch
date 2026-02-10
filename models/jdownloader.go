package models

type DownloadQueryLinksParams struct {
	BytesTotal       bool    `json:"bytesTotal,omitempty"`
	Comment          bool    `json:"comment,omitempty"`
	Status           bool    `json:"status,omitempty"`
	Enabled          bool    `json:"enabled,omitempty"`
	MaxResults       int     `json:"maxResults,omitempty"`
	StartAt          int     `json:"startAt,omitempty"`
	Host             bool    `json:"host,omitempty"`
	BytesLoaded      bool    `json:"bytesLoaded,omitempty"`
	Speed            bool    `json:"speed,omitempty"`
	Eta              bool    `json:"eta,omitempty"`
	Finished         bool    `json:"finished,omitempty"`
	FinishedDate     bool    `json:"finishedDate,omitempty"`
	Running          bool    `json:"running,omitempty"`
	Skipped          bool    `json:"skipped,omitempty"`
	ExtractionStatus bool    `json:"extractionStatus,omitempty"`
	PackageUUIDs     []int64 `json:"packageUUIDs"`
	Url              bool    `json:"url"`
	Priority         bool    `json:"priority"`
}
