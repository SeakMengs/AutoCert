package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

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

// TODO: update this temp
func (fc FileController) ServePdfContentThumbnail(ctx *gin.Context) {
	pdfPath := "autocert_tmp/certificate_merged.pdf"
	outDir := "autocert_tmp/tmp"
	selectedPages := ctx.Params.ByName("page")
	if selectedPages == "" {
		selectedPages = "1"
	}

	tempOutDir, err := os.MkdirTemp(outDir, "autocert_pdf_thumbnail_*")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create temporary directory",
		})
		return
	}
	defer os.RemoveAll(tempOutDir)

	pngFile, err := autocert.PdfToPngByPage(pdfPath, tempOutDir, selectedPages)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to convert PDF to PNG",
		})
		return
	}

	if pngFile == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create PNG file",
		})
		return
	}

	ctx.File(*pngFile)
}
