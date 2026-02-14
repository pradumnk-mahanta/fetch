package logger

import (
	"fetch/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func InitLogger() {
	var zapConfig zap.Config

	if config.APPLICATION_LOG_LEVEL.GetValue() == "DEBUG" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	zapConfig.EncoderConfig.TimeKey = "Timestamp"
	zapConfig.EncoderConfig.LevelKey = "Level"
	zapConfig.EncoderConfig.MessageKey = "Message"
	zapConfig.Encoding = "json"
	zapConfig.DisableCaller = true
	//	zapConfig.EncoderConfig.EncodeLevel = zapcore.COL
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := zapConfig.Build()
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
