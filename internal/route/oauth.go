package route

import (
	"github.com/SeakMengs/go-api-boilerplate/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_OAuth(r *gin.RouterGroup, oauthController *controller.OAuthController) {
	v1 := r.Group("/v1/oauth")
	{
		v1.GET("/google", oauthController.ContinueWithGoogle)
		v1.GET("/google/callback", oauthController.ContinueWithGoogleCallback)
	}
}
