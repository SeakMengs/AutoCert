package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type ProjectController struct {
	*baseController
}

const (
	ErrTemplateFileRequired                 = "template file is required"
	ErrTemplateFileIsInvalidOrNotSupported  = "template file is invalid or not supported"
	ErrFailedToGetPageCountFromTemplateFile = "failed to get page count from template file"
	ErrInvalidPageNumber                    = "page number must be between 1 and %d for the provided template, but got %d"
)

func getProjectDirectoryPath(projectId string) string {
	return fmt.Sprintf("projects/%s", projectId)
}

func getGeneratedCertificateDirectoryPath(projectId string) string {
	return getProjectDirectoryPath(projectId) + "/generated"
}

func (pc ProjectController) CreateProject(ctx *gin.Context) {
	type Request struct {
		Title string `json:"title" form:"title" binding:"required,strNotEmpty,min=1,max=100"`
		Page  int    `json:"page" form:"page" binding:"required,number,gte=1"`
	}
	var body Request

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	err = ctx.ShouldBind(&body)
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err), nil)
		return
	}

	file, err := ctx.FormFile("templateFile")
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "No template file uploaded", util.GenerateErrorMessages(errors.New(ErrTemplateFileRequired), "templateFile"), nil)
		return
	}

	// create temp file for validate and optimized pdf
	tempFile, err := os.CreateTemp("", "template-*.pdf")
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(tempFile.Name())

	// Optimize also validate the file
	err = autocert.OptimizePdf(*file, tempFile.Name())
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(errors.New(ErrTemplateFileIsInvalidOrNotSupported), "templateFile"), nil)
		return
	}

	src, err := os.Open(tempFile.Name())
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to open optimized file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer src.Close()

	outDir, err := os.MkdirTemp("", "extracted-")
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(outDir)

	pageCount, err := autocert.GetPageCount(src)
	if err != nil {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(errors.New(ErrFailedToGetPageCountFromTemplateFile), "templateFile"), nil)
		return
	}
	if body.Page > 0 && body.Page > int(pageCount) {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid page number", util.GenerateErrorMessages(fmt.Errorf(ErrInvalidPageNumber, pageCount, body.Page), "templateFile"), nil)
		return
	}

	// extract the pdf file with selected page, will be removed after function end
	finalPdf, err := autocert.ExtractPdfByPage(src.Name(), outDir, fmt.Sprintf("%d", body.Page))
	if err != nil || finalPdf == "" {
		pc.app.Logger.Error(err)
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(errors.New(ErrTemplateFileIsInvalidOrNotSupported), "templateFile"), nil)
		return
	}

	newProjectId := uuid.NewString()

	info, err := pc.uploadFileToS3ByPath(finalPdf, &FileUploadOptions{
		DirectoryPath: getProjectDirectoryPath(newProjectId),
		UniquePrefix:  true,
	})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to upload file", util.GenerateErrorMessages(err), nil)
		return
	}

	tx := pc.app.Repository.DB.Begin()
	defer tx.Commit()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create project", util.GenerateErrorMessages(errors.New("failed to create project")), nil)
			return
		}
	}()

	_, err = pc.app.Repository.Project.Create(ctx, tx, &model.Project{
		BaseModel: model.BaseModel{
			ID: newProjectId,
		},
		Title:  body.Title,
		UserID: user.ID,
		TemplateFile: model.File{
			FileName:       filepath.Base(finalPdf),
			UniqueFileName: info.Key,
			BucketName:     info.Bucket,
			Size:           info.Size,
		},
	})
	if err != nil {
		// delete the file from s3 if project creation failed
		if err := pc.app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
			pc.app.Logger.Errorf("Failed to delete file: %v", err)
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to delete file", util.GenerateErrorMessages(err), nil)
			return
		}

		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create project", util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, nil)
}

const (
	ErrProjectIdRequired = "project ID is required"
	ErrProjectNotFound   = "project not found"
)

func (pc ProjectController) GetProjectRole(ctx *gin.Context) {
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, _, err := pc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"roles": roles,
	})
}

func (pc ProjectController) GetProjectById(ctx *gin.Context) {
	type ProjectById struct {
		ID                 string                    `json:"id"`
		Title              string                    `json:"title"`
		TemplateUrl        string                    `json:"templateUrl"`
		IsPublic           bool                      `json:"isPublic"`
		Status             constant.ProjectStatus    `json:"status"`
		EmbedQr            bool                      `json:"embedQr"`
		CSVFileUrl         string                    `json:"csvFileUrl"`
		ColumnAnnotates    []model.ColumnAnnotate    `json:"columnAnnotates"`
		SignatureAnnotates []model.SignatureAnnotate `json:"signatureAnnotates"`
	}

	type GetProjectByIdResponse struct {
		Roles   []constant.ProjectRole `json:"roles"`
		Project ProjectById            `json:"project"`
	}

	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, project, err := pc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "notFound"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project"), "forbidden"), nil)
		return
	}

	var templateUrl string
	if project.TemplateFileID != "" {
		templateUrl, err = project.TemplateFile.ToPresignedUrl(ctx, pc.app.S3)
		if err != nil {
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get template file URL", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	var csvFileUrl string
	if project.CSVFileID != "" {
		csvFileUrl, err = project.CSVFile.ToPresignedUrl(ctx, pc.app.S3)
		if err != nil {
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get CSV file URL", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	if len(project.SignatureAnnotates) == 0 {
		project.SignatureAnnotates = []model.SignatureAnnotate{}
	}

	if len(project.ColumnAnnotates) == 0 {
		project.ColumnAnnotates = []model.ColumnAnnotate{}
	}

	if len(csvFileUrl) == 0 {
		csvFileUrl = ""
	}

	if len(templateUrl) == 0 {
		templateUrl = ""
	}

	util.ResponseSuccess(ctx, GetProjectByIdResponse{
		Roles: roles,
		Project: ProjectById{
			ID:                 project.ID,
			Title:              project.Title,
			TemplateUrl:        templateUrl,
			IsPublic:           project.IsPublic,
			Status:             project.Status,
			EmbedQr:            project.EmbedQr,
			CSVFileUrl:         csvFileUrl,
			ColumnAnnotates:    project.ColumnAnnotates,
			SignatureAnnotates: project.SignatureAnnotates,
		},
	})
}

type GetProjectsRequest struct {
	Page     uint                     `json:"page" form:"page" binding:"omitempty"`
	PageSize uint                     `json:"pageSize" form:"pageSize" binding:"omitempty"`
	Status   []constant.ProjectStatus `json:"status" form:"status" binding:"omitempty"`
	Search   string                   `json:"search" form:"search" binding:"omitempty"`
}

func (pc ProjectController) GetOwnProjectList(ctx *gin.Context) {
	var params GetProjectsRequest

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	err = ctx.ShouldBindQuery(&params)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err), nil)
		return
	}

	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = constant.DefaultPageSize
	}
	if params.PageSize > constant.MaxPageSize {
		params.PageSize = constant.MaxPageSize
	}
	if params.Status == nil {
		params.Status = []constant.ProjectStatus{constant.ProjectStatusDraft, constant.ProjectStatusProcessing, constant.ProjectStatusCompleted}
	}

	projectList, totalCount, err := pc.app.Repository.Project.GetProjectsForOwner(ctx, nil, user, params.Search, params.Status, params.Page, params.PageSize)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project list", util.GenerateErrorMessages(err), nil)
		return
	}

	if len(projectList) == 0 {
		projectList = []repository.ProjectResponse{}
	}

	util.ResponseSuccess(ctx, gin.H{
		"total":     totalCount,
		"projects":  projectList,
		"page":      params.Page,
		"pageSize":  params.PageSize,
		"totalPage": util.CalculateTotalPage(totalCount, params.PageSize),
		"search":    params.Search,
		"status":    params.Status,
	})
}

func (pc ProjectController) GetSignatoryProjectList(ctx *gin.Context) {
	var params GetProjectsRequest

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
		return
	}

	err = ctx.ShouldBindQuery(&params)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err), nil)
		return
	}

	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = constant.DefaultPageSize
	}
	if params.PageSize > constant.MaxPageSize {
		params.PageSize = constant.MaxPageSize
	}
	if params.Status == nil {
		params.Status = []constant.ProjectStatus{constant.ProjectStatusDraft, constant.ProjectStatusProcessing, constant.ProjectStatusCompleted}
	}

	projectList, totalCount, err := pc.app.Repository.Project.GetProjectsForSignatory(ctx, nil, user, params.Search, params.Status, params.Page, params.PageSize)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project list", util.GenerateErrorMessages(err), nil)
		return
	}

	if len(projectList) == 0 {
		projectList = []repository.ProjectResponse{}
	}

	util.ResponseSuccess(ctx, gin.H{
		"total":     totalCount,
		"projects":  projectList,
		"page":      params.Page,
		"pageSize":  params.PageSize,
		"totalPage": util.CalculateTotalPage(totalCount, params.PageSize),
		"search":    params.Search,
		"status":    params.Status,
	})
}

// TODO: refactor and cleanup
func (pc ProjectController) Generate(ctx *gin.Context) {
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, project, err := pc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "notFound"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project"), "forbidden"), nil)
		return
	}

	if project.Status != constant.ProjectStatusDraft {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project is not in draft status", util.GenerateErrorMessages(errors.New("project is not in draft status"), "status"), nil)
		return
	}

	tx := pc.app.Repository.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to generate certificate", util.GenerateErrorMessages(errors.New("failed to update project status to proccessing")), nil)
		}
	}()

	if err := pc.app.Repository.Project.UpdateStatus(ctx, tx, project.ID, constant.ProjectStatusProcessing); err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to update project status", util.GenerateErrorMessages(err), nil)
		return
	}

	pageAnnotations := autocert.PageAnnotations{
		PageSignatureAnnotations: make(map[uint][]autocert.SignatureAnnotate),
		PageColumnAnnotations:    make(map[uint][]autocert.ColumnAnnotate),
	}

	for _, signature := range project.SignatureAnnotates {
		if signature.Status != constant.SignatoryStatusSigned {
			continue
		}

		annotate, err := signature.ToAutoCertSignatureAnnotate(ctx, pc.app.S3)
		if err != nil {
			tx.Rollback()

			pc.app.Logger.Error("failed to convert signature to autocert signature annotate", err)
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to process signature annotation", util.GenerateErrorMessages(errors.New("failed to convert signature annotate to autocert signature annotate, most likely because signature file does not exist")), nil)
			return
		}
		pageAnnotations.PageSignatureAnnotations[signature.Page] = append(pageAnnotations.PageSignatureAnnotations[signature.Page], *annotate)

		defer os.Remove(annotate.SignatureFilePath)
	}

	for _, column := range project.ColumnAnnotates {
		pageAnnotations.PageColumnAnnotations[column.Page] = append(pageAnnotations.PageColumnAnnotations[column.Page], *column.ToAutoCertColumnAnnotate())
	}

	ext := filepath.Ext(project.TemplateFile.FileName)

	templatePath, err := os.CreateTemp("", "template-*"+ext)
	if err != nil {
		tx.Rollback()

		pc.app.Logger.Error("failed to create temp file", err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(templatePath.Name())

	err = project.TemplateFile.DownloadToLocal(ctx, pc.app.S3, templatePath.Name())
	if err != nil {
		tx.Rollback()

		pc.app.Logger.Error("failed to download template file", err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to download template file", util.GenerateErrorMessages(err), nil)
		return
	}

	csvPath, err := os.CreateTemp("", "csv-*"+ext)
	if err != nil {
		tx.Rollback()

		pc.app.Logger.Error("failed to create temp file", err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(csvPath.Name())

	if project.CSVFileID != "" {
		err = project.CSVFile.DownloadToLocal(ctx, pc.app.S3, csvPath.Name())
		if err != nil {
			tx.Rollback()

			pc.app.Logger.Error("failed to download csv file", err)
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to download csv file", util.GenerateErrorMessages(err), nil)
			return
		}
	} else {
		// create empty csv file
		_, err = csvPath.WriteString("")
		if err != nil {
			tx.Rollback()

			pc.app.Logger.Error("failed to create empty csv file", err)
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create empty csv file", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	cfg := autocert.NewDefaultConfig()
	settings := autocert.NewDefaultSettings()
	outFilePattern := "certificate %d.pdf"
	cg := autocert.NewCertificateGenerator(project.ID, templatePath.Name(), csvPath.Name(), *cfg, pageAnnotations, *settings, outFilePattern)

	// TIP: Remove this for testing to see the generated files in the output directory
	defer os.RemoveAll(cg.GetOutputDir())

	nowGenerate := time.Now()
	generatedFiles, err := cg.Generate()
	if err != nil {
		tx.Rollback()

		pc.app.Logger.Error("failed to generate certificate", err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to generate certificate", util.GenerateErrorMessages(err), nil)
		return
	}
	thenGenerate := time.Now()
	pc.app.Logger.Infof("Time taken to generate %d certificates: %v", len(generatedFiles), thenGenerate.Sub(nowGenerate))

	tx.Commit()

	tx2 := pc.app.Repository.DB.Begin()
	defer tx2.Commit()
	defer func() {
		if r := recover(); r != nil {
			// change project status back to it original status
			pc.app.Repository.Project.UpdateStatus(ctx, tx2, project.ID, project.Status)

			tx2.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to generate certificate", util.GenerateErrorMessages(errors.New("failed to generate certificate")), nil)
		}
	}()

	nowUpload := time.Now()
	for i, filePath := range generatedFiles {
		pc.app.Logger.Info("Generated file:", filePath)

		info, err := pc.uploadFileToS3ByPath(filePath, &FileUploadOptions{
			DirectoryPath: getGeneratedCertificateDirectoryPath(project.ID),
			UniquePrefix:  false,
		})
		if err != nil {
			pc.app.Logger.Error("failed to upload file to s3", err)
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to upload file", util.GenerateErrorMessages(err), nil)
			return
		}

		_, err = pc.app.Repository.Certificate.Create(ctx, tx2, &model.Certificate{
			Number:    i + 1,
			ProjectID: project.ID,
			CertificateFile: model.File{
				FileName:       filepath.Base(filePath),
				UniqueFileName: info.Key,
				BucketName:     info.Bucket,
				Size:           info.Size,
			},
		})

		if err != nil {
			// delete the file from s3 if certificate creation in db failed
			// Doesn't remove all files, only the last upload. This could potentially leave some files in S3. Can be improved by deleting all files in the directory.
			if err := pc.app.S3.RemoveObject(ctx, info.Bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
				pc.app.Logger.Errorf("Failed to delete file: %v", err)
				tx2.Rollback()
				util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to delete file", util.GenerateErrorMessages(err), nil)
				return
			}

			tx2.Rollback()
			util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to save certificate", util.GenerateErrorMessages(err), nil)
			return
		}
	}

	if err := pc.app.Repository.Project.UpdateStatus(ctx, tx2, project.ID, constant.ProjectStatusCompleted); err != nil {
		tx2.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to update project status", util.GenerateErrorMessages(err), nil)
		return
	}

	thenUpload := time.Now()
	pc.app.Logger.Infof("Time taken to upload and save all certificates: %v", thenUpload.Sub(nowUpload))

	thenTotal := time.Now()

	util.ResponseSuccess(ctx, gin.H{
		"files":             generatedFiles,
		"count":             len(generatedFiles),
		"generateTimeTaken": fmt.Sprintf("Time taken to generate %d certificates: %v", len(generatedFiles), thenGenerate.Sub(nowGenerate)),
		"uploadTimeTaken":   fmt.Sprintf("Time taken to upload and save all certificates: %v", thenUpload.Sub(nowUpload)),
		"totalTimeTaken":    fmt.Sprintf("Total time taken: %v", thenTotal.Sub(nowGenerate)),
	})
}
