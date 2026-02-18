package services

import (
	"fetch/config"
	"fetch/databases"
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
	os.Symlink(remotePath, symlinkPath)

	localDownloadsInstanceItem.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	databases.UpdateLocalDownloadItem(localDownloadsInstanceItem)
}
