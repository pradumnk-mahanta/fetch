package adapters

import (
	"context"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fetch/services"
	"mime/multipart"
	"strconv"

	"github.com/forest6511/gdl"
)

func CreateDownload(protocol string, downloadName string, downloadFile multipart.File, downloadUrl string, category string) (string, error) {
	// This is where you would implement the logic to create a download entry in the database
	switch protocol {
	case "usenet":
		id := databases.AddLocalDownload(protocol, config.USENET_DOWNLOAD_PROVIDER.GetValue(), downloadName, downloadUrl, downloadFile, category)
		localDownload, err := databases.GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		usenetdownload_id, error := services.TorboxUsenetCreateDownload(localDownload)
		if error != nil {
			return "", error
		}

		errorUpdate := databases.UpdateLocalDownloadProviderId(id, usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
		if errorUpdate != nil {
			return "", errorUpdate
		}

		localDownloadUpdated, err := databases.GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		logger.Log.Debugw("Created download on provider", "download", localDownloadUpdated)

		return localDownloadUpdated.ID, nil
	default:
		// "torrent":
		return "", nil
	}
}

func DeleteDownload(protocol string, downloadName string, downloadFile multipart.File, downloadUrl string, category string) (string, error) {
	// This is where you would implement the logic to create a download entry in the database
	switch protocol {
	case "usenet":
		id := databases.AddLocalDownload(protocol, config.USENET_DOWNLOAD_PROVIDER.GetValue(), downloadName, downloadUrl, downloadFile, category)
		localDownload, err := databases.GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		usenetdownload_id, error := services.TorboxUsenetCreateDownload(localDownload)
		if error != nil {
			return "", error
		}

		errorUpdate := databases.UpdateLocalDownloadProviderId(id, usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
		if errorUpdate != nil {
			return "", errorUpdate
		}

		localDownloadUpdated, err := databases.GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		logger.Log.Debugw("Created download on provider", "download", localDownloadUpdated)

		return localDownloadUpdated.ID, nil
	default:
		// "torrent":
		return "", nil
	}
}

func UpdateDownloads() (string, error) {
	localDownloads, err := databases.GetLocalPendingDownloads()
	if err != nil {
		logger.Log.Errorw("Unable to retrieve pending downloads at this time. Please try again later!")
		return "Unable to retrieve pending downloads at this time. Please try again later!", err
	}

	for _, localDownload := range localDownloads {
		switch localDownload.Protocol {
		case "usenet":
			switch localDownload.Status {
			case config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING:
				providerDownloads, err := services.TorboxUsenetGetDownloadList()
				if err != nil {
					logger.Log.Errorw("Failed to get download status from provider", "error", err)
					continue
				}

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
				continue

			case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
				if config.TB_CONFIG_PREFER_ZIPPED_FOLDER.GetBoolValue() {
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
					downloadItemId := databases.AddLocalDownloadItem(localDownload.ID, config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE, fileInfo.Filename, fileInfo.Filename, fileInfo.Size, "-1", downloadLink)
					logger.Log.Infow("Added download Item Id", "item", downloadItemId)
				} else {
					var providerDownloadFromStorage models.DAT
					providerDownloadFromStorage.LoadJSON(localDownload.ExternalProviderDataObject)
					for _, providerDownloadItem := range providerDownloadFromStorage.Files {
						downloadItemId := databases.AddLocalDownloadItem(localDownload.ID, config.DOWNLOAD_ITEM_TYPE_INDIVIDUAL_FILE, providerDownloadItem.ShortName, providerDownloadItem.Name, providerDownloadItem.Size, strconv.FormatInt(providerDownloadItem.ID, 10), "")
						logger.Log.Infow("Added download Item Id", "download", localDownload.ID, "item", downloadItemId)
					}
				}
				databases.UpdateLocalDownloadStatus(localDownload.ID, config.DOWNLOAD_STATUS_DOWNLOADER_ADDED)
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
	err := downloader.Resume(id)
	if err != nil {
		return false, err
	}
	return true, nil
}
