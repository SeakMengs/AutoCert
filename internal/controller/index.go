package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

type IndexController struct {
	*baseController
}

func (ic IndexController) Index(ctx *gin.Context) {
	bucketName := "mybucket"
	objectName := "certificate.png"

	// Get the object from MinIO
	object, err := ic.app.S3.GetObject(context.Background(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error getting object",
			"error":   err,
		})
		return
	}
	defer object.Close()

	// Get object metadata
	info, err := object.Stat()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Error retrieving object info", "error": err})
		return
	}

	// Set headers and stream image
	ctx.Header("Content-Type", info.ContentType)
	ctx.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	io.Copy(ctx.Writer, object)
}

// func (ic IndexController) Index(ctx *gin.Context) {
// 	// util.ResponseSuccess(ctx, gin.H{
// 	// 	"message": "Welcome to the api",
// 	// })
//
// 	ctx.JSON(http.StatusUnauthorized, gin.H{
// 		"message": "Unauthorized refresh access token test",
// 	})
// }
