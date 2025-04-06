package middleware

import (
	appcontext "github.com/SeakMengs/AutoCert/internal/app_context"
	ratelimiter "github.com/SeakMengs/AutoCert/internal/rate_limiter"
)

type Middleware struct {
	rateLimiter *ratelimiter.FixedWindowRateLimiter
	app         *appcontext.Application
}

func NewMiddleware(app *appcontext.Application,
	rateLimiter *ratelimiter.FixedWindowRateLimiter,
) *Middleware {
	return &Middleware{app: app, rateLimiter: rateLimiter}
}
