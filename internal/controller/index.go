package controller

import (
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

type IndexController struct {
	*baseController
}

func (ic IndexController) Index(ctx *gin.Context) {
	util.ResponseSuccess(ctx, gin.H{
		"message": "Welcome to the api",
	})

	// ctx.JSON(http.StatusUnauthorized, gin.H{
	// 	"message": "Unauthorized refresh access token test",
	// })
}
