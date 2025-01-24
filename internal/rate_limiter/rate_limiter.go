package ratelimiter

import (
	"github.com/SeakMengs/AutoCert/internal/config"
	"go.uber.org/zap"
)

func NewRateLimiter(cfg config.RateLimiterConfig, logger *zap.SugaredLogger) *FixedWindowRateLimiter {
	return NewFixedWindowLimiter(cfg, logger)
}
