package util

import "go.uber.org/zap"

func NewLogger() *zap.SugaredLogger {
	logger := zap.Must(zap.NewDevelopment()).Sugar()
	defer logger.Sync()

	return logger
}
