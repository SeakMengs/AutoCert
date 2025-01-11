package middleware

import (
	"fmt"
	"net/http"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-gonic/gin"
)

func (m Middleware) RateLimiterMiddleware(ctx *gin.Context) {
	if !m.rateLimiter.Enabled {
		ctx.Next()
	}

	ip := ctx.ClientIP()
	m.logger.Infof("Check rate limit for IP: %s", ip)
	allow, retryAfter := m.rateLimiter.AllowRequest(ip)

	if !allow {
		m.logger.Infof("IP: %s exceeded rate limit", ip)
		ctx.Header("Retry-After", retryAfter.String())
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, util.BuildResponseFailed(fmt.Sprintf("Too many request, rate limit exceeded. Retry after %s", retryAfter.String()), nil, nil))
		return
	}

	ctx.Next()
}
