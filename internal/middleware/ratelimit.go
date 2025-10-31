package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // requests per window
	window   time.Duration // time window
	cleanupInterval time.Duration
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: maximum number of requests per window
// window: time window duration (e.g., time.Minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		window:  window,
		cleanupInterval: window * 10, // Clean up old buckets periodically
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// cleanup removes old buckets to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > rl.window*2 {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed based on the key (e.g., IP address or user ID)
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()
	
	if !exists {
		rl.mu.Lock()
		b = &bucket{
			tokens:     rl.rate,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = b
		rl.mu.Unlock()
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	
	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		b.tokens = rl.rate
		b.lastRefill = now
	}
	
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	
	return false
}

// RateLimit returns a middleware that rate limits requests based on IP address
func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)
	
	return func(c *gin.Context) {
		// Use client IP as the key
		clientIP := c.ClientIP()
		
		if !limiter.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// RateLimitByUser returns a middleware that rate limits requests based on authenticated user ID
func RateLimitByUser(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)
	
	return func(c *gin.Context) {
		// Get user ID from context (set by AuthRequired middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			// If no user context, fall back to IP-based rate limiting
			clientIP := c.ClientIP()
			if !limiter.Allow(clientIP) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				c.Abort()
				return
			}
		} else {
			// Use user ID as the key
			key := string(rune(userID.(uint)))
			if !limiter.Allow(key) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}
