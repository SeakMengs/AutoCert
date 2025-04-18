package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_File(r *gin.RouterGroup, fc *controller.FileController) {
	v1 := r.Group("/v1/files")
	{
		// TODO: update this temp
		// v1.GET("/pdf/:page", fc.ServePdfContentThumbnail)
		v1.GET("/:objectName", fc.ReadFilePublic)
		v1.POST("", fc.UploadFilePublic)
	}
}
