package adapters

import (
	"fetchtb/config"
	"fetchtb/databases"
	"fetchtb/models"
	"fetchtb/services"
	"fetchtb/utils"
	"mime/multipart"
	"strconv"
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

		utils.Logger.Debugw("Created download on provider", localDownloadUpdated)

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

		utils.Logger.Debugw("Created download on provider", localDownloadUpdated)

		return localDownloadUpdated.ID, nil
	default:
		// "torrent":
		return "", nil
	}
}

func UpdateDownloads() (string, error) {

	downloads, err := databases.GetLocalPendingDownloads()
	if err != nil {
		return "", err
	}

	for _, download := range downloads {
		switch download.Protocol {
		case "usenet":
			switch download.Status {
			case config.DOWNLOAD_STATUS_PROVIDER_ADDED, config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING, config.DOWNLOAD_STATUS_PROVIDER_PROCESSING:
				providerDownloads, err := services.TorboxUsenetGetDownloadList()
				if err != nil {
					utils.Logger.Errorw("Failed to get download status from provider", "error", err)
					continue
				}

				for _, providerDownload := range providerDownloads {
					providerDownloadID := strconv.FormatInt(*providerDownload.ID, 10)
					if providerDownloadID == download.ExternalIDProvider {
						err := databases.UpdateLocalDownloadStatus(download.ID, services.TorboxUsenetDownloadStatusTranslate(*providerDownload.DownloadState))
						if err != nil {
							utils.Logger.Errorw("Failed to update download status", "error", err)
						}
					}
				}
				utils.Logger.Debugw("Updating download", "download", download, "providerStatus", providerDownloads)
				continue

			case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
				downloadLink, requestDownloadLinkError := services.TorboxUsenetRequestDownloadLink(download)
				if requestDownloadLinkError != nil {
					utils.Logger.Errorw("Failed to request download link from provider", "error", requestDownloadLinkError)
					continue
				}
				utils.Logger.Debugw("Requested download link from provider", "downloadLink", downloadLink)

				updateExternalLinkError := databases.UpdateLocalDownloadProviderUrl(download.ID, downloadLink)
				if updateExternalLinkError != nil {
					utils.Logger.Errorw("Failed to update download link in database", "error", updateExternalLinkError)
					continue
				}

				utils.Logger.Debugw("Updated download link in database", "downloadID", download.ID, "downloadLink", downloadLink)
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

func ListDownloads() ([]models.LocalDownloadInstance, error) {
	downloads, err := databases.GetLocalDownloads()
	if err != nil {
		return nil, err
	}

	return downloads, nil
}
