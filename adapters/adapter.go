package adapters

import (
	"context"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"mime/multipart"
	"slices"
	"strconv"

	"github.com/forest6511/gdl"
)

func CreateDownload(protocol string, downloadName string, fileBytes []byte, downloadUrl string, category string) (string, error) {
	switch protocol {
	case "usenet":
		id, err := databases.AddLocalDownload(protocol, config.AppConfig.AppUsenetDownloadProvider, downloadName, downloadUrl, fileBytes, category)
		if err != nil {
			return "", err
		}
		return id, nil
	default:
		return "", nil
	}
}

func DeleteDownload(id string, downloadName string, downloadFile multipart.File, downloadUrl string, category string) (string, error) {
	return "", nil
}

func ProcessDownloads() (string, error) {
	logger.Log.Debugw("Started Processing Downloads on Shedule!")

	localDownloads, err := databases.GetLocalPendingDownloads()
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
		switch localDownload.Protocol {
		case "usenet":
			switch localDownload.Status {
			case config.DOWNLOAD_STATUS_CLIENT_ADDED:
				if databases.GetLocalDownloadsAddedToProviderCount() < config.AppConfig.AppMaxDownloadSendToProvider {
					usenetdownload_id, errAdd := services.TorboxUsenetCreateDownload(localDownload)
					if errAdd != nil {
						logger.Log.Errorw("Failed to add download to provider", "error", errAdd)
						continue
					}

					errorUpdate := databases.UpdateLocalDownloadProviderId(localDownload.ID, usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
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
						errStatus := databases.UpdateLocalDownloadStatus(localDownload.ID, services.TorboxUsenetDownloadStatusTranslate(*providerDownload.DownloadState))
						if errStatus != nil {
							logger.Log.Errorw("Failed to update download status", "error", errStatus)
							continue
						}
						provData, provDataErr := providerDownload.ToJSON()
						if provDataErr != nil {
							logger.Log.Errorw("Failed to get provider data in json", "error", provDataErr)
							continue
						}
						errProvData := databases.UpdateLocalDownloadProviderData(localDownload.ID, provData)
						if errProvData != nil {
							logger.Log.Errorw("Failed to update download status", "error", errProvData)
							continue
						}
					}
				}
				logger.Log.Debugw("Updating download", "download", localDownload, "providerStatus", providerDownloads)

			case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
				if config.AppConfig.ProviderTBPreferZippedFolder {
					downloadLink, requestDownloadLinkError := services.TorboxUsenetRequestDownloadLink(localDownload.ExternalProviderID, "-1")
					if requestDownloadLinkError != nil {
						logger.Log.Errorw("Failed to request download link from provider", "error", requestDownloadLinkError)
						continue
					}
					logger.Log.Debugw("Requested Zipped download link from provider", "downloadLink", downloadLink)

					fileInfo, err := gdl.GetFileInfo(context.Background(), downloadLink)
					if err != nil {
						logger.Log.Errorw("Unable to retrieve file metadata: " + err.Error())
						continue
					}
					downloadItemId := databases.AddLocalDownloadItem(localDownload.ID, config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE, localDownload.Category, fileInfo.Filename, fileInfo.Filename, fileInfo.Size, localDownload.ExternalProviderID, "-1", downloadLink)
					logger.Log.Infow("Added download Item Id", "item", downloadItemId)
				} else {
					var providerDownloadFromStorage models.DAT
					providerDownloadFromStorage.LoadJSON(localDownload.ExternalProviderDataObject)
					for _, providerDownloadItem := range providerDownloadFromStorage.Files {
						downloadItemId := databases.AddLocalDownloadItem(localDownload.ID, config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE, localDownload.Category, providerDownloadItem.ShortName, providerDownloadItem.Name, providerDownloadItem.Size, localDownload.ExternalProviderID, strconv.FormatInt(providerDownloadItem.ID, 10), "")
						logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)
					}
				}
				databases.UpdateLocalDownloadStatus(localDownload.ID, config.DOWNLOAD_STATUS_DOWNLOADER_ADDED)

			case config.DOWNLOAD_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING, config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING:
				databases.UpdateLocalDownloadStatus(localDownload.ID, GetLocalDownloadStatusBasedOnItems(localDownload.ID, config.DOWNLOAD_STATUS_DOWNLOADER_ADDED))

			case config.DOWNLOAD_STATUS_DOWNLOADER_FAILED:
				localDownloadItems, err := databases.GetLocalDownloadItemsForDownload(localDownload.ID)
				if err != nil {
					logger.Log.Errorw("Unable to get local download items")
					continue
				}
				hasFailedMoreThanMaxAllowed := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
					return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED && item.RetryCounter >= config.AppConfig.DownloaderMaxRetryDownloads
				})

				if hasFailedMoreThanMaxAllowed {
					logger.Log.Debugw("Download failed due to no retry attempts left!")
					databases.UpdateLocalDownloadStatus(localDownload.ID, config.DOWNLOAD_STATUS_CLIENT_FAILED)
				}
				continue

			case config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED:
				databases.UpdateLocalDownloadStatus(localDownload.ID, config.DOWNLOAD_STATUS_CLIENT_COMPLETED)
				continue

			default:
				continue
			}
		default:
			continue
		}
	}
	return "Successfully updated downloads", nil
}

func ProcessDownloadItems() (string, error) {
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
			if !downloader.IsDownloading(localDownloadItem.ID) {
				databases.UpdateLocalDownloadItemStatus(localDownloadItem.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED)
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
			if activeDownloads < config.AppConfig.DownloaderMaxParallelDownloads {
				var downloadLink string
				if localDownloadItem.ExternalProviderDownloadURL == "" {
					providerDownloadLink, requestDownloadLinkError := services.TorboxUsenetRequestDownloadLink(localDownloadItem.ExternalProviderID, localDownloadItem.ExternalProviderItemID)
					if requestDownloadLinkError != nil {
						logger.Log.Errorw("Failed to request download link from provider", "error", requestDownloadLinkError)
						continue
					}
					linkDbUpdErr := databases.UpdateLocalDownloadItemExternalUrl(localDownloadItem.ID, downloadLink)
					if linkDbUpdErr != nil {
						logger.Log.Errorw("Failed to save download link from provider", "error", linkDbUpdErr)
						continue
					}
					downloadLink = providerDownloadLink
				} else {
					downloadLink = localDownloadItem.ExternalProviderDownloadURL
				}
				localDownloadItem.ExternalProviderDownloadURL = downloadLink
				downloader.Download(context.Background(), localDownloadItem)
			}
			continue

		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED:
			if localDownloadItem.RetryCounter < config.AppConfig.DownloaderMaxRetryDownloads {
				localDownloadItem.RetryCounter++
				databases.UpdateLocalDownloadItemRetryCounter(localDownloadItem.ID, localDownloadItem.RetryCounter)
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
	hasAdded := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED
	})
	hasDownloading := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING
	})
	hasRetry := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY
	})
	hasFailed := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED
	})
	hasCompleted := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	})
	hasProcessing := slices.ContainsFunc(localDownloadItems, func(item models.LocalDownloadInstanceItem) bool {
		return item.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING
	})
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

func LocalDownloadList() ([]models.LocalDownloadInstance, error) {
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
