package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/queue"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
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

	info, err := util.UploadFileToS3ByPath(finalPdf, &util.FileUploadOptions{
		DirectoryPath: util.GetProjectDirectoryPath(newProjectId),
		UniquePrefix:  true,
		Bucket:        pc.app.Config.Minio.BUCKET,
		S3:            pc.app.S3,
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

	project, err := pc.app.Repository.Project.Create(ctx, tx, &model.Project{
		BaseModel: model.BaseModel{
			ID: newProjectId,
		},
		Title:  body.Title,
		UserID: user.ID,
		TemplateFile: model.File{
			FileName:       util.ToProjectDirectoryPath(newProjectId, file.Filename),
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

	util.ResponseSuccess(ctx, gin.H{
		"projectId": project.ID,
	})
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
// TODO: handle remove signature file
// TODO: handle empty annotates
func (pc ProjectController) Generate(ctx *gin.Context) {
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err), nil)
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

	// if len(project.SignatureAnnotates) == 0 && len(project.ColumnAnnotates) == 0 {
	// 	util.ResponseFailed(ctx, http.StatusBadRequest, "Project must have at least one signature or column annotate", util.GenerateErrorMessages(errors.New("project must have at least one signature or column annotate"), "noAnnotate"), nil)
	// 	return
	// }

	tx := pc.app.Repository.DB.Begin()
	defer tx.Commit()
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
	// commit before publishing to ensure the status is updated
	tx.Commit()

	tx2 := pc.app.Repository.DB.Begin()
	defer tx2.Commit()

	payloadBytes, err := json.Marshal(queue.NewCertificateGeneratePayload(project.ID, user.ID))
	if err != nil {
		tx.Rollback()
		pc.app.Repository.Project.UpdateStatus(ctx, nil, project.ID, project.Status)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create job payload", util.GenerateErrorMessages(err), nil)
		return
	}

	if err := pc.app.Queue.Publish(queue.QueueCertificateGenerate, payloadBytes); err != nil {
		tx.Rollback()
		pc.app.Repository.Project.UpdateStatus(ctx, nil, project.ID, project.Status)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to queue certificate generation", util.GenerateErrorMessages(err), nil)
		return
	}

	if err := pc.app.Repository.ProjectLog.Save(ctx, tx2, &model.ProjectLog{
		ProjectID: project.ID,
		Role:      user.Email,
		Action:    "Requestor generate certificates",
		Description: fmt.Sprintf(
			"%s has requested to generate certificates. The system has added the project to the queue for certificates generation.",
			user.Email,
		),
		Timestamp: time.Now().Format(time.RFC3339),
	}); err != nil {
		tx2.Rollback()
		pc.app.Repository.Project.UpdateStatus(ctx, nil, project.ID, project.Status)
		pc.app.Logger.Errorf("Failed to save project log: %v", err)
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to save project log", util.GenerateErrorMessages(err), nil)
		return
	}

	tx2.Commit()
	util.ResponseSuccess(ctx, nil)
}

func (pc ProjectController) UpdateProjectVisibility(ctx *gin.Context) {
	type UpdateProjectVisibility struct {
		IsPublic bool `json:"isPublic" form:"isPublic"`
	}

	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project id is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}
	var body UpdateProjectVisibility
	err := ctx.ShouldBind(&body)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err), nil)
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

	err = pc.app.Repository.Project.UpdateProjectVisibility(ctx, nil, projectId, body.IsPublic)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to toggle project visibility", util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, nil)
}

func (pc ProjectController) ProjectStatusSSE(ctx *gin.Context) {
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

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")

	sent := 0

	// Stream status, update ever 2 seconds, if status is not processing, close the stream.
	ctx.Stream(func(w io.Writer) bool {
		status, err := pc.app.Repository.Project.GetProjectStatus(ctx, nil, projectId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.SSEvent("status", gin.H{
					"status": -1,
					"error":  util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "notFound"),
				})
				return false
			}

			pc.app.Logger.Errorf("Failed to get project status: %v", err)
			ctx.SSEvent("status", gin.H{
				"status": -1,
				"error":  util.GenerateErrorMessages(err),
			})
			return true
		}

		if status != constant.ProjectStatusProcessing {
			// If status is not processing, close the stream
			ctx.SSEvent("status", gin.H{
				"status": status,
			})
			return false
		}

		// If status is processing, send the status every 2 seconds
		ctx.SSEvent("status", gin.H{
			"status": status,
		})

		if sent > 0 {
			time.Sleep(2 * time.Second)
		}

		sent++
		return true
	})
}
