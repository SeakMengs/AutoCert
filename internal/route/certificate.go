package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Certificates(r *gin.RouterGroup, cc *controller.CertificateController, fc *controller.FileController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/certificates")
	v1Auth := r.Group("/v1/projects/:projectId/certificates")
	v1Auth.Use(middleware.AuthMiddleware)
	{
		v1Auth.GET("/:certificateNumber/thumbnail", fc.ServeProjectCertificateNumberThumbnail)
		v1Auth.GET("", cc.GetCertificatesByProjectId)
		v1Auth.GET("/download", cc.CertificatesToZipByProjectId)
		v1Auth.GET("/merge", cc.MergeCertificatesByProjectId)
	}

	{
		v1.GET("/:certificateId", cc.GetGeneratedCertificateById)
	}
}
