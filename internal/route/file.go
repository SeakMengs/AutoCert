package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_File(r *gin.RouterGroup, fileController *controller.FileController) {
	v1 := r.Group("/v1/files")
	{
		// TODO: update this temp
		v1.GET("/pdf/:page", fileController.ServePdfContentThumbnail)
		v1.GET("/:objectName", fileController.ReadFilePublic)
		v1.POST("", fileController.UploadFilePublic)
	}
}
