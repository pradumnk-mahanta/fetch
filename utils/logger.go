package utils

import (
	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func InitLogger() {
	// Create a production logger (JSON format, Info level)
	logger, _ := zap.NewProduction()

	// Flush buffer when program exits
	// Note: We can't defer here effectively in a global init,
	// but it's good practice in main()

	// Create a "Sugared" logger for easier printf-style logging
	Logger = logger.Sugar()
}

func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}
