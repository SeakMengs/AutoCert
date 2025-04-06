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
		// Test endpoint with curl: curl http://localhost:8080/api/v1/users/1
		v1.GET("/:id", projectController.CreateProject)
	}
}
