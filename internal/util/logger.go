package util

import "go.uber.org/zap"

func NewLogger(env string) *zap.SugaredLogger {
	var logger *zap.SugaredLogger

	if env == "production" {
		logger = zap.Must(zap.NewProduction()).Sugar()
	} else {
		logger = zap.Must(zap.NewDevelopment()).Sugar()
	}

	defer logger.Sync()

	return logger
}
