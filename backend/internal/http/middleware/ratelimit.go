package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// normalizeOrigin returns scheme://host[:port] with no trailing slash/path.
func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(strings.TrimSpace(raw), "/")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}

// RateLimiter is a simple per-IP token bucket (per process; not distributed).
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     time.Duration
	burst    int
}

type visitor struct {
	tokens    float64
	last      time.Time
	lastSeen  time.Time
}

// NewRateLimiter allows burst tokens replenished at 1 token per rate interval.
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	if rate <= 0 {
		rate = time.Second
	}
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &visitor{tokens: float64(rl.burst - 1), last: now, lastSeen: now}
		return true
	}
	elapsed := now.Sub(v.last).Seconds()
	v.tokens += elapsed / rl.rate.Seconds()
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.last = now
	v.lastSeen = now
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// Limit applies the rate limiter; on exceed returns 429.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(clientIP(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "too many requests"},
			})
			return
		}
		c.Next()
	}
}
