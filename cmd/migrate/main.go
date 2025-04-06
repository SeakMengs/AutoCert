package main

import (
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/database"
	"github.com/SeakMengs/AutoCert/internal/env"
	"github.com/SeakMengs/AutoCert/internal/model"
	"go.uber.org/zap"
)

func init() {
	env.LoadEnv(".env")
}

func main() {
	logger := zap.Must(zap.NewDevelopment()).Sugar()
	defer logger.Sync()
	cfg := config.GetConfig()

	logger.Debugf("Database configuration: %+v", cfg.DB)

	db, err := database.ConnectReturnGormDB(cfg.DB)
	if err != nil {
		logger.Panic(err)
	}

	db.Exec(`CREATE EXTENSION IF NOT EXISTS citext`)

	migrateErr := db.AutoMigrate(&model.User{}, &model.Token{}, &model.OAuthProvider{}, &model.Project{}, &model.ProjectLogs{}, &model.ColumnAnnotate{}, &model.SignatureAnnotate{}, &model.File{})
	if migrateErr != nil {
		logger.Panic(migrateErr)
	}
}
