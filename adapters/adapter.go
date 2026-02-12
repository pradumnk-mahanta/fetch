package adapters

import (
	"fetch/config"
	"fetch/databases"
	"fetch/models"
	"fetch/services"
	"log/slog"
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

		slog.Debug("Created download on provider", localDownloadUpdated)

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

		slog.Debug("Created download on provider", localDownloadUpdated)

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
					slog.Error("Failed to get download status from provider", "error", err)
					continue
				}

				for _, providerDownload := range providerDownloads {
					providerDownloadID := strconv.FormatInt(*providerDownload.ID, 10)
					if providerDownloadID == download.ExternalIDProvider {
						err := databases.UpdateLocalDownloadStatus(download.ID, services.TorboxUsenetDownloadStatusTranslate(*providerDownload.DownloadState))
						if err != nil {
							slog.Error("Failed to update download status", "error", err)
						}
					}
				}
				slog.Debug("Updating download", "download", download, "providerStatus", providerDownloads)
				continue

			case config.DOWNLOAD_STATUS_PROVIDER_COMPLETED:
				downloadLink, requestDownloadLinkError := services.TorboxUsenetRequestDownloadLink(download)
				if requestDownloadLinkError != nil {
					slog.Error("Failed to request download link from provider", "error", requestDownloadLinkError)
					continue
				}
				slog.Debug("Requested download link from provider", "downloadLink", downloadLink)

				updateExternalLinkError := databases.UpdateLocalDownloadProviderUrl(download.ID, downloadLink)
				if updateExternalLinkError != nil {
					slog.Error("Failed to update download link in database", "error", updateExternalLinkError)
					continue
				}

				download.ExternalDownloadProviderURL = downloadLink
				downloader := services.GetGDLService()
				err := downloader.Download(download)
				if err != nil {
					slog.Error("Failed to start download in downloader", "error", err)
					continue
				}

				slog.Debug("Updated download link in database and started download", "downloadID", download.ID, "downloadLink", downloadLink)
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
