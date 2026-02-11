package services

import (
	"bytes"
	"fetchtb/config"
	"fetchtb/models"
	"fetchtb/utils"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

func TorboxUsenetCreateDownload(localDownload models.LocalDownloadInstance) (string, error) {

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
	request, requestError := http.NewRequest("POST", config.TB_API_BASE_URL+"/usenet/createusenetdownload", payload)

	if requestError != nil {
		utils.Logger.Errorw("Failed to create HTTP request", "error", requestError)
		return "", requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.TB_API_KEY.GetValue())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		utils.Logger.Errorw("Failed to execute HTTP request", "error", requestError)
		return "", requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to read HTTP response body", "error", requestError)
		return "", requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return "", requestError
	}

	if response.Success != true {
		utils.Logger.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return "", requestError
	}

	return strconv.FormatInt(*response.Data.DAT.UsenetdownloadID, 10), nil
}

func TorboxUsenetRequestDownloadLink(localDownload models.LocalDownloadInstance) (string, error) {

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
	request, requestError := http.NewRequest("POST", config.TB_API_BASE_URL+"/usenet/createusenetdownload", payload)

	if requestError != nil {
		utils.Logger.Errorw("Failed to create HTTP request", "error", requestError)
		return "", requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.TB_API_KEY.GetValue())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		utils.Logger.Errorw("Failed to execute HTTP request", "error", requestError)
		return "", requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to read HTTP response body", "error", requestError)
		return "", requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return "", requestError
	}

	if response.Success != true {
		utils.Logger.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return "", requestError
	}

	return strconv.FormatInt(*response.Data.DAT.UsenetdownloadID, 10), nil
}

func TorboxUsenetGetDownloadList() ([]models.DAT, error) {

	client := &http.Client{Timeout: time.Second * 60}
	request, requestError := http.NewRequest("GET", config.TB_API_BASE_URL+"/usenet/mylist", nil)

	var tbDownloads []models.DAT

	if requestError != nil {
		utils.Logger.Errorw("Failed to create HTTP request", "error", requestError)
		return tbDownloads, requestError
	}

	request.Header.Add("Authorization", "Bearer "+config.TB_API_KEY.GetValue())
	request.Header.Set("Accept", "application/json")
	requestResponse, requestError := client.Do(request)
	if requestError != nil {
		utils.Logger.Errorw("Failed to execute HTTP request", "error", requestError)
		return tbDownloads, requestError
	}
	defer requestResponse.Body.Close()

	body, requestError := io.ReadAll(requestResponse.Body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to read HTTP response body", "error", requestError)
		return tbDownloads, requestError
	}

	response, requestError := models.UnmarshalTorBoxAPIRespose(body)
	if requestError != nil {
		utils.Logger.Errorw("Failed to unmarshal Torbox API response", "error", requestError, "responseBody", string(body))
		return tbDownloads, requestError
	}

	if response.Success != true {
		utils.Logger.Errorw("Failed to create usenet download in Torbox", "responseBody", string(body))
		return tbDownloads, requestError
	}

	return response.Data.DATArray, nil
}
