package services

import (
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"os"
	"path/filepath"
)

func CreateStrmlink(localDownloadsInstanceItem databases.LocalDownloadsInstanceItem) {
	basePath := filepath.Join(config.ApplicationDownloadRoot, localDownloadsInstanceItem.Category, localDownloadsInstanceItem.FilePath)
	strmPath := basePath + ".strm"

	if errDir := os.MkdirAll(filepath.Dir(strmPath), 0755); errDir != nil {
		logger.Log.Errorw("Unable to create STRM Directory", "error", errDir)
		return
	}

	err := os.WriteFile(strmPath, []byte(localDownloadsInstanceItem.ExternalProviderDownloadURL), 0644)
	if err != nil {
		logger.Log.Errorw("Unable to write STRM file", "path", strmPath, "error", err)
		return
	}

	localDownloadsInstanceItem.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	databases.UpdateLocalDownloadItem(localDownloadsInstanceItem)
	logger.Log.Infow("Strm Created", "strmPath", strmPath)
}
