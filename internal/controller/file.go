package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/SeakMengs/AutoCert/pkg/autocert"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

type FileController struct {
	*baseController
}

func createBucketIfNotExists(s3 *minio.Client, bucketName string) error {
	exists, err := s3.BucketExists(context.Background(), bucketName)
	if err != nil {
		return err
	}

	if !exists {
		err = s3.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (fc FileController) ReadFilePublic(ctx *gin.Context) {
	objectName := ctx.Params.ByName("objectName")

	object, err := fc.app.S3.GetObject(context.Background(), fc.app.Config.Minio.BUCKET, objectName, minio.GetObjectOptions{})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error getting object", util.GenerateErrorMessages(err), nil)
		return
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error retrieving object info", util.GenerateErrorMessages(err), nil)
		return
	}

	// Set headers and stream image
	ctx.Header("Content-Type", info.ContentType)
	ctx.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	io.Copy(ctx.Writer, object)
}

func (fc FileController) UploadFilePublic(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "No file uploaded", util.GenerateErrorMessages(err), nil)
		return
	}

	src, err := file.Open()
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error opening file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer src.Close()

	err = createBucketIfNotExists(fc.app.S3, fc.app.Config.Minio.BUCKET)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating bucket", util.GenerateErrorMessages(err), nil)
		return
	}

	fileName := util.AddUniquePrefixToFileName(file.Filename)
	_, err = fc.app.S3.PutObject(ctx, fc.app.Config.Minio.BUCKET, fileName, src, file.Size, minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error uploading file", util.GenerateErrorMessages(err), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"fileName": fileName,
		"route":    fmt.Sprintf("api/v1/files/%s", fileName),
	})
}

func (fc FileController) ServePdfContentThumbnail(ctx *gin.Context) {
	selectedPages := "1"
	projectId := ctx.Params.ByName("projectId")
	if projectId == "" {
		util.ResponseFailed(ctx, http.StatusBadRequest, "Project ID is required", util.GenerateErrorMessages(errors.New(ErrProjectIdRequired), "projectId"), nil)
		return
	}

	roles, project, err := fc.getProjectRole(ctx, projectId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Failed to get project roles", util.GenerateErrorMessages(err), nil)
		return
	}

	if project == nil || project.ID == "" {
		util.ResponseFailed(ctx, http.StatusNotFound, "Project not found", util.GenerateErrorMessages(errors.New(ErrProjectNotFound), nil, "project"), nil)
		return
	}

	if !util.HasRole(roles, []constant.ProjectRole{constant.ProjectRoleOwner, constant.ProjectRoleSignatory}) {
		util.ResponseFailed(ctx, http.StatusForbidden, "You do not have permission to access this project", util.GenerateErrorMessages(errors.New("you do not have permission to access this project"), nil), nil)
		return
	}

	tempOutDir, err := os.MkdirTemp("", "autocert_pdf_thumbnail_*")
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating temporary directory", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.RemoveAll(tempOutDir)

	pdfPath := fmt.Sprintf("%s/%s.pdf", tempOutDir, projectId)

	err = project.TemplateFile.DownloadToLocal(ctx, fc.app.S3, pdfPath)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error downloading PDF file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(pdfPath)

	if _, err := os.Stat(pdfPath); errors.Is(err, os.ErrNotExist) {
		util.ResponseFailed(ctx, http.StatusNotFound, "PDF file not found", util.GenerateErrorMessages(err), nil)
		return
	}

	pngFile, err := autocert.PdfToPngByPage(pdfPath, tempOutDir, selectedPages)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error converting PDF to PNG", util.GenerateErrorMessages(err), nil)
		return
	}

	if pngFile == nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating PNG file", util.GenerateErrorMessages(err), nil)
		return
	}
	defer os.Remove(*pngFile)

	ctx.File(*pngFile)
}
