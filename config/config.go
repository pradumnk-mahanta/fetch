package config

import "os"

type EnvKey string

func (key EnvKey) GetValue() string {
	return os.Getenv(string(key))
}

const (
	JD_EMAIL         EnvKey = "JD_EMAIL"
	JD_PASSWORD      EnvKey = "JD_PASSWORD"
	JD_DEVICE_NAME   EnvKey = "JD_DEVICE_NAME"
	JD_DOWNLOAD_ROOT EnvKey = "JD_DOWNLOAD_ROOT"
	TB_API_KEY       EnvKey = "TB_API_KEY"
)

const (
	TB_API_BASE_URL = "https://api.torbox.net/v1"
)
