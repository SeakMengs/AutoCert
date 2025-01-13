package controller

import (
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	*baseController
}

func (ac AuthController) VerifyJwtToken(ctx *gin.Context) {
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
		util.ResponseFailed(ctx, http.StatusUnauthorized, "", "Invalid token", gin.H{
			"tokenValid": false,
		})
		return
	}

	util.ResponseSuccess(ctx, gin.H{
		"tokenValid": true,
		"payload":    jwtClaims,
	})
}
