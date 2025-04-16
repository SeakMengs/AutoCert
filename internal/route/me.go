package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Me(r *gin.RouterGroup, pc *controller.ProjectController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/me")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.GET("/projects", pc.GetOwnProjectList)
		v1.GET("/projects/signatory", pc.GetSignatoryProjectList)
	}
}
