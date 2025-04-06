package controller

import (
	"fmt"
	"net/http"
	"os"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/model"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
)

type ProjectController struct {
	*baseController
}

func (pc ProjectController) CreateProject(ctx *gin.Context) {
	type Request struct {
		Title string `json:"title" form:"title" binding:"required,strNotEmpty,min=1,max=100"`
		Page  uint   `json:"page" form:"page" binding:"required"`
	}
	var body Request

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	err = ctx.ShouldBind(&body)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	file, err := ctx.FormFile("templateFile")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "No template file uploaded", util.GenerateErrorMessages(fmt.Errorf("template file is required"), nil), nil)
		return
	}

	// create temp file for validate and optimized pdf
	tempFile, err := os.CreateTemp("", "template-*.pdf")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp file", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	defer os.Remove(tempFile.Name())

	// Optimize also validate the file
	err = autocert.OptimizePdf(*file, tempFile.Name())
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	src, err := os.Open(tempFile.Name())
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to open optimized file", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	defer src.Close()

	outDir, err := os.MkdirTemp("", "extracted-")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create temp directory", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	defer os.RemoveAll(outDir)

	pageCount, err := autocert.GetPageCount(src)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	if body.Page > 0 && body.Page > uint(pageCount) {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid page number", util.GenerateErrorMessages(fmt.Errorf("page number must be between 1 and %d for the provided template, but got %d", pageCount, body.Page), nil), nil)
		return
	}

	// extract the pdf file with selected page, will be removed after function end
	finalPdf, err := autocert.ExtractPdfByPage(src.Name(), outDir, fmt.Sprintf("%d", body.Page))
	if err != nil || finalPdf == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid template file", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	info, err := pc.uploadFileToS3ByPath(finalPdf)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to upload file", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	tx := pc.app.Repository.DB.Begin()
	defer tx.Commit()

	_, err = pc.app.Repository.Project.Create(ctx, nil, &model.Project{
		Title:  body.Title,
		UserID: user.ID,
		TemplateFile: model.File{
			FileName:       info.Key,
			UniqueFileName: util.AddUniquePrefixToFileName(info.Key),
			BucketName:     info.Bucket,
			Size:           info.Size,
		},
	})
	if err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to create project", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, nil)
}

func (pc ProjectController) GetProjectRole(ctx *gin.Context) {
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(fmt.Errorf("project ID is required"), nil, "projectId"), nil)
		return
	}

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	role, _, err := pc.app.Repository.Project.GetRoleOfProject(ctx, nil, projectId, user)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"role": role,
	})
}

func (pc ProjectController) GetProjectById(ctx *gin.Context) {
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(fmt.Errorf("project ID is required"), nil, "projectId"), nil)
		return
	}

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	role, project, err := pc.app.Repository.Project.GetRoleOfProject(ctx, nil, projectId, user)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project role", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	if project == nil {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(fmt.Errorf("project not found"), nil), nil)
		return
	}

	signatories, err := pc.app.Repository.Project.GetProjectSignatories(ctx, nil, project.ID)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project signatories", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	templateUrl, err := project.TemplateFile.ToPresignedUrl(ctx, pc.app.S3)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get template file URL", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"role": role,
		"project": repository.ProjectResponse{
			ID:          project.ID,
			Title:       project.Title,
			TemplateUrl: templateUrl,
			IsPublic:    project.IsPublic,
			Signatories: signatories,
			Status:      project.Status,
			CreatedAt:   project.CreatedAt,
		},
	})
}

func (pc ProjectController) GetProjectList(ctx *gin.Context) {
	type Request struct {
		Page     uint                     `json:"page" form:"page" binding:"omitempty"`
		PageSize uint                     `json:"pageSize" form:"pageSize" binding:"omitempty"`
		Status   []constant.ProjectStatus `json:"status" form:"status" binding:"omitempty"`
		Search   string                   `json:"search" form:"search" binding:"omitempty"`
	}
	var params Request

	user, err := pc.getAuthUser(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "Unauthorized", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	err = ctx.ShouldBindQuery(&params)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Invalid request", util.GenerateErrorMessages(err, nil), nil)
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
		params.Status = []constant.ProjectStatus{constant.ProjectStatusCompleted, constant.ProjectStatusPreparing, constant.ProjectStatusCompleted}
	}

	projectList, totalCount, err := pc.app.Repository.Project.GetProjectsForOwner(ctx, nil, user, params.Search, params.Status, params.Page, params.PageSize)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project list", util.GenerateErrorMessages(err, nil), nil)
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
