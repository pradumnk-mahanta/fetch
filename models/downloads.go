package models

import (
	"encoding/json"
	"fetch/databases"
	"fetch/logger"
)

type CombinedDownloadDetails struct {
	databases.LocalDownloadsInstance
	ItemDetails []CombinedItemDetails `json:"item_details"`
}

type CombinedItemDetails struct {
	databases.LocalDownloadsInstanceItem
	DownloaderStats *GDLDownload `json:"downloader_stats,omitempty"`
}

func GetCombinedDownloadDetails(liveDownloads []GDLDownload) ([]CombinedDownloadDetails, error) {
	localDownloads, err := databases.GetLocalDownloads()
	if err != nil {
		logger.Log.Errorw("Failed to fetch local downloads for combination", "error", err)
		return nil, err
	}

	liveMap := make(map[string]GDLDownload)
	for _, live := range liveDownloads {
		liveMap[live.ID] = live
	}

	var combined []CombinedDownloadDetails
	for _, dl := range localDownloads {
		combo := CombinedDownloadDetails{
			LocalDownloadsInstance: dl,
			ItemDetails:            []CombinedItemDetails{},
		}

		for _, item := range dl.DownloadItems {
			itemCombo := CombinedItemDetails{
				LocalDownloadsInstanceItem: item,
			}

			if stats, exists := liveMap[item.IDString()]; exists {
				itemCombo.DownloaderStats = &stats
			}
			combo.ItemDetails = append(combo.ItemDetails, itemCombo)
		}
		combined = append(combined, combo)
	}

	logger.Log.Debugw("Combined details generated", "total_downloads", len(combined))
	return combined, nil
}

func (c *CombinedDownloadDetails) ToJson() string {
	if c.ItemDetails == nil {
		c.ItemDetails = []CombinedItemDetails{}
	}
	if c.DownloadItems == nil {
		c.DownloadItems = []databases.LocalDownloadsInstanceItem{}
	}

	b, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func CombinedDownloadsToJSONArray(details []CombinedDownloadDetails) string {
	if details == nil {
		details = []CombinedDownloadDetails{}
	}

	for i := range details {
		if details[i].ItemDetails == nil {
			details[i].ItemDetails = []CombinedItemDetails{}
		}
		if details[i].DownloadItems == nil {
			details[i].DownloadItems = []databases.LocalDownloadsInstanceItem{}
		}
	}

	b, err := json.Marshal(details)
	if err != nil {
		return "[]"
	}
	return string(b)
}
