package adapters

import (
	"fetchtb/config"
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

		errorUpdate := UpdateLocalDownloadDetails(id, usenetdownload_id, "", config.DOWNLOAD_STATUS_PROVIDER_ADDED)
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
