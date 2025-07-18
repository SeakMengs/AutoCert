package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	"github.com/gin-gonic/gin"
)

func V1_Projects(r *gin.RouterGroup, pc *controller.ProjectController, pbc *controller.ProjectBuilderController, cc *controller.CertificateController, fc *controller.FileController, middleware *middleware.Middleware) {
	v1 := r.Group("/v1/projects")
	v1.Use(middleware.AuthMiddleware)
	{
		v1.POST("", pc.CreateProject)
		v1.GET("/:projectId", pc.GetProjectById)
		v1.DELETE("/:projectId", pc.DeleteProject)
		v1.GET("/:projectId/sse/status", pc.ProjectStatusSSE)
		v1.GET("/:projectId/thumbnail", fc.ServeProjectThumbnail)

		v1.PATCH("/:projectId/visibility", pc.UpdateProjectVisibility)
		v1.PATCH("/:projectId/builder", pbc.ProjectBuilder)
		v1.POST("/:projectId/builder/generate", pc.Generate)
	}
}
