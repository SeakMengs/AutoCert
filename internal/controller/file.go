package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

type FileController struct {
	*baseController
}

const (
	PUBLIC_BUCKET_NAME    = "public"
	ENCRYPTED_BUCKET_NAME = "encrypted"
)

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

	object, err := fc.app.S3.GetObject(context.Background(), PUBLIC_BUCKET_NAME, objectName, minio.GetObjectOptions{})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error getting object", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error retrieving object info", util.GenerateErrorMessages(err, nil), nil)
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
		util.ResponseFailed(ctx, http.StatusBadRequest, "No file uploaded", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	src, err := file.Open()
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error opening file", util.GenerateErrorMessages(err, nil), nil)
		return
	}
	defer src.Close()

	err = createBucketIfNotExists(fc.app.S3, PUBLIC_BUCKET_NAME)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error creating bucket", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	fileName := util.AddUniqueSuffixToFilename(file.Filename)
	_, err = fc.app.S3.PutObject(context.Background(), PUBLIC_BUCKET_NAME, fileName, src, file.Size, minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	})
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "Error uploading file", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"fileName": fileName,
		"route":    fmt.Sprintf("api/v1/files/%s", fileName),
	})
}
