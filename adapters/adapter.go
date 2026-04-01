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
	var downloadType string
	switch config.AppConfig.ConfiguredDownloaders.ID {
	case config.DownloaderIdDoNotDownload:
		downloadType = config.DOWNLOAD_TYPE_DO_NOT_DOWNLOAD
	case config.DownloaderIdSymlink:
		downloadType = config.DOWNLOAD_TYPE_CREATE_SYMLINK
	case config.DownloaderIdStrmLink:
		downloadType = config.DOWNLOAD_TYPE_CREATE_STRM
	case config.DownloaderIdInternal:
		if protocol == config.ProtocolUsenet {
			if config.GetUsenetProvider().PreferZippedFolder {
				downloadType = config.DOWNLOAD_TYPE_FULL_ARCHIVE
			} else {
				downloadType = config.DOWNLOAD_TYPE_INDIVIDUAL_FILE
			}
		} else {
			if config.GetTorrentsProvider().PreferZippedFolder {
				downloadType = config.DOWNLOAD_TYPE_FULL_ARCHIVE
			} else {
				downloadType = config.DOWNLOAD_TYPE_INDIVIDUAL_FILE
			}
		}
	default:
		downloadType = config.DOWNLOAD_TYPE_DO_NOT_DOWNLOAD
	}

	switch protocol {
	case config.ProtocolUsenet:
		var download databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:             protocol,
			Provider:             config.GetUsenetProvider().ID,
			DownloadName:         GetSanitizedPath(strings.TrimSuffix(downloadName, filepath.Ext(downloadName))),
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
		var download databases.LocalDownloadsInstance = databases.LocalDownloadsInstance{
			Protocol:                  protocol,
			Provider:                  config.GetTorrentsProvider().ID,
			DownloadName:              GetSanitizedPath(strings.TrimSuffix(downloadName, filepath.Ext(downloadName))),
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

func ProcessDownloadsQueue() (string, error) {
	logger.Log.Debugw("Processing Downloads!")
	localDownloads, err := databases.GetLocalPendingDownloads("")
	if err != nil {
		logger.Log.Errorw("Unable to retrieve pending downloads at this time. Please try again later!")
		return "Unable to retrieve pending downloads at this time. Please try again later!", err
	}

	var usernetDownloadsCount int = 0
	var torrentsDownloadsCount int = 0
	for _, ld := range localDownloads {
		if ld.Status == config.DOWNLOAD_STATUS_PROVIDER_ADDED || ld.Status == config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING || ld.Status == config.DOWNLOAD_STATUS_PROVIDER_PROCESSING {
			if ld.Protocol == config.ProtocolTorrent {
				torrentsDownloadsCount++
			} else {
				usernetDownloadsCount++
			}
		}
	}

	var providerDownloads []models.DAT
	if usernetDownloadsCount > 0 {
		provDl, err := services.TorboxUsenetGetDownloadList()
		if err != nil {
			logger.Log.Errorw("Failed to get usenet downloads list from provider", "error", err)
			return "Unable to retrieve pending downloads at this time. Please try again later!", err
		}
		for _, dl := range provDl {
			providerDownloads = append(providerDownloads, dl)
		}
	}

	if torrentsDownloadsCount > 0 {
		provDl, err := services.TorboxTorrentGetDownloadList()
		if err != nil {
			logger.Log.Errorw("Failed to get torrents downloads list from provider", "error", err)
			return "Unable to retrieve pending downloads at this time. Please try again later!", err
		}
		for _, dl := range provDl {
			providerDownloads = append(providerDownloads, dl)
		}
	}

	for _, localDownload := range localDownloads {
		switch localDownload.Status {
		case config.DOWNLOAD_STATUS_CLIENT_ADDED:
			if (usernetDownloadsCount + torrentsDownloadsCount) < config.GetMaxSendToProvider() {
				if localDownload.Protocol == config.ProtocolUsenet {
					usenetdownload_id, errAdd := services.TorboxUsenetCreateDownload(localDownload)
					if errAdd != nil {
						logger.Log.Errorw("Failed to add download to provider", "error", errAdd)
						continue
					}
					localDownload.ExternalProviderID = usenetdownload_id
					usernetDownloadsCount++
				} else {
					torrent_id, errAdd := services.TorboxTorrentCreateDownload(localDownload)
					if errAdd != nil {
						logger.Log.Errorw("Failed to add download to provider", "error", errAdd)
						continue
					}
					localDownload.ExternalProviderID = torrent_id
					torrentsDownloadsCount++
				}
				localDownload.Status = config.DOWNLOAD_STATUS_PROVIDER_ADDED
				errorUpdate := databases.UpdateLocalDownload(localDownload)
				if errorUpdate != nil {
					logger.Log.Errorw("Failed to updated local download provider id", "error", errorUpdate)
					continue
				}
				logger.Log.Debugw("Created download on provider", "download", localDownload.ID, "provider", localDownload.ExternalProviderID)
			}

		case config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING:
			for _, providerDownload := range providerDownloads {
				providerDownloadID := strconv.FormatInt(*providerDownload.ID, 10)
				var protocol string
				if providerDownload.DownloadID != nil && strings.Contains(*providerDownload.DownloadID, "SABnzbd") {
					protocol = config.ProtocolUsenet
				} else {
					protocol = config.ProtocolTorrent
				}
				if providerDownloadID == localDownload.ExternalProviderID && protocol == localDownload.Protocol {
					provData, provDataErr := providerDownload.ToJSON()
					if provDataErr != nil {
						logger.Log.Errorw("Failed to get provider data in json", "error", provDataErr)
						continue
					}
					localDownload.Status = services.TranslateTorboxDownloadStatusToLocalStatus(*providerDownload.DownloadState)
					if localDownload.OriginalDownloadUrl != "" {
						localDownload.DownloadName = *providerDownload.Name
					}
					localDownload.ExternalProviderDataObject = provData
					errLocalDownloadUpdate := databases.UpdateLocalDownload(localDownload)
					if errLocalDownloadUpdate != nil {
						logger.Log.Errorw("Failed to update download", "error", errLocalDownloadUpdate)
						continue
					}
				}
			}

		case config.DOWNLOAD_STATUS_PROVIDER_FAILED:
			logger.Log.Infow("Download failed on provider! Marking as failed.", "download", localDownload.DownloadName)
			localDownload.Status = config.DOWNLOAD_STATUS_CLIENT_FAILED
			localDownload.CompletedAt = time.Now()
			databases.UpdateLocalDownload(localDownload)
			continue

		case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
			switch localDownload.DownloadType {
			case config.DOWNLOAD_TYPE_DO_NOT_DOWNLOAD:
				localDownload.Status = config.DOWNLOAD_STATUS_CLIENT_COMPLETED_NOT_DOWNLOADED
				updErr := databases.UpdateLocalDownload(localDownload)
				if updErr != nil {
					logger.Log.Errorw("Failed to update download status", "error", updErr)
				}
				continue

			case config.DOWNLOAD_TYPE_INDIVIDUAL_FILE, config.DOWNLOAD_TYPE_CREATE_SYMLINK, config.DOWNLOAD_TYPE_CREATE_STRM:
				var providerDownloadFromStorage models.DAT
				providerDownloadFromStorage.LoadJSON(localDownload.ExternalProviderDataObject)

				for _, providerDownloadItem := range providerDownloadFromStorage.Files {
					var downloadLink string
					if localDownload.Protocol == config.ProtocolUsenet {
						downloadLink = services.TorboxUsenetRequestDownloadLink(localDownload.ExternalProviderID, providerDownloadItem.IDString(), false)
					} else {
						downloadLink = services.TorboxTorrentRequestDownloadLink(localDownload.ExternalProviderID, providerDownloadItem.IDString(), false)
					}
					downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
						DownloadID:                  localDownload.ID,
						DownloadType:                localDownload.DownloadType,
						Protocol:                    localDownload.Protocol,
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
						continue
					}
					logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)
				}

			case config.DOWNLOAD_TYPE_FULL_ARCHIVE:
				var downloadLink string
				if localDownload.Protocol == config.ProtocolUsenet {
					downloadLink = services.TorboxUsenetRequestDownloadLink(localDownload.ExternalProviderID, "-1", true)
				} else {
					downloadLink = services.TorboxTorrentRequestDownloadLink(localDownload.ExternalProviderID, "-1", true)
				}
				fileInfo, errFile := gdl.GetFileInfo(context.Background(), downloadLink)
				if errFile != nil {
					logger.Log.Errorw("Unable to retrieve file metadata: ", errFile.Error())
					continue
				}
				downloadItemId, errAdd := databases.AddLocalDownloadItem(databases.LocalDownloadsInstanceItem{
					DownloadID:                  localDownload.ID,
					DownloadType:                config.DOWNLOAD_TYPE_FULL_ARCHIVE,
					Protocol:                    localDownload.Protocol,
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
					continue
				}
				logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)

			default:
				continue
			}
			localDownload.Status = config.DOWNLOAD_STATUS_DOWNLOADER_ADDED
			databases.UpdateLocalDownload(localDownload)

		case config.DOWNLOAD_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING, config.DOWNLOAD_STATUS_DOWNLOADER_PROCESSING:
			localDownload.Status = GetLocalDownloadStatusBasedOnItems(localDownload.DownloadItems, config.DOWNLOAD_STATUS_DOWNLOADER_ADDED)
			databases.UpdateLocalDownload(localDownload)

		case config.DOWNLOAD_STATUS_DOWNLOADER_FAILED:
			localDownloadItems := localDownload.DownloadItems
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
				localDownload.Status = config.DOWNLOAD_STATUS_CLIENT_FAILED
				localDownload.CompletedAt = time.Now()
				databases.UpdateLocalDownload(localDownload)
			}
			continue

		case config.DOWNLOAD_STATUS_DOWNLOADER_COMPLETED:
			localDownload.Status = config.DOWNLOAD_STATUS_CLIENT_COMPLETED
			localDownload.CompletedAt = time.Now()
			databases.UpdateLocalDownload(localDownload)
			if localDownload.DownloadType == config.DOWNLOAD_TYPE_CREATE_STRM || localDownload.DownloadType == config.DOWNLOAD_TYPE_CREATE_SYMLINK {
				databases.AddArchivedLocalDownload(databases.LocalArchivedDownloadsInstance{
					Protocol:                   localDownload.Protocol,
					Provider:                   localDownload.Provider,
					DownloadName:               localDownload.DownloadName,
					DownloadType:               localDownload.DownloadType,
					OriginalDownloadUrl:        localDownload.OriginalDownloadUrl,
					OriginalDownloadFile:       localDownload.OriginalDownloadFile,
					OriginalDownloadReference:  localDownload.OriginalDownloadReference,
					Category:                   localDownload.Category,
					Status:                     localDownload.Status,
					ExternalProviderID:         localDownload.ExternalProviderID,
					ExternalProviderDataObject: localDownload.ExternalProviderDataObject,
					AddedAt:                    localDownload.AddedAt,
					CompletedAt:                localDownload.CompletedAt,
					Refresh:                    true,
					LastRefreshAt:              localDownload.CompletedAt,
					DownloadItems:              []databases.LocalArchivedDownloadsInstanceItem{},
				})
			}
			continue

		default:
			continue
		}
	}
	return "Successfully updated downloads", nil
}

func ProcessDownloadItemsQueue() (string, error) {
	logger.Log.Debugw("Processing Download Items!")

	localDownloadItems, err := databases.GetLocalDownloadItems()
	if err != nil {
		logger.Log.Errorw("Unable to retrieve downloads at this time. Please try again later!")
		return "Unable to retrieve downloads at this time. Please try again later!", err
	}

	downloader := services.GetGDLService()
	for _, localDownloadItem := range localDownloadItems {
		switch localDownloadItem.Status {
		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_RETRY:
			switch localDownloadItem.DownloadType {
			case config.DOWNLOAD_TYPE_CREATE_SYMLINK:
				services.CreateSymlink(localDownloadItem)
			case config.DOWNLOAD_TYPE_CREATE_STRM:
				services.CreateStrmlink(localDownloadItem)
			default:
				allDownloaderItems := downloader.Status()
				activeDownloads := 0
				for _, downloaderItem := range allDownloaderItems {
					if downloaderItem.Status == config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING {
						activeDownloads++
					}
				}
				if activeDownloads < config.AppConfig.DownloaderMaxDownloadsConcurrent {
					downloader.Download(context.Background(), localDownloadItem)
				}
			}
			continue

		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING:
			if !downloader.IsDownloading(localDownloadItem.IDString()) {
				localDownloadItem.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_ADDED
				databases.UpdateLocalDownloadItem(localDownloadItem)
			}
			continue

		case config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED:
			if localDownloadItem.RetryCounter < config.AppConfig.DownloaderMaxDownloadsRetry {
				localDownloadItem.RetryCounter = localDownloadItem.RetryCounter + 1
				databases.UpdateLocalDownloadItem(localDownloadItem)
			}
			continue

		default:
			continue
		}
	}
	return "Successfully updated download Items", nil
}

func GetLocalDownloadStatusBasedOnItems(localDownloadItems []databases.LocalDownloadsInstanceItem, currentStatus string) string {
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
