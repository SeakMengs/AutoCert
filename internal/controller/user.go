package controller

import (
	"net/http"

	"github.com/SeakMengs/go-api-boilerplate/internal/model"
	"github.com/SeakMengs/go-api-boilerplate/internal/util"
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
		util.ResponseFailed(ctx, http.StatusInternalServerError, "", err, nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"user": user,
	})
}

func (uc UserController) RegisterUser(ctx *gin.Context) {
	var newUser model.User
	if err := ctx.ShouldBind(&newUser); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "", err, newUser)
		return
	}

	if err := uc.app.Repository.User.CheckDupAndCreate(ctx, nil, newUser); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "", err, newUser)
		return
	}

	util.ResponseSuccess(ctx, newUser)
}
