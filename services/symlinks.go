package services

import (
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"os"
	"path/filepath"
)

func CreateSymlink(localDownloadsInstanceItem databases.LocalDownloadsInstanceItem) {
	var remotePath string
	if localDownloadsInstanceItem.Protocol == config.ProtocolUsenet {
		remotePath = filepath.Join(config.GetUsenetProvider().MountPoint, localDownloadsInstanceItem.FilePath)
	} else {
		remotePath = filepath.Join(config.GetTorrentsProvider().MountPoint, localDownloadsInstanceItem.FilePath)
	}

	symlinkPath := filepath.Join(config.ApplicationDownloadRoot, localDownloadsInstanceItem.Category, localDownloadsInstanceItem.FilePath)
	if _, err := os.Lstat(symlinkPath); err == nil {
		_ = os.Remove(symlinkPath)
	}

	if errDir := os.MkdirAll(filepath.Dir(symlinkPath), 0755); errDir != nil {
		logger.Log.Errorw("Unable to create Symlink Directory", "error", errDir)
		return
	}

	if symError := os.Symlink(remotePath, symlinkPath); symError != nil {
		logger.Log.Errorw("Unable to create Symlink", "error", symError)
		return
	}

	localDownloadsInstanceItem.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	databases.UpdateLocalDownloadItem(localDownloadsInstanceItem)
	logger.Log.Infow("Symlink Created", "remotePath", remotePath, "symlinkPath", symlinkPath)
}
