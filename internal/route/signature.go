package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Signatures(r *gin.RouterGroup, sc *controller.SignatureController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/signatures")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.POST("", sc.AddSignature)
		v1.DELETE("/:signatureId", sc.RemoveSignature)
		v1.GET("/:signatureId", sc.GetSignatureById)
	}
}
