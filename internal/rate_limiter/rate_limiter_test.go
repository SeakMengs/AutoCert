package ratelimiter

import (
	"testing"
	"time"

	"github.com/SeakMengs/AutoCert/internal/config"
)

// Test: Allow 2 requests per 4 seconds
func TestAllowRequest(t *testing.T) {
	const TIME_FRAME_SECOND = 4
	const REQUEST_PER_TIME_FRAME = 2
	cfg := config.RateLimiterConfig{
		RequestsPerTimeFrame: REQUEST_PER_TIME_FRAME,
		TimeFrame:            TIME_FRAME_SECOND * time.Second,
		Enabled:              true,
	}

	limiter := NewRateLimiter(cfg, nil)

	ip := "192.168.1.1"

	start := time.Now()
	for i := 0; i < TIME_FRAME_SECOND; i++ {
		ok, _ := limiter.AllowRequest(ip)

		// When exceed the limit and request allow, it's considered an error
		if i > REQUEST_PER_TIME_FRAME && ok {
			t.Errorf("At request %d expected behavior: deny but got allowed for IP: %s at %s\n", i+1, ip, time.Since(start))
		}
		// Wait a second between requests
		time.Sleep(1 * time.Second)
	}
}
