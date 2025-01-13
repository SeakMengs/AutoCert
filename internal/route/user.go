package route

import (
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/gin-gonic/gin"
)

func V1_Users(r *gin.RouterGroup, userController *controller.UserController) {
	v1 := r.Group("/v1/users")
	{
		// Test endpoint with curl: curl http://localhost:8080/api/v1/users/1
		v1.GET("/:user_id", userController.GetUserById)
	}
}
