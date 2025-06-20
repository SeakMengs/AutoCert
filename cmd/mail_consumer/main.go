package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/database"
	"github.com/SeakMengs/AutoCert/internal/env"
	filestorage "github.com/SeakMengs/AutoCert/internal/file_storage"
	"github.com/SeakMengs/AutoCert/internal/mailer"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// this function run before main
func init() {
	env.LoadEnv(".env")
}

const (
	MAX_WORKER = 3
)

func main() {
	cfg := config.GetConfig()
	logger := util.NewLogger(cfg.ENV)

	db, err := database.ConnectReturnGormDB(cfg.DB)
	if err != nil {
		logger.Panic(err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		logger.Panic(err)
	}
	defer sqlDb.Close()
	logger.Info("Database connected \n")

	s3, err := filestorage.NewMinioClient(&cfg.Minio)
	if err != nil {
		logger.Error("Error connecting to minio")
		logger.Panic(err)
	}

	// Custom validation
	v := validator.New()
	if err := util.RegisterCustomValidations(v); err != nil {
		logger.Panicf("Failed to register custom validations: %v", err)
	}

	mail := mailer.NewGmailMailer(cfg.Mail.GMAIL_USERNAME, cfg.Mail.GMAIL_APP_PASSWORD, logger)
	jwtService := auth.NewJwt(cfg.Auth,
		logger)
	repo := repository.NewRepository(db, logger, jwtService, s3)
	app := queue.MailConsumerContext{
		Config:     &cfg,
		Repository: repo,
		Logger:     logger,
		Mailer:     mail,
	}

	rabbitMQ, err := queue.NewRabbitMQ(cfg.RabbitMQ.GetConnectionString())
	if err != nil {
		logger.Panic("Error connecting to RabbitMQ: ", err)
	}
	logger.Info("RabbitMQ connected \n")
	defer func() {
		if err := rabbitMQ.Close(); err != nil {
			logger.Errorf("Failed to close RabbitMQ connection: %v", err)
		}
	}()

	logger.Infof("Connected to RabbitMQ at %s", cfg.RabbitMQ.GetConnectionString())

	ctx := context.Background()

	if err := rabbitMQ.ConsumeMailJob(ctx, mailJobHandler, MAX_WORKER, &app); err != nil {
		logger.Fatalf("Failed to consume mail job: %v", err)
	}

	logger.Infof("Started consuming mail job")

	// Block forever to keep the consumer running
	select {}
}

func mailJobHandler(ctx context.Context, jobPayload queue.MailJobPayload, app *queue.MailConsumerContext) (bool, error) {
	switch jobPayload.TemplateFile {
	case mailer.TemplateSignatureRequestInvitation:
		var data mailer.SignatureRequestInvitationData
		if err := json.Unmarshal(jobPayload.Data, &data); err != nil {
			return false, fmt.Errorf("failed to unmarshal SignatureRequestInvitationData: %w", err)
		}

		sigAnnot, err := app.Repository.SignatureAnnotate.GetById(ctx, nil, data.SignatureRequestID, data.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, fmt.Errorf("signature request not found: %s", data.SignatureRequestID)
			}

			return true, fmt.Errorf("failed to get signature request: %w", err)
		}

		if sigAnnot.Email != jobPayload.ToEmail {
			return false, fmt.Errorf("email %s does not match signature request email %s", jobPayload.ToEmail, sigAnnot.Email)
		}

		status, err := app.Mailer.Send(jobPayload.TemplateFile, jobPayload.ToEmail, data)
		if err != nil {
			return true, fmt.Errorf("failed to send email: %w", err)
		}

		if status != http.StatusOK {
			return true, fmt.Errorf("email sending failed with status: %d", status)
		}

		return true, nil
	default:
		return false, fmt.Errorf("unsupported template: %s", jobPayload.TemplateFile)
	}
}
