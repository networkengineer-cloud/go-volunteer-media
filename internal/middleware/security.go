package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection (legacy but still useful)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy - strict policy for security
		// Note: Adjust CSP based on your frontend requirements
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		c.Header("Content-Security-Policy", csp)

		// Referrer policy - don't leak referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy - restrict feature access
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS - only over HTTPS in production
		// Note: Only enable this if you have HTTPS configured
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}
