package ratelimiter

import (
	"sync"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
	"go.uber.org/zap"
)

type FixedWindowRateLimiter struct {
	Enabled bool
	mutex   sync.RWMutex
	clients map[string]int
	limit   int
	window  time.Duration
	logger  *zap.SugaredLogger
}

func NewFixedWindowLimiter(cfg config.RateLimiterConfig, logger *zap.SugaredLogger) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		Enabled: cfg.Enabled,
		clients: make(map[string]int),
		limit:   cfg.RequestsPerTimeFrame,
		window:  cfg.TimeFrame,
		mutex:   sync.RWMutex{},
		logger:  logger,
	}
}

// The return time is not remaining time of rate limit. It's default time frame of rate limit time that we define
func (rl *FixedWindowRateLimiter) AllowRequest(ip string) (bool, time.Duration) {
	if !rl.Enabled {
		rl.logger.Infof("Check rate limit on %s but ratelimiter is disabled! skip check!", ip)
		return true, 0
	}

	rl.mutex.RLock()

	count, exists := rl.clients[ip]
	rl.mutex.RUnlock()

	// Check the current count for the client
	if !exists || count < rl.limit {
		rl.mutex.Lock()

		// Technically if not exists, run resetCount concurrently which allow us to not interrupt the current process.
		// the resetCount will then sleep until time frame we define to reset count.
		if !exists {
			go rl.resetCount(ip)
		}

		rl.clients[ip]++
		rl.mutex.Unlock()
		return true, 0
	}

	return false, rl.window
}

func (rl *FixedWindowRateLimiter) resetCount(ip string) {
	time.Sleep(rl.window)
	rl.mutex.Lock()
	delete(rl.clients, ip)
	rl.mutex.Unlock()
}
