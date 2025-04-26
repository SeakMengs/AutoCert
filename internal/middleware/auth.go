package middleware

import (
	"github.com/SeakMengs/AutoCert/internal/constant"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

func (m Middleware) AuthMiddleware(ctx *gin.Context) {
	token, err := util.ReadBearerToken(ctx)
	if err != nil {
		m.app.Logger.Debugf("Failed to read token: %v", err)
		util.ResponseFailed(ctx, 401, "", util.GenerateErrorMessages(err, "unauthorized"), nil)
		ctx.Abort()
		return
	}

	claim, err := m.app.JWTService.VerifyJwtToken(token)
	if err != nil {
		m.app.Logger.Debugf("Failed to verify token: %v", err)
		util.ResponseFailed(ctx, 401, "Invalid token", util.GenerateErrorMessages(err, "unauthorized"), nil)
		ctx.Abort()
		return
	}

	if claim.Type != constant.JWT_TYPE_ACCESS {
		m.app.Logger.Debugf("Invalid token type: %s", claim.Type)
		util.ResponseFailed(ctx, 401, "Invalid access token type", util.GenerateErrorMessages(err, "unauthorized"), nil)
		ctx.Abort()
		return
	}

	ctx.Set("user", claim.User)
	ctx.Next()
}
