package services

import (
	"fetchtb/config"
	"fetchtb/utils"
	"os"

	"github.com/rkosegi/jdownloader-go/jdownloader"
)

type JDClient struct {
	Client jdownloader.JdClient
	Device jdownloader.Device
}

func NewJDClient() (*JDClient, error) {

	client := jdownloader.NewClient(config.JD_EMAIL.GetValue(), config.JD_PASSWORD.GetValue(), utils.Logger)

	if err := client.Connect(); err != nil {
		utils.Logger.Error("Failed to connect to MyJDownloader", "error", err)
		return nil, err
	}

	var dev jdownloader.Device
	var err error

	dev, err = client.Device(config.JD_DEVICE_NAME.GetValue())

	if err != nil {
		utils.Logger.Error("Failed to obtain JDownloader device", "name", config.JD_DEVICE_NAME.GetValue(), "error", err)
		return nil, err
	}

	utils.Logger.Info("JDownloader Client Connected", "device", dev.Name())

	return &JDClient{
		Client: client,
		Device: dev,
	}, nil
}

func (c *JDClient) AddLink(link string, packageName string, category string) error {

	downloadRoot := os.Getenv("JD_DOWNLOAD_ROOT")

	opts := []jdownloader.AddLinksOptions{
		jdownloader.AddLinksOptionAutostart(true),
		jdownloader.AddLinksOptionPackage(packageName),
		jdownloader.AddLinksOptionDestinationDir(downloadRoot + "/" + category), // Optional
	}

	jdownloader.DefaultMyJdownloaderSettings()

	linkGrabber := c.Device.LinkGrabber()
	_, err := linkGrabber.Add([]string{link}, opts...)

	if err != nil {
		utils.Logger.Errorw("Failed to add link", "error", err)
		return err
	}

	utils.Logger.Infow("Link sent to JDownloader", "package", packageName)
	return nil
}

func (client *JDClient) CheckPackageStatus() ([]string, error) {
	dlPckgs, err := client.Device.Downloader().Packages(
		jdownloader.QueryPackagesOptionDefault(),
	)

	if err != nil {
		utils.Logger.Errorw("Failed to query download packages", "error", err)
	}

	for _, pkg := range *dlPckgs {
		utils.Logger.Infow("Downloading package", "name", *pkg.Name, "status", *pkg.Status)
	}

	var statusList []string

	return statusList, nil
}

func (client *JDClient) CheckLinksStatus() ([]string, error) {
	dlLinks, err := client.Device.Downloader().Links(
		jdownloader.DefaultDownloadQueryLinksOptions(),
	)

	if err != nil {
		utils.Logger.Errorw("Failed to query download links", "error", err)
	}

	for _, link := range *dlLinks {
		utils.Logger.Infow("Downloading link", "url", *link.Url, "status", *link.Status)
	}

	var statusList []string

	return statusList, nil
}

func boolPtr(b bool) *bool {
	return &b
}
