package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_Auth(r *gin.RouterGroup, authController *controller.AuthController) {
	v1 := r.Group("/v1/auth")
	{
		v1.POST("/jwt/access/verify", authController.VerifyJwtAccessToken)
		v1.POST("/jwt/refresh", authController.RefreshAccessToken)
	}
}
