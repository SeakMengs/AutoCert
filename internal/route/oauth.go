package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_OAuth(r *gin.RouterGroup, oc *controller.OAuthController) {
	v1 := r.Group("/v1/oauth")
	{
		v1.GET("/google", oc.ContinueWithGoogle)
		v1.GET("/google/callback", oc.ContinueWithGoogleCallback)
	}
}
