package logger

import (
	"fetch/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func InitLogger() {
	var zapCionfig zap.Config
	if config.APPLICATION_LOG_MODE.GetValue() == "DEBUG" {
		zapCionfig = zap.NewDevelopmentConfig()
		zapCionfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapCionfig = zap.NewProductionConfig()
		zapCionfig.EncoderConfig.TimeKey = "timestamp"
		zapCionfig.Encoding = "console"
	}

	zapCionfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapCionfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := zapCionfig.Build()
	if err != nil {
		panic(err)
	}

	Log = logger.Sugar()
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
