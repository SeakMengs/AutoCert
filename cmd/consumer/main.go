package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/database"
	"github.com/SeakMengs/AutoCert/internal/env"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	// TODO: write the minio.New into a function so it can be used in ./cmd/api/main.go
	// and ./cmd/consumer/main.go
	s3, err := minio.New(cfg.Minio.ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.ACCESS_KEY, cfg.Minio.SECRET_KEY, ""),
		Secure: cfg.Minio.USE_SSL,
		Region: "us-east-1",
	})
	if err != nil {
		logger.Error("Error connecting to minio")
		logger.Panic(err)
	}

	// Custom validation
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("strNotEmpty", util.StrNotEmpty); err != nil {
			return
		}
		if err = v.RegisterValidation("cmin", util.CustomMin); err != nil {
			return
		}
		if err = v.RegisterValidation("cmax", util.CustomMax); err != nil {
			return
		}
	}

	jwtService := auth.NewJwt(cfg.Auth,
		logger)
	repo := repository.NewRepository(db, logger, jwtService, s3)
	app := queue.ConsumerContext{
		Config:     &cfg,
		Repository: repo,
		Logger:     logger,
		JWTService: jwtService,
		S3:         s3,
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

	if err := rabbitMQ.ConsumeCertificateGenerateJob(ctx, CertificateGenerateJobHandler, MAX_WORKER, &app); err != nil {
		logger.Fatalf("Failed to consume certificate generate job: %v", err)
	}

	logger.Infof("Started consuming certificate generate job")

	// Block forever to keep the consumer running
	select {}
}

// Return shouldRequeue, err
func CertificateGenerateJobHandler(ctx context.Context, jobPayload queue.CertificateGeneratePayload, app *queue.ConsumerContext) (bool, error) {
	var queueWaitDuration string
	createdAtTime, err := time.Parse(time.RFC3339, jobPayload.CreatedAt)
	if err != nil {
		app.Logger.Errorf("Failed to parse created_at time: %v", err)
		queueWaitDuration = "unknown"
	} else {
		queueWaitDuration = time.Since(createdAtTime).String()
	}

	user, err := app.Repository.User.GetById(ctx, nil, jobPayload.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			app.Logger.Error("User not found: ", jobPayload.UserID)
			return false, errors.New("user not found")
		}

		app.Logger.Error("Failed to get user: ", err)
		return true, err
	}

	if user == nil {
		app.Logger.Error("User not found: ", jobPayload.UserID)
		return false, errors.New("user not found")
	}

	roles, project, err := app.Repository.Project.GetRoleOfProject(ctx, nil, jobPayload.ProjectID, &auth.JWTPayload{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		ProfileURL: user.ProfileURL,
	})
	if err != nil {
		return true, fmt.Errorf("failed to get project roles: %w", err)
	}

	if project == nil || project.ID == "" {
		app.Logger.Error("Project not found: ", jobPayload.ProjectID)
		return false, errors.New("project not found")
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner}) {
		app.Logger.Warnf("User %s does not have permission to generate certificates for project %s", user.Email, project.ID)
		return false, errors.New("user does not have permission to generate certificates for this project")
	}

	if project.Status != constant.ProjectStatusProcessing {
		app.Logger.Warnf("Project %s is not in processing status, current status: %d", project.ID, project.Status)
		return false, errors.New("project is not in processing status")
	}

	pageAnnotations := autocert.PageAnnotations{
		PageSignatureAnnotations: make(map[uint][]autocert.SignatureAnnotate),
		PageColumnAnnotations:    make(map[uint][]autocert.ColumnAnnotate),
	}

	for _, signature := range project.SignatureAnnotates {
		if signature.Status != constant.SignatoryStatusSigned {
			continue
		}

		annotate, err := signature.ToAutoCertSignatureAnnotate(ctx, app.S3)
		if err != nil {
			app.Logger.Error("failed to convert signature to autocert signature annotate: ", err)
			return true, err
		}
		pageAnnotations.PageSignatureAnnotations[signature.Page] = append(pageAnnotations.PageSignatureAnnotations[signature.Page], *annotate)

		defer os.Remove(annotate.SignatureFilePath)
	}

	for _, column := range project.ColumnAnnotates {
		pageAnnotations.PageColumnAnnotations[column.Page] = append(pageAnnotations.PageColumnAnnotations[column.Page], *column.ToAutoCertColumnAnnotate())
	}

	if len(pageAnnotations.PageSignatureAnnotations) == 0 && len(pageAnnotations.PageColumnAnnotations) == 0 {
		app.Logger.Error("No annotations found for certificate generation")
		return false, errors.New("at least one signed signature or column annotation is required to generate the certificate")
	}

	ext := filepath.Ext(project.TemplateFile.FileName)
	templatePath, err := os.CreateTemp("", "autocert-template-*"+ext)
	if err != nil {
		app.Logger.Error("failed to create temp file", err)
		return true, err
	}
	defer os.Remove(templatePath.Name())

	err = project.TemplateFile.DownloadToLocal(ctx, app.S3, templatePath.Name())
	if err != nil {
		app.Logger.Error("failed to download template file", err)
		return true, err
	}

	csvPath, err := os.CreateTemp("", "autocert-csv-*"+ext)
	if err != nil {
		app.Logger.Error("failed to create temp file", err)
		return true, err
	}
	defer os.Remove(csvPath.Name())

	if project.CSVFileID != "" {
		err = project.CSVFile.DownloadToLocal(ctx, app.S3, csvPath.Name())
		if err != nil {
			app.Logger.Error("failed to download csv file", err)
			return true, err
		}
	} else {
		// create empty csv file
		_, err = csvPath.WriteString("")
		if err != nil {
			app.Logger.Error("failed to create empty csv file", err)
			return true, err
		}
	}

	// Generate certificates
	cfg := autocert.NewDefaultConfig()
	settings := autocert.NewDefaultSettings(fmt.Sprintf("%s/share/certificates", app.Config.FRONTEND_URL) + "/%s")
	settings.EmbedQRCode = project.EmbedQr
	outFilePattern := "certificate_%d.pdf"
	cg := autocert.NewCertificateGenerator(project.ID, templatePath.Name(), csvPath.Name(), *cfg, pageAnnotations, *settings, outFilePattern)

	defer os.RemoveAll(cg.GetOutputDir())

	nowGenerate := time.Now()
	generatedResults, err := cg.Generate()
	if err != nil {
		app.Logger.Error("failed to generate certificate", err)
		return true, fmt.Errorf("failed to generate certificate: %w", err)
	}
	thenGenerate := time.Now()
	app.Logger.Infof("Time taken to generate %d certificates: %v", len(generatedResults), thenGenerate.Sub(nowGenerate))

	tx := app.Repository.DB.Begin()
	defer tx.Commit()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	nowUpload := time.Now()
	for _, gr := range generatedResults {
		// app.Logger.Info("Generated file:", gr.FilePath)

		info, err := util.UploadFileToS3ByPath(gr.FilePath, &util.FileUploadOptions{
			DirectoryPath: util.GetGeneratedCertificateDirectoryPath(project.ID),
			UniquePrefix:  false,
			Bucket:        app.Config.Minio.BUCKET,
			S3:            app.S3,
		})
		if err != nil {
			tx.Rollback()
			app.Logger.Error("failed to upload file to s3", err)
			return true, fmt.Errorf("failed to upload file to s3: %w", err)
		}

		_, err = app.Repository.Certificate.Create(ctx, tx, &model.Certificate{
			BaseModel: model.BaseModel{
				ID: gr.ID,
			},
			Number:    gr.Number,
			ProjectID: project.ID,
		}, &model.File{
			FileName:       util.ToGeneratedCertificateDirectoryPath(project.ID, gr.FilePath),
			UniqueFileName: info.Key,
			BucketName:     info.Bucket,
			Size:           info.Size,
		})

		if err != nil {
			// Delete the file from s3 if certificate creation in db failed
			// Doesn't remove all files, only the last upload. This could potentially leave some files in S3. Can be improved by deleting all files in the directory.
			if err := app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
				app.Logger.Errorf("Failed to delete file: %v", err)
			}
			tx.Rollback()
			return true, fmt.Errorf("failed to create certificate in db: %w", err)
		}
	}

	if err := app.Repository.Project.UpdateStatus(ctx, tx, project.ID, constant.ProjectStatusCompleted); err != nil {
		tx.Rollback()
		return true, fmt.Errorf("failed to update project status to completed: %w", err)
	}

	tx.Commit()

	thenUpload := time.Now()
	app.Logger.Infof("Time taken to upload and save all certificates: %v", thenUpload.Sub(nowUpload))

	thenTotal := time.Now()

	err = app.Repository.ProjectLog.Save(ctx, nil, &model.ProjectLog{
		ProjectID: project.ID,
		Role:      user.Email,
		Action:    "Certificates generated successfully",
		Description: fmt.Sprintf(
			"Generated %d certificates in %s, upload and save in %s, total time taken: %s, total time waited in queue: %s",
			len(generatedResults),
			thenGenerate.Sub(nowGenerate).String(),
			thenUpload.Sub(nowUpload).String(),
			thenTotal.Sub(nowGenerate).String(),
			queueWaitDuration,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		app.Logger.Errorf("Failed to save project log: %v", err)
		return true, fmt.Errorf("failed to save project log: %w", err)
	}

	app.Logger.Infof("Successfully generated %d certificates for project %s", len(generatedResults), project.ID)
	return false, nil
}
