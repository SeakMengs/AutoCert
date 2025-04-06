package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Projects(r *gin.RouterGroup, projectController *controller.ProjectController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/projects")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.POST("", projectController.CreateProject)
		v1.GET("/me", projectController.GetProjectList)
		v1.GET("/:projectId", projectController.GetProjectById)
		v1.GET("/:projectId/role", projectController.GetProjectRole)
	}
}
