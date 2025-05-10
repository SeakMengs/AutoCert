package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Certificates(r *gin.RouterGroup, cc *controller.CertificateController, fc *controller.FileController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/projects/:projectId/certificates")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.GET("/:certificateNumber/thumbnail", fc.ServeProjectCertificateNumberThumbnail)
		v1.GET("", cc.GetCertificatesByProjectId)
		v1.GET("/download", cc.CertificatesToZipByProjectId)
		v1.GET("/merge", cc.MergeCertificatesByProjectId)
	}
}
