package services

import (
	"bytes"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func TorboxUsenetCreateDownload(localDownload databases.LocalDownloadsInstance) (string, error) {

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, errFileCreation := writer.CreateFormFile("file", localDownload.DownloadName+".nzb")
	if errFileCreation != nil {
		return "", errFileCreation
	}

	_, errWriteFile := part.Write(localDownload.OriginalDownloadFile)
	if errWriteFile != nil {
		return "", errWriteFile
	}

	_ = writer.WriteField("name", localDownload.DownloadName)

	errWriterClose := writer.Close()
	if errWriterClose != nil {
		return "", errWriterClose
	}

	client := &http.Client{Timeout: time.Second * 60}
	request, requestError := http.NewRequest("POST", config.GetUsenetProvider().APIEndpoint+"/usenet/createusenetdownload", payload)

	if requestError != nil {
		logger.Log.Errorw("Failed to create HTTP request", "error", requestError)
		return "", requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.GetUsenetProvider().APIKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		logger.Log.Errorw("Failed to execute HTTP request", "error", requestError)
		return "", requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		logger.Log.Errorw("Failed to read HTTP response body", "error", requestError)
		return "", requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		logger.Log.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return "", requestError
	}

	if response.Success != true {
		logger.Log.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return "", requestError
	}

	return strconv.FormatInt(*response.Data.DAT.UsenetdownloadID, 10), nil
}

func TorboxUsenetRequestDownloadLink(externalProviderId string, externalProviderItemId string) (string, error) {
	baseUrl, _ := url.Parse(config.GetUsenetProvider().APIEndpoint + "/usenet/requestdl")

	queryParams := baseUrl.Query()
	queryParams.Set("token", config.GetUsenetProvider().APIKey)
	queryParams.Set("usenet_id", externalProviderId)
	if externalProviderItemId != "-1" {
		queryParams.Set("file_id", externalProviderItemId)
	}
	if config.GetUsenetProvider().PreferZippedFolder {
		queryParams.Set("zip_link", "true")
	}

	baseUrl.RawQuery = queryParams.Encode()

	client := &http.Client{Timeout: time.Second * 60}
	request, requestError := http.NewRequest("GET", baseUrl.String(), nil)

	if requestError != nil {
		logger.Log.Errorw("Failed to create HTTP request", "error", requestError)
		return "", requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.GetUsenetProvider().APIKey)
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		logger.Log.Errorw("Failed to execute HTTP request", "error", requestError)
		return "", requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		logger.Log.Errorw("Failed to read HTTP response body", "error", requestError)
		return "", requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		logger.Log.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return "", requestError
	}

	if response.Success != true {
		logger.Log.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return "", requestError
	}

	return *response.Data.String, nil
}

func TorboxUsenetGetDownloadList() ([]models.DAT, error) {

	client := &http.Client{Timeout: time.Second * 60}
	request, requestError := http.NewRequest("GET", config.GetUsenetProvider().APIEndpoint+"/usenet/mylist", nil)

	var tbDownloads []models.DAT

	if requestError != nil {
		logger.Log.Errorw("Failed to create HTTP request", "error", requestError)
		return tbDownloads, requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.GetUsenetProvider().APIKey)
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		logger.Log.Errorw("Failed to execute HTTP request", "error", requestError)
		return tbDownloads, requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		logger.Log.Errorw("Failed to read HTTP response body", "error", requestError)
		return tbDownloads, requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		logger.Log.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return tbDownloads, requestError
	}

	if response.Success != true {
		logger.Log.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return tbDownloads, requestError
	}

	return response.Data.DATArray, nil
}

func TorboxUsenetDownloadStatusTranslate(status string) string {
	switch {
	case strings.Contains(status, "processing"):
		return config.DOWNLOAD_STATUS_PROVIDER_PROCESSING
	case strings.Contains(status, "downloading"):
		return config.DOWNLOAD_STATUS_PROVIDER_DOWNLOADING
	case strings.Contains(status, "completed"):
		return config.DOWNLOAD_STATUS_PROVIDER_COMPLETED
	case strings.Contains(status, "failed"):
		return config.DOWNLOAD_STATUS_PROVIDER_FAILED
	default:
		return config.DOWNLOAD_STATUS_PROVIDER_ADDED
	}
}
