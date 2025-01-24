package controller

import (
	"errors"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	*baseController
}

func (ac AuthController) VerifyJwtAccessToken(ctx *gin.Context) {
	type Form struct {
		Token string `json:"token" form:"token" binding:"required,strNotEmpty"`
	}
	var form Form

	err := ctx.ShouldBind(&form)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(err, nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	token := form.Token

	// Keep in mind that verify jwt token does not check database.
	jwtClaims, err := ac.app.JWTService.VerifyJwtToken(token)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(err, nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	if jwtClaims == nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(errors.New("jwt claim empty"), nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	if jwtClaims.Type != constant.JWT_TYPE_ACCESS {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(errors.New("invalid jwt token type"), nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"tokenValid": true,
		"payload":    jwtClaims,
	})
}

func (ac AuthController) RefreshAccessToken(ctx *gin.Context) {
	refreshToken, err := util.ReadRefreshToken(ctx)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	jwtClaims, err := ac.app.JWTService.VerifyJwtToken(refreshToken)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	if jwtClaims == nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(errors.New("jwt claim empty"), nil), nil)
		return
	}

	if jwtClaims.Type != constant.JWT_TYPE_REFRESH {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(errors.New("invalid jwt token type"), nil), nil)
		return
	}

	tx := ac.app.Repository.DB.Begin()
	defer tx.Commit()
	ac.app.Logger.Debugf("Refresh access token, Transaction begin")

	newRefreshToken, newAccessToken, err := ac.app.Repository.JWT.RefreshToken(ctx, tx, refreshToken)
	if err != nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(err, nil), nil)
		return
	}

	if newRefreshToken == nil || newAccessToken == nil {
		tx.Rollback()
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessages(errors.New("failed to refresh token"), nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"refreshToken": newRefreshToken,
		"accessToken":  newAccessToken,
	})
}
