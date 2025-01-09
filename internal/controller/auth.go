package controller

import (
	"net/http"

	"github.com/SeakMengs/go-api-boilerplate/internal/util"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	*baseController
}

func (ac AuthController) LoginWithEmail(ctx *gin.Context) {
	type LoginBody struct {
		Email    string `json:"email" form:"email" binding:"required"`
		Password string `json:"password" form:"password" binding:"required"`
	}

	var credential LoginBody
	if err := ctx.ShouldBind(&credential); err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "", err, nil)
		return
	}
	ac.app.Logger.Debugf("Login with email: %s", credential.Email)

	user, err := ac.app.Repository.User.GetByEmail(ctx, nil, credential.Email)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusBadRequest, "", err, nil)
		return
	}

	ac.app.Logger.Debug("Checking input password with hashed password in database")
	correctPw, err := util.CheckPassword(user.Password, []byte(credential.Password))
	if !correctPw {
		ac.app.Logger.Debugf("User try to login with email but the password is incorrect, error: %v", err)
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", err, nil)
		return
	}

	refreshToken, accessToken, err := ac.app.Repository.JWT.GenRefreshAndAccessToken(ctx, nil, *user)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusInternalServerError, "", err, nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"refreshToken": refreshToken,
		"accessToken":  accessToken,
	})
}

func (ac AuthController) VerifyJwtToken(ctx *gin.Context) {
	//  TODO: check database for token??
	token := ctx.Param("token")

	if err := ac.app.JWTService.VerifyJwtToken(token); err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", err, gin.H{
			"tokenValid": false,
		})
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"tokenValid": true,
	})
}
