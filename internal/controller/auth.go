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
	token := ctx.Param("token")

	// Keep in mind that verify jwt token does not check database.
	jwtClaims, err := ac.app.JWTService.VerifyJwtToken(token)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(err, nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	if jwtClaims == nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(errors.New("jwt claim empty"), nil), gin.H{
			"tokenValid": false,
		})
		return
	}

	if jwtClaims.Type != constant.JWT_TYPE_ACCESS {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(errors.New("invalid jwt token type"), nil), gin.H{
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
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	jwtClaims, err := ac.app.JWTService.VerifyJwtToken(refreshToken)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	if jwtClaims == nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(errors.New("jwt claim empty"), nil), nil)
		return
	}

	if jwtClaims.Type != constant.JWT_TYPE_REFRESH {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(errors.New("invalid jwt token type"), nil), nil)
		return
	}

	newRefreshToken, newAccessToken, err := ac.app.Repository.JWT.RefreshToken(ctx, nil, refreshToken)
	if err != nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(err, nil), nil)
		return
	}

	if newRefreshToken == nil || newAccessToken == nil {
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", util.GenerateErrorMessage(errors.New("failed to refresh token"), nil), nil)
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"refreshToken": newRefreshToken,
		"accessToken":  newAccessToken,
	})
}
