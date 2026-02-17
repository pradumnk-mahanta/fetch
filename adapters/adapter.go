package adapters

import (
	"context"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/forest6511/gdl"
)

// Change based on Provider Later. Right now only focus on TB
func CreateDownload(protocol string, downloadName string, fileBytes []byte, downloadUrl string, reference string, category string) (string, error) {
	switch protocol {
	case config.ProtocolUsenet:
		var downloadType string = config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE
		if config.GetUsenetProvider().PreferZippedFolder {
			downloadType = config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE
		}
		var download databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:             protocol,
			Provider:             config.GetTorrentsProvider().ID,
			DownloadName:         strings.TrimSuffix(downloadName, filepath.Ext(downloadName)),
			DownloadType:         downloadType,
			OriginalDownloadFile: fileBytes,
			Category:             category,
			Status:               config.DOWNLOAD_STATUS_CLIENT_ADDED,
			AddedAt:              time.Now(),
			DownloadItems:        []databases.LocalDownloadsInstanceItem{},
		}

		locaDownloadId, err := databases.AddLocalDownload(download)
		if err != nil {
			return "", err
		}
		return locaDownloadId, nil

	case config.ProtocolTorrent:
		var downloadType string = config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE
		if config.GetTorrentsProvider().PreferZippedFolder {
			downloadType = config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE
		}
		var download databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:                  protocol,
			Provider:                  config.GetTorrentsProvider().ID,
			DownloadName:              strings.TrimSuffix(downloadName, filepath.Ext(downloadName)),
			DownloadType:              downloadType,
			OriginalDownloadUrl:       downloadUrl,
			OriginalDownloadFile:      fileBytes,
			OriginalDownloadReference: reference,
			Category:                  category,
			Status:                    config.DOWNLOAD_STATUS_CLIENT_ADDED,
			AddedAt:                   time.Now(),
			DownloadItems:             []databases.LocalDownloadsInstanceItem{},
		}

		locaDownloadId, err := databases.AddLocalDownload(download)
		if err != nil {
			return "", err
		}
		return locaDownloadId, nil

	default:
		return "", nil
	}
}

func ProcessUsenetDownloadsQueue() (string, error) {
	logger.Log.Debugw("Processing Usenet Downloads!")

	localDownloads, err := databases.GetLocalPendingDownloads(config.ProtocolUsenet)
	if err != nil {
		logger.Log.Errorw("Unable to retrieve pending downloads at this time. Please try again later!")
		return "Unable to retrieve pending downloads at this time. Please try again later!", err
	}

	var providerDownloads []models.DAT
	if databases.GetLocalDownloadsAddedToProviderCount() > 0 {
		provDl, err := services.TorboxUsenetGetDownloadList()
		if err != nil {
			logger.Log.Errorw("Failed to get download status from provider", "error", err)
			return "Unable to retrieve pending downloads at this time. Please try again later!", err
		}
		providerDownloads = provDl
	}

	for _, localDownload := range localDownloads {
		switch localDownload.Status {
		case config.DOWNLOAD_STATUS_CLIENT_ADDED:
			if databases.GetLocalDownloadsAddedToProviderCount() < config.GetUsenetProvider().MaxSend {
				usenetdownload_id, errAdd := services.TorboxUsenetCreateDownload(localDownload)
				if errAdd != nil {
					logger.Log.Errorw("Failed to add download to provider", "error", errAdd)
					continue
				}

				errorUpdate := databases.UpdateLocalDownloadProviderId(localDownload.IDString(), usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
				if errorUpdate != nil {
					logger.Log.Errorw("Failed to updated local download provider id", "error", errorUpdate)
					continue
				}

				logger.Log.Debugw("Created download on provider", "download", localDownload.ID, "provider", usenetdownload_id)
			}

		case config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING:
			for _, providerDownload := range providerDownloads {
				providerDownloadID := strconv.FormatInt(*providerDownload.ID, 10)
				if providerDownloadID == localDownload.ExternalProviderID {
					errStatus := databases.UpdateLocalDownloadStatus(localDownload.IDString(), services.TranslateTorboxDownloadStatusToLocalStatus(*providerDownload.DownloadState))
					if errStatus != nil {
						logger.Log.Errorw("Failed to update download status", "error", errStatus)
						continue
					}
					provData, provDataErr := providerDownload.ToJSON()
					if provDataErr != nil {
						logger.Log.Errorw("Failed to get provider data in json", "error", provDataErr)
						continue
					}
					errProvData := databases.UpdateLocalDownloadProviderData(localDownload.IDString(), *providerDownload.Name, provData)
					if errProvData != nil {
						logger.Log.Errorw("Failed to update download status", "error", errProvData)
						continue
					}
				}
			}
			logger.Log.Debugw("Updating download", "download", localDownload, "providerStatus", providerDownloads)

		case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
			if localDownload.DownloadType == config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE {
				downloadLink := services.TorboxUsenetRequestDownloadLink(localDownload.ExternalProviderID, "-1", true)
				logger.Log.Debugw("Requested Zipped download link from provider", "downloadLink", downloadLink)

				fileInfo, errFile := gdl.GetFileInfo(context.Background(), downloadLink)
				if errFile != nil {
					logger.Log.Errorw("Unable to retrieve file metadata: " + errFile.Error())
					continue
				}
				downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
					DownloadID:                  localDownload.ID,
					DownloadType:                config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE,
					Category:                    localDownload.Category,
					FileName:                    fileInfo.Filename,
					FilePath:                    GetSanitizedPath(fileInfo.Filename),
					FileSize:                    fileInfo.Size,
					Status:                      config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED,
					ExternalProviderID:          localDownload.ExternalProviderID,
					ExternalProviderItemID:      "-1",
					ExternalProviderDownloadURL: downloadLink,
					RetryCounter:                0,
					AddedAt:                     time.Now(),
				})
				if errAdd != nil {
					logger.Log.Infow("Unable to add download Item Id", "download", localDownload.ID, "item", downloadItemId)
				}
				logger.Log.Infow("Added download Item Id", "item", downloadItemId)
			} else {
				var providerDownloadFromStorage models.DAT
				providerDownloadFromStorage.LoadJSON(localDownload.ExternalProviderDataObject)
				for _, providerDownloadItem := range providerDownloadFromStorage.Files {
					downloadLink := services.TorboxUsenetRequestDownloadLink(localDownload.ExternalProviderID, providerDownloadItem.IDString(), false)
					downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
						DownloadID:                  localDownload.ID,
						DownloadType:                config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE,
						Category:                    localDownload.Category,
						FileName:                    providerDownloadItem.ShortName,
						FilePath:                    GetSanitizedPath(providerDownloadItem.Name),
						FileSize:                    providerDownloadItem.Size,
						Status:                      config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED,
						ExternalProviderID:          localDownload.ExternalProviderID,
						ExternalProviderItemID:      providerDownloadItem.IDString(),
						ExternalProviderDownloadURL: downloadLink,
						RetryCounter:                0,
						AddedAt:                     time.Now(),
					})
					if errAdd != nil {
						logger.Log.Infow("Unable to add download Item Id", "download", localDownload.ID, "provider item", providerDownloadItem.IDString())
					}
					logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)
				}
			}
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_DOWNLOADER_ADDED)

		case config.DOWNLOAD_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING, config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING:
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), GetLocalDownloadStatusBasedOnItems(localDownload.IDString(), config.DOWNLOAD_STATUS_DOWNLOADER_ADDED))

		case config.DOWNLOAD_STATUS_DOWNLOADER_FAILED:
			localDownloadItems, err := databases.GetLocalDownloadItemsForDownload(localDownload.IDString())
			if err != nil {
				logger.Log.Errorw("Unable to get local download items")
				continue
			}

			var hasFailedMoreThanMaxAllowed bool

			for _, item := range localDownloadItems {
				if item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED &&
					item.RetryCounter >= config.AppConfig.DownloaderMaxDownloadsRetry {
					hasFailedMoreThanMaxAllowed = true
					break
				}
			}

			if hasFailedMoreThanMaxAllowed {
				logger.Log.Debugw("Download failed due to no retry attempts left!")
				databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_CLIENT_FAILED)
			}
			continue

		case config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED:
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_CLIENT_COMPLETED)
			continue

		default:
			continue
		}
	}
	return "Successfully updated downloads", nil
}

func ProcessTorrentsDownloadsQueue() (string, error) {
	logger.Log.Debugw("Processing Torrent Downloads!")

	localDownloads, err := databases.GetLocalPendingDownloads(config.ProtocolTorrent)
	if err != nil {
		logger.Log.Errorw("Unable to retrieve pending downloads at this time. Please try again later!")
		return "Unable to retrieve pending downloads at this time. Please try again later!", err
	}

	var providerDownloads []models.DAT
	if databases.GetLocalDownloadsAddedToProviderCount() > 0 {
		provDl, err := services.TorboxTorrentGetDownloadList()
		if err != nil {
			logger.Log.Errorw("Failed to get download status from provider", "error", err)
			return "Unable to retrieve pending downloads at this time. Please try again later!", err
		}
		providerDownloads = provDl
	}

	for _, localDownload := range localDownloads {

		switch localDownload.Status {
		case config.DOWNLOAD_STATUS_CLIENT_ADDED:
			if databases.GetLocalDownloadsAddedToProviderCount() < config.GetTorrentsProvider().MaxSend {
				torrent_id, errAdd := services.TorboxTorrentCreateDownload(localDownload)
				if errAdd != nil {
					logger.Log.Errorw("Failed to add download to provider", "error", errAdd)
					continue
				}

				errorUpdate := databases.UpdateLocalDownloadProviderId(localDownload.IDString(), torrent_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
				if errorUpdate != nil {
					logger.Log.Errorw("Failed to updated local download provider id", "error", errorUpdate)
					continue
				}

				logger.Log.Debugw("Created download on provider", "download", localDownload.ID, "provider", torrent_id)
			}

		case config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING:
			for _, providerDownload := range providerDownloads {
				providerDownloadID := strconv.FormatInt(*providerDownload.ID, 10)
				if providerDownloadID == localDownload.ExternalProviderID {
					errStatus := databases.UpdateLocalDownloadStatus(localDownload.IDString(), services.TranslateTorboxDownloadStatusToLocalStatus(*providerDownload.DownloadState))
					if errStatus != nil {
						logger.Log.Errorw("Failed to update download status", "error", errStatus)
						continue
					}
					provData, provDataErr := providerDownload.ToJSON()
					if provDataErr != nil {
						logger.Log.Errorw("Failed to get provider data in json", "error", provDataErr)
						continue
					}
					errProvData := databases.UpdateLocalDownloadProviderData(localDownload.IDString(), *providerDownload.Name, provData)
					if errProvData != nil {
						logger.Log.Errorw("Failed to update download status", "error", errProvData)
						continue
					}
				}
			}
			logger.Log.Debugw("Updating download", "download", localDownload, "providerStatus", providerDownloads)

		case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
			if localDownload.DownloadType == config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE {
				downloadLink := services.TorboxTorrentRequestDownloadLink(localDownload.ExternalProviderID, "-1", true)
				logger.Log.Debugw("Requested Zipped download link from provider", "downloadLink", downloadLink)

				fileInfo, errFile := gdl.GetFileInfo(context.Background(), downloadLink)
				if errFile != nil {
					logger.Log.Errorw("Unable to retrieve file metadata: " + errFile.Error())
					continue
				}
				downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
					DownloadID:                  localDownload.ID,
					DownloadType:                config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE,
					Category:                    localDownload.Category,
					FileName:                    fileInfo.Filename,
					FilePath:                    fileInfo.Filename,
					FileSize:                    fileInfo.Size,
					Status:                      config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED,
					ExternalProviderID:          localDownload.ExternalProviderID,
					ExternalProviderItemID:      "-1",
					ExternalProviderDownloadURL: downloadLink,
					RetryCounter:                0,
					AddedAt:                     time.Now(),
				})
				if errAdd != nil {
					logger.Log.Infow("Unable to add download Item Id", "download", localDownload.ID, "item", downloadItemId)
				}
				logger.Log.Infow("Added download Item Id", "item", downloadItemId)
			} else {
				var providerDownloadFromStorage models.DAT
				providerDownloadFromStorage.LoadJSON(localDownload.ExternalProviderDataObject)
				for _, providerDownloadItem := range providerDownloadFromStorage.Files {
					downloadLink := services.TorboxTorrentRequestDownloadLink(localDownload.ExternalProviderID, providerDownloadItem.IDString(), false)
					downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
						DownloadID:                  localDownload.ID,
						DownloadType:                config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE,
						Category:                    localDownload.Category,
						FileName:                    providerDownloadItem.ShortName,
						FilePath:                    providerDownloadItem.Name,
						FileSize:                    providerDownloadItem.Size,
						Status:                      config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED,
						ExternalProviderID:          localDownload.ExternalProviderID,
						ExternalProviderItemID:      providerDownloadItem.IDString(),
						ExternalProviderDownloadURL: downloadLink,
						RetryCounter:                0,
						AddedAt:                     time.Now(),
					})
					if errAdd != nil {
						logger.Log.Infow("Unable to add download Item Id", "download", localDownload.ID, "provider item", providerDownloadItem.IDString())
					}
					logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)
				}
			}
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_DOWNLOADER_ADDED)

		case config.DOWNLOAD_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING, config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING:
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), GetLocalDownloadStatusBasedOnItems(localDownload.IDString(), config.DOWNLOAD_STATUS_DOWNLOADER_ADDED))

		case config.DOWNLOAD_STATUS_DOWNLOADER_FAILED:
			localDownloadItems, err := databases.GetLocalDownloadItemsForDownload(localDownload.IDString())
			if err != nil {
				logger.Log.Errorw("Unable to get local download items")
				continue
			}

			var hasFailedMoreThanMaxAllowed bool

			for _, item := range localDownloadItems {
				if item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED &&
					item.RetryCounter >= config.AppConfig.DownloaderMaxDownloadsRetry {
					hasFailedMoreThanMaxAllowed = true
					break
				}
			}

			if hasFailedMoreThanMaxAllowed {
				logger.Log.Debugw("Download failed due to no retry attempts left!")
				databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_CLIENT_FAILED)
			}
			continue

		case config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED:
			databases.UpdateLocalDownloadStatus(localDownload.IDString(), config.DOWNLOAD_STATUS_CLIENT_COMPLETED)
			continue

		default:
			continue
		}
	}
	return "Successfully updated downloads", nil
}

func ProcessDownloadItemsQueue() (string, error) {
	logger.Log.Debugw("Started Processing Downloads Items on Shedule!")

	localDownloadItems, err := databases.GetLocalDownloadItems()
	if err != nil {
		logger.Log.Errorw("Unable to retrieve downloads at this time. Please try again later!")
		return "Unable to retrieve downloads at this time. Please try again later!", err
	}

	for _, localDownloadItem := range localDownloadItems {
		switch localDownloadItem.Status {
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING:
			downloader := services.GetGDLService()
			if !downloader.IsDownloading(localDownloadItem.IDString()) {
				databases.UpdateLocalDownloadItemStatus(localDownloadItem.IDString(), config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED)
			}
			continue

		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY:
			downloader := services.GetGDLService()
			downloads := downloader.Status()
			activeDownloads := 0
			for _, download := range downloads {
				if download.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING {
					activeDownloads++
				}
			}
			if activeDownloads < config.AppConfig.DownloaderMaxDownloadsConcurrent {
				downloader.Download(context.Background(), localDownloadItem)
			}
			continue

		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED:
			if localDownloadItem.RetryCounter < config.AppConfig.DownloaderMaxDownloadsRetry {
				localDownloadItem.RetryCounter++
				databases.UpdateLocalDownloadItemRetryCounter(localDownloadItem.IDString(), localDownloadItem.RetryCounter)
			}
			continue

		default:
			continue
		}
	}
	return "Successfully updated download Items", nil
}

func GetLocalDownloadStatusBasedOnItems(downloadId string, currentStatus string) string {
	localDownloadItems, err := databases.GetLocalDownloadItemsForDownload(downloadId)
	if err != nil {
		logger.Log.Errorw("Unable to get local download items")
		return currentStatus
	}

	var hasAdded, hasDownloading, hasRetry, hasFailed, hasCompleted, hasProcessing bool
	for _, item := range localDownloadItems {
		switch item.Status {
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED:
			hasAdded = true
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING:
			hasDownloading = true
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY:
			hasRetry = true
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED:
			hasFailed = true
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED:
			hasCompleted = true
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING:
			hasProcessing = true
		}
	}

	if hasDownloading || hasRetry || hasAdded {
		return config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING
	} else if hasFailed {
		return config.DOWNLOAD_STATUS_DOWNLOADER_FAILED
	} else if hasProcessing {
		return config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING
	} else if hasCompleted {
		return config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED
	} else {
		return currentStatus
	}
}

func LocalDownloadList() ([]databases.LocalDownloadsInstance, error) {
	downloads, err := databases.GetLocalDownloads()
	if err != nil {
		return nil, err
	}

	return downloads, nil
}

func LocalDownloadUpdateStatus(id string, status string) error {
	err := databases.UpdateLocalDownloadStatus(id, status)
	if err != nil {
		return err
	}
	return nil
}

func DownloaderActiveDownloads() int {
	activeDownloads := 0
	downloader := services.GetGDLService()
	downloads := downloader.Status()
	for _, download := range downloads {
		if download.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING {
			activeDownloads++
		}
	}
	return activeDownloads
}

func DownloaderListStatus() ([]models.GDLDownload, error) {
	downloader := services.GetGDLService()
	downloads := downloader.Status()
	return downloads, nil
}

func DownloaderDeleteDownload(id string) (bool, error) {
	downloader := services.GetGDLService()
	err := downloader.Delete(id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func DownloaderResumeDownload(id string) (bool, error) {
	downloader := services.GetGDLService()
	err := downloader.Resume(context.Background(), id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func GetSanitizedPath(path string) string {
	for strings.Contains(path, "..") {
		path = strings.ReplaceAll(path, "..", ".")
	}
	return path
}
