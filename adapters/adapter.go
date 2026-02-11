package adapters

import (
	"fetchtb/config"
	"fetchtb/models"
	"fetchtb/services"
	"fetchtb/utils"
	"mime/multipart"
)

func CreateDownload(protocol string, downloadName string, downloadFile multipart.File, downloadUrl string, category string) (string, error) {
	// This is where you would implement the logic to create a download entry in the database
	switch protocol {
	case "usenet":
		id := AddLocalDownload(protocol, config.USENET_DOWNLOAD_PROVIDER.GetValue(), downloadName, downloadUrl, downloadFile, category)
		localDownload, err := GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		usenetdownload_id, error := services.TorboxUsenetCreateDownload(localDownload)
		if error != nil {
			return "", error
		}

		errorUpdate := UpdateLocalDownloadProviderId(id, usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
		if errorUpdate != nil {
			return "", errorUpdate
		}

		localDownloadUpdated, err := GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		utils.Logger.Debugw("Created download on provider", localDownloadUpdated)

		return localDownload.ID, nil
	default:
		// "torrent":
		return "", nil
	}
}

func DeleteDownload(protocol string, downloadName string, downloadFile multipart.File, downloadUrl string, category string) (string, error) {
	// This is where you would implement the logic to create a download entry in the database
	switch protocol {
	case "usenet":
		id := AddLocalDownload(protocol, config.USENET_DOWNLOAD_PROVIDER.GetValue(), downloadName, downloadUrl, downloadFile, category)
		localDownload, err := GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		usenetdownload_id, error := services.TorboxUsenetCreateDownload(localDownload)
		if error != nil {
			return "", error
		}

		errorUpdate := UpdateLocalDownloadProviderId(id, usenetdownload_id, config.DOWNLOAD_STATUS_PROVIDER_ADDED)
		if errorUpdate != nil {
			return "", errorUpdate
		}

		localDownloadUpdated, err := GetLocalDownloadDetails(id)
		if err != nil {
			return "", err
		}

		utils.Logger.Debugw("Created download on provider", localDownloadUpdated)

		return localDownload.ID, nil
	default:
		// "torrent":
		return "", nil
	}
}

func UpdateDownloads() (string, error) {
	downloads, err := GetLocalPendingDownloads()
	if err != nil {
		return "", err
	}

	for _, download := range downloads {
		switch download.Protocol {
		case "usenet":
			switch download.Status {
			case config.DOWNLOAD_STATUS_PROVIDER_ADDED:
				providerStatus, err := services.TorboxUsenetGetDownloadList()
				if err != nil {
					utils.Logger.Errorw("Failed to get download status from provider", "error", err)
					continue
				}

				utils.Logger.Debugw("Updating download", "download", download, "providerStatus", providerStatus)

				if err != nil {
					utils.Logger.Errorw("Failed to update download", "error", err)
					continue
				}

				return "Successfully updated downloads", nil
			default:
				return "", nil
			}
		default:
			return "", nil
		}
	}

	return "", nil
}

func ListDownloads() ([]models.LocalDownloadInstance, error) {
	downloads, err := GetLocalDownloads()
	if err != nil {
		return nil, err
	}

	return downloads, nil
}
