package controller

import (
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	*baseController
}

func (uc UserController) GetUserById(ctx *gin.Context) {
	// userId, err := strconv.Atoi(c.Param("user_id"))
	userId := ctx.Param("user_id")
	user, err := uc.app.Repository.User.GetById(ctx, nil, userId)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"user": user,
	})
}
