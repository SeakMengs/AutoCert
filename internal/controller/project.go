package controller

import (
	"fmt"
	"net/http"
	"os"

	"github.com/SeakMengs/AutoCert/internal/model"
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

	// extract the pdf file with selected page
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
