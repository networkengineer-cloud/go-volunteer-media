package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cloudflare IP ranges (update periodically from https://www.cloudflare.com/ips-v4)
var cloudflareIPv4Ranges = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
}

var cloudflareIPv6Ranges = []string{
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

// CloudflareOnly middleware restricts access to only Cloudflare IP ranges
// This ensures all traffic is proxied through Cloudflare for DDoS protection and WAF
func CloudflareOnly() gin.HandlerFunc {
	// Parse CIDR ranges once at startup
	var ipv4Nets []*net.IPNet
	var ipv6Nets []*net.IPNet

	for _, cidr := range cloudflareIPv4Ranges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			ipv4Nets = append(ipv4Nets, ipNet)
		}
	}

	for _, cidr := range cloudflareIPv6Ranges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			ipv6Nets = append(ipv6Nets, ipNet)
		}
	}

	return func(c *gin.Context) {
		// Skip check in development mode (allow local testing)
		if gin.Mode() == gin.DebugMode {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()
		ip := net.ParseIP(clientIP)
		if ip == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid IP address",
			})
			c.Abort()
			return
		}

		// Check if IP is in Cloudflare ranges
		allowed := false
		if ip.To4() != nil {
			// IPv4
			for _, ipNet := range ipv4Nets {
				if ipNet.Contains(ip) {
					allowed = true
					break
				}
			}
		} else {
			// IPv6
			for _, ipNet := range ipv6Nets {
				if ipNet.Contains(ip) {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied - requests must come through Cloudflare",
			})
			c.Abort()
			return
		}

		// Validate Cloudflare headers (additional security layer)
		cfRay := c.GetHeader("CF-Ray")
		if cfRay == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Missing Cloudflare headers",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CloudflareRealIP middleware extracts the real client IP from Cloudflare headers
// Use this AFTER CloudflareOnly() to get the actual visitor IP
func CloudflareRealIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		// CF-Connecting-IP contains the real client IP
		if cfIP := c.GetHeader("CF-Connecting-IP"); cfIP != "" {
			c.Request.RemoteAddr = cfIP
		}

		// Set country code from Cloudflare
		if cfCountry := c.GetHeader("CF-IPCountry"); cfCountry != "" {
			c.Set("country", cfCountry)
		}

		c.Next()
	}
}

// CloudflareHeaders middleware validates and extracts useful Cloudflare headers
func CloudflareHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := map[string]string{
			"ray":           c.GetHeader("CF-Ray"),           // Unique request ID
			"visitor":       c.GetHeader("CF-Visitor"),       // Protocol info (http/https)
			"connecting_ip": c.GetHeader("CF-Connecting-IP"), // Real client IP
			"country":       c.GetHeader("CF-IPCountry"),     // Client country
			"device_type":   c.GetHeader("CF-Device-Type"),   // mobile/desktop/tablet
		}

		// Store in context for logging
		for key, value := range headers {
			if value != "" {
				c.Set("cf_"+key, value)
			}
		}

		c.Next()
	}
}

// isPrivateIP checks if an IP is in private ranges (for development)
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// DevelopmentBypass allows local development without Cloudflare
func DevelopmentBypass(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		ip := net.ParseIP(strings.Split(clientIP, ":")[0])

		if ip != nil && isPrivateIP(ip) {
			c.Next()
			return
		}

		c.Next()
	}
}
