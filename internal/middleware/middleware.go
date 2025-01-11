package middleware

import (
	ratelimiter "github.com/SeakMengs/AutoCert/internal/rate_limiter"
	"go.uber.org/zap"
)

type Middleware struct {
	logger      *zap.SugaredLogger
	rateLimiter *ratelimiter.FixedWindowRateLimiter
}

func NewMiddleware(logger *zap.SugaredLogger,
	rateLimiter *ratelimiter.FixedWindowRateLimiter,
) *Middleware {
	return &Middleware{logger: logger, rateLimiter: rateLimiter}
}
