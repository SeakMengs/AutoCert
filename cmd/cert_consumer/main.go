package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/database"
	"github.com/SeakMengs/AutoCert/internal/env"
	filestorage "github.com/SeakMengs/AutoCert/internal/file_storage"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// this function run before main
func init() {
	env.LoadEnv(".env")
}

const MAX_WORKERS = 3

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
	logger.Info("Minio connected \n")

	v := validator.New()
	if err := util.RegisterCustomValidations(v); err != nil {
		logger.Panicf("Failed to register custom validations: %v", err)
	}

	jwtService := auth.NewJwt(cfg.Auth, logger)
	repo := repository.NewRepository(db, logger, jwtService, s3)
	app := queue.CertificateConsumerContext{
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

	if err := rabbitMQ.ConsumeCertificateGenerateJob(ctx, certificateGenerateJobHandler, MAX_WORKERS, &app); err != nil {
		logger.Fatalf("Failed to consume certificate generate job: %v", err)
	}

	logger.Infof("Started consuming certificate generate job with %d workers", MAX_WORKERS)

	// Block forever to keep the consumer running
	select {}
}

type uploadResult struct {
	certificateID     string
	certificateNumber int
	fileInfo          minio.UploadInfo
	err               error
}

// Return shouldRequeue, err
func certificateGenerateJobHandler(ctx context.Context, jobPayload queue.CertificateGeneratePayload, app *queue.CertificateConsumerContext) (bool, error) {
	var queueWaitDuration string
	createdAtTime, err := time.Parse(time.RFC3339, jobPayload.CreatedAt)
	if err != nil {
		app.Logger.Errorf("Failed to parse created_at time: %v", err)
		queueWaitDuration = "unknown"
	} else {
		queueWaitDuration = time.Since(createdAtTime).String()
	}

	user, project, _, shouldRequeue, err := validateUserAndProject(ctx, jobPayload, app)
	if err != nil {
		return shouldRequeue, err
	}

	pageAnnotations, tempSigFiles, err := prepareAnnotations(ctx, project, app)
	if err != nil {
		cleanupTempFiles(tempSigFiles)
		return true, err
	}
	defer cleanupTempFiles(tempSigFiles)

	if len(pageAnnotations.PageSignatureAnnotations) == 0 && len(pageAnnotations.PageColumnAnnotations) == 0 {
		app.Logger.Error("No annotations found for certificate generation")
		return false, errors.New("at least one signed signature or column annotation is required to generate the certificate")
	}

	templatePath, csvPath, err := prepareFiles(ctx, project, app)
	if err != nil {
		return true, err
	}
	defer os.Remove(templatePath.Name())
	defer os.Remove(csvPath.Name())

	generatedResults, outputDir, generateDuration, err := generateCertificates(project, templatePath.Name(), csvPath.Name(), pageAnnotations, app)
	if err != nil {
		return true, err
	}
	defer os.RemoveAll(outputDir)

	uploadDuration, err := uploadAndSaveCertificates(ctx, generatedResults, project, app)
	if err != nil {
		return true, err
	}

	totalDuration := generateDuration + uploadDuration
	if err := logProjectSuccess(ctx, user, project, len(generatedResults), generateDuration, uploadDuration, totalDuration, queueWaitDuration, app); err != nil {
		app.Logger.Errorf("Failed to save project log: %v", err)
		return true, fmt.Errorf("failed to save project log: %w", err)
	}

	app.Logger.Infof("Successfully generated %d certificates for project %s", len(generatedResults), project.ID)
	return false, nil
}

func validateUserAndProject(ctx context.Context, jobPayload queue.CertificateGeneratePayload, app *queue.CertificateConsumerContext) (*model.User, *model.Project, []constant.ProjectRole, bool, error) {
	user, err := app.Repository.User.GetById(ctx, nil, jobPayload.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			app.Logger.Error("User not found: ", jobPayload.UserID)
			return nil, nil, nil, false, errors.New("user not found")
		}
		app.Logger.Error("Failed to get user: ", err)
		return nil, nil, nil, true, err
	}

	if user == nil {
		app.Logger.Error("User not found: ", jobPayload.UserID)
		return nil, nil, nil, false, errors.New("user not found")
	}

	roles, project, err := app.Repository.Project.GetRoleOfProject(ctx, nil, jobPayload.ProjectID, &auth.JWTPayload{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		ProfileURL: user.ProfileURL,
	})
	if err != nil {
		return nil, nil, nil, true, fmt.Errorf("failed to get project roles: %w", err)
	}

	if project == nil || project.ID == "" {
		app.Logger.Error("Project not found: ", jobPayload.ProjectID)
		return nil, nil, nil, false, errors.New("project not found")
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner}) {
		app.Logger.Warnf("User %s does not have permission to generate certificates for project %s", user.Email, project.ID)
		return nil, nil, nil, false, errors.New("user does not have permission to generate certificates for this project")
	}

	if project.Status != constant.ProjectStatusProcessing {
		app.Logger.Warnf("Project %s is not in processing status, current status: %d", project.ID, project.Status)
		return nil, nil, nil, false, errors.New("project is not in processing status")
	}

	return user, project, roles, false, nil
}

func prepareAnnotations(ctx context.Context, project *model.Project, app *queue.CertificateConsumerContext) (autocert.PageAnnotations, []string, error) {
	pageAnnotations := autocert.PageAnnotations{
		PageSignatureAnnotations: make(map[uint][]autocert.SignatureAnnotate),
		PageColumnAnnotations:    make(map[uint][]autocert.ColumnAnnotate),
	}
	var tempFiles []string

	for _, signature := range project.SignatureAnnotates {
		if signature.Status != constant.SignatoryStatusSigned {
			continue
		}

		annotate, err := signature.ToAutoCertSignatureAnnotate(ctx, app.S3)
		if err != nil {
			app.Logger.Error("failed to convert signature to autocert signature annotate: ", err)
			return pageAnnotations, tempFiles, err
		}
		pageAnnotations.PageSignatureAnnotations[signature.Page] = append(pageAnnotations.PageSignatureAnnotations[signature.Page], *annotate)
		tempFiles = append(tempFiles, annotate.SignatureFilePath)
	}

	for _, column := range project.ColumnAnnotates {
		pageAnnotations.PageColumnAnnotations[column.Page] = append(pageAnnotations.PageColumnAnnotations[column.Page], *column.ToAutoCertColumnAnnotate())
	}

	return pageAnnotations, tempFiles, nil
}

func cleanupTempFiles(files []string) {
	for _, file := range files {
		os.Remove(file)
	}
}

func prepareFiles(ctx context.Context, project *model.Project, app *queue.CertificateConsumerContext) (*os.File, *os.File, error) {
	ext := filepath.Ext(project.TemplateFile.FileName)
	templatePath, err := util.CreateTemp("autocert-template-*" + ext)
	if err != nil {
		app.Logger.Error("failed to create temp file", err)
		return nil, nil, err
	}

	err = project.TemplateFile.DownloadToLocal(ctx, app.S3, templatePath.Name())
	if err != nil {
		os.Remove(templatePath.Name())
		app.Logger.Error("failed to download template file", err)
		return nil, nil, err
	}

	csvPath, err := util.CreateTemp("autocert-csv-*" + ext)
	if err != nil {
		os.Remove(templatePath.Name())
		app.Logger.Error("failed to create temp file", err)
		return nil, nil, err
	}

	if project.CSVFileID != "" {
		err = project.CSVFile.DownloadToLocal(ctx, app.S3, csvPath.Name())
		if err != nil {
			os.Remove(templatePath.Name())
			os.Remove(csvPath.Name())
			app.Logger.Error("failed to download csv file", err)
			return nil, nil, err
		}
	} else {
		// create empty csv file
		_, err = csvPath.WriteString("")
		if err != nil {
			os.Remove(templatePath.Name())
			os.Remove(csvPath.Name())
			app.Logger.Error("failed to create empty csv file", err)
			return nil, nil, err
		}
	}

	return templatePath, csvPath, nil
}

func generateCertificates(project *model.Project, templatePath, csvPath string, pageAnnotations autocert.PageAnnotations, app *queue.CertificateConsumerContext) ([]autocert.GeneratedResult, string, time.Duration, error) {
	cfg := autocert.NewDefaultConfig()
	settings := autocert.NewDefaultSettings(fmt.Sprintf("%s/share/certificates", app.Config.FRONTEND_URL) + "/%s")
	settings.EmbedQRCode = project.EmbedQr
	outFilePattern := "certificate_%d.pdf"
	cg := autocert.NewCertificateGenerator(project.ID, templatePath, csvPath, *cfg, pageAnnotations, *settings, outFilePattern)

	startTime := time.Now()
	generatedResults, err := cg.Generate()
	duration := time.Since(startTime)

	if err != nil {
		app.Logger.Error("failed to generate certificate", err)
		return nil, cg.OutputDir(), 0, fmt.Errorf("failed to generate certificate: %w", err)
	}

	app.Logger.Infof("Time taken to generate %d certificates: %v", len(generatedResults), duration)
	return generatedResults, cg.OutputDir(), duration, nil
}

func uploadAndSaveCertificates(ctx context.Context, generatedResults []autocert.GeneratedResult, project *model.Project, app *queue.CertificateConsumerContext) (time.Duration, error) {
	startTime := time.Now()

	uploadedFiles, err := uploadFilesWithCleanup(ctx, generatedResults, project, app)
	if err != nil {
		return 0, err
	}

	certificates := make([]*model.Certificate, len(uploadedFiles))
	for i, result := range uploadedFiles {
		certificates[i] = &model.Certificate{
			BaseModel: model.BaseModel{
				ID: result.certificateID,
			},
			Number:    result.certificateNumber,
			ProjectID: project.ID,
			CertificateFile: model.File{
				FileName:       util.ToGeneratedCertificateDirectoryPath(project.ID, result.fileInfo.Key),
				UniqueFileName: result.fileInfo.Key,
				BucketName:     result.fileInfo.Bucket,
				Size:           result.fileInfo.Size,
			},
		}
	}

	tx := app.Repository.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if _, err := app.Repository.Certificate.CreateMany(ctx, tx, certificates); err != nil {
		tx.Rollback()
		cleanupUploadedFiles(ctx, uploadedFiles, app)
		return 0, fmt.Errorf("failed to create certificates in db: %w", err)
	}

	if err := app.Repository.Project.UpdateStatus(ctx, tx, project.ID, constant.ProjectStatusCompleted); err != nil {
		tx.Rollback()
		cleanupUploadedFiles(ctx, uploadedFiles, app)
		return 0, fmt.Errorf("failed to update project status to completed: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		cleanupUploadedFiles(ctx, uploadedFiles, app)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(startTime)
	app.Logger.Infof("Time taken to upload and save all certificates: %v", duration)
	return duration, nil
}

func uploadFilesWithCleanup(ctx context.Context, generatedResults []autocert.GeneratedResult, project *model.Project, app *queue.CertificateConsumerContext) ([]uploadResult, error) {
	maxUploadWorkers := util.DetermineWorkers(len(generatedResults))
	app.Logger.Infof("Using %d workers for uploading files", maxUploadWorkers)

	taskChan := make(chan autocert.GeneratedResult, len(generatedResults))
	resultChan := make(chan uploadResult, len(generatedResults))
	var wg sync.WaitGroup

	for range maxUploadWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for result := range taskChan {
				info, err := util.UploadFileToS3ByPath(result.FilePath, &util.FileUploadOptions{
					DirectoryPath: util.GetGeneratedCertificateDirectoryPath(project.ID),
					UniquePrefix:  false,
					Bucket:        app.Config.Minio.BUCKET,
					S3:            app.S3,
				})

				resultChan <- uploadResult{
					certificateID:     result.ID,
					certificateNumber: result.Number,
					fileInfo:          info,
					err:               err,
				}
			}
		}()
	}

	// Send task to worker
	for _, gr := range generatedResults {
		taskChan <- gr
	}
	close(taskChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and check for errors
	var uploadedFiles []uploadResult
	var uploadErrors []error

	for result := range resultChan {
		if result.err != nil {
			uploadErrors = append(uploadErrors, result.err)
		} else {
			uploadedFiles = append(uploadedFiles, result)
		}
	}

	// Remove the uploaded files if there were any errors
	if len(uploadErrors) > 0 {
		app.Logger.Errorf("Failed to upload %d files: %v", len(uploadErrors), uploadErrors[0])
		cleanupUploadedFiles(ctx, uploadedFiles, app)
		return nil, fmt.Errorf("failed to upload %d files: %v", len(uploadErrors), uploadErrors[0])
	}

	return uploadedFiles, nil
}

func cleanupUploadedFiles(ctx context.Context, uploadedFiles []uploadResult, app *queue.CertificateConsumerContext) {
	for _, result := range uploadedFiles {
		if err := app.S3.RemoveObject(ctx, result.fileInfo.Bucket, result.fileInfo.Key, minio.RemoveObjectOptions{}); err != nil {
			app.Logger.Errorf("Failed to delete file during cleanup of certificate number %d: %v", result.certificateNumber, err)
		}
	}
}

func logProjectSuccess(ctx context.Context, user *model.User, project *model.Project, certCount int, generateDuration, uploadDuration, totalDuration time.Duration, queueWaitDuration string, app *queue.CertificateConsumerContext) error {
	return app.Repository.ProjectLog.Save(ctx, nil, &model.ProjectLog{
		ProjectID: project.ID,
		Role:      user.Email,
		Action:    "Certificates generated successfully",
		Description: fmt.Sprintf(
			"Generated %d certificates in %s, upload and save in %s, total time taken: %s, total time waited in queue: %s",
			certCount,
			generateDuration.String(),
			uploadDuration.String(),
			totalDuration.String(),
			queueWaitDuration,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
