package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_Auth(r *gin.RouterGroup, ac *controller.AuthController) {
	v1 := r.Group("/v1/auth")
	{
		v1.DELETE("/logout", ac.Logout)
		v1.POST("/jwt/access/verify", ac.VerifyJwtAccessToken)
		v1.POST("/jwt/refresh", ac.RefreshAccessToken)
	}
}
