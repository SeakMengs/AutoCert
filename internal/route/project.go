package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Projects(r *gin.RouterGroup, pc *controller.ProjectController, pbc *controller.ProjectBuilderController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/projects")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.POST("", pc.CreateProject)
		v1.GET("/:projectId", pc.GetProjectById)
		v1.GET("/:projectId/role", pc.GetProjectRole)
		v1.PATCH("/:projectId/builder", pbc.ProjectBuilder)
	}
}
