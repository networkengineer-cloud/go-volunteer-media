package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCloudflareOnly(t *testing.T) {
	// Set to release mode to enable IP checking
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.DebugMode) // Reset after test

	tests := []struct {
		name           string
		clientIP       string
		cfRayHeader    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid Cloudflare IPv4 - first range",
			clientIP:       "173.245.48.1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Valid Cloudflare IPv4 - middle range",
			clientIP:       "104.16.0.1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Valid Cloudflare IPv4 - last range",
			clientIP:       "131.0.72.1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Invalid IP - not Cloudflare",
			clientIP:       "1.2.3.4",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied - requests must come through Cloudflare",
		},
		{
			name:           "Valid Cloudflare IP but missing CF-Ray header",
			clientIP:       "173.245.48.1",
			cfRayHeader:    "",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Missing Cloudflare headers",
		},
		{
			name:           "Private IP (should be blocked in release mode)",
			clientIP:       "192.168.1.1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied - requests must come through Cloudflare",
		},
		{
			name:           "Localhost (should be blocked in release mode)",
			clientIP:       "127.0.0.1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied - requests must come through Cloudflare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router with middleware
			router := gin.New()
			router.Use(CloudflareOnly())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)
			req.RemoteAddr = tt.clientIP + ":12345"
			if tt.cfRayHeader != "" {
				req.Header.Set("CF-Ray", tt.cfRayHeader)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestCloudflareOnly_DebugMode(t *testing.T) {
	// Ensure debug mode
	gin.SetMode(gin.DebugMode)

	router := gin.New()
	router.Use(CloudflareOnly())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test with non-Cloudflare IP - should pass in debug mode
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "1.2.3.4:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestCloudflareOnly_IPv6(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.DebugMode)

	tests := []struct {
		name           string
		clientIP       string
		cfRayHeader    string
		expectedStatus int
	}{
		{
			name:           "Valid Cloudflare IPv6",
			clientIP:       "2606:4700::1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid IPv6",
			clientIP:       "2001:db8::1",
			cfRayHeader:    "8c9a1b2c3d4e5f6g-DFW",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CloudflareOnly())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)
			req.RemoteAddr = "[" + tt.clientIP + "]:12345"
			req.Header.Set("CF-Ray", tt.cfRayHeader)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCloudflareRealIP(t *testing.T) {
	tests := []struct {
		name               string
		cfConnectingIP     string
		cfCountry          string
		expectedRemoteAddr string
		expectedCountry    string
	}{
		{
			name:               "Extract real IP and country",
			cfConnectingIP:     "203.0.113.1",
			cfCountry:          "US",
			expectedRemoteAddr: "203.0.113.1",
			expectedCountry:    "US",
		},
		{
			name:               "No Cloudflare headers",
			cfConnectingIP:     "",
			cfCountry:          "",
			expectedRemoteAddr: "1.2.3.4:12345",
			expectedCountry:    "",
		},
		{
			name:               "Only IP header present",
			cfConnectingIP:     "203.0.113.1",
			cfCountry:          "",
			expectedRemoteAddr: "203.0.113.1",
			expectedCountry:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CloudflareRealIP())
			router.GET("/test", func(c *gin.Context) {
				country, _ := c.Get("country")
				c.JSON(http.StatusOK, gin.H{
					"remote_addr": c.Request.RemoteAddr,
					"country":     country,
				})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "1.2.3.4:12345"
			if tt.cfConnectingIP != "" {
				req.Header.Set("CF-Connecting-IP", tt.cfConnectingIP)
			}
			if tt.cfCountry != "" {
				req.Header.Set("CF-IPCountry", tt.cfCountry)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedRemoteAddr)
			if tt.expectedCountry != "" {
				assert.Contains(t, w.Body.String(), tt.expectedCountry)
			}
		})
	}
}

func TestCloudflareHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		expectKeys  []string
		missingKeys []string
	}{
		{
			name: "All Cloudflare headers present",
			headers: map[string]string{
				"CF-Ray":           "8c9a1b2c3d4e5f6g-DFW",
				"CF-Visitor":       `{"scheme":"https"}`,
				"CF-Connecting-IP": "203.0.113.1",
				"CF-IPCountry":     "US",
				"CF-Device-Type":   "desktop",
			},
			expectKeys:  []string{"cf_ray", "cf_visitor", "cf_connecting_ip", "cf_country", "cf_device_type"},
			missingKeys: []string{},
		},
		{
			name: "Partial Cloudflare headers",
			headers: map[string]string{
				"CF-Ray":     "8c9a1b2c3d4e5f6g-DFW",
				"CF-Country": "US",
			},
			expectKeys:  []string{"cf_ray"},
			missingKeys: []string{"cf_visitor", "cf_connecting_ip", "cf_device_type"},
		},
		{
			name:        "No Cloudflare headers",
			headers:     map[string]string{},
			expectKeys:  []string{},
			missingKeys: []string{"cf_ray", "cf_visitor", "cf_connecting_ip", "cf_country", "cf_device_type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CloudflareHeaders())
			router.GET("/test", func(c *gin.Context) {
				// Check for expected keys
				result := make(map[string]interface{})
				for _, key := range tt.expectKeys {
					if value, exists := c.Get(key); exists {
						result[key] = value
					}
				}
				c.JSON(http.StatusOK, result)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			// Verify expected keys are present
			for _, key := range tt.expectKeys {
				assert.Contains(t, w.Body.String(), key)
			}

			// Verify missing keys are not present
			for _, key := range tt.missingKeys {
				assert.NotContains(t, w.Body.String(), key)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		isPrivate bool
	}{
		{"Private IPv4 - 10.x.x.x", "10.0.0.1", true},
		{"Private IPv4 - 192.168.x.x", "192.168.1.1", true},
		{"Private IPv4 - 172.16.x.x", "172.16.0.1", true},
		{"Localhost IPv4", "127.0.0.1", true},
		{"Public IPv4", "8.8.8.8", false},
		{"Cloudflare IPv4", "104.16.0.1", false},
		{"Localhost IPv6", "::1", true},
		{"Private IPv6 - fc00", "fc00::1", true},
		{"Public IPv6", "2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIPHelper(t, tt.ip)
			result := isPrivateIP(ip)
			assert.Equal(t, tt.isPrivate, result)
		})
	}
}

func TestDevelopmentBypass(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "Bypass enabled - private IP allowed",
			enabled:        true,
			clientIP:       "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Bypass enabled - public IP still processed",
			enabled:        true,
			clientIP:       "8.8.8.8",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Bypass disabled - all IPs processed",
			enabled:        false,
			clientIP:       "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(DevelopmentBypass(tt.enabled))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)
			req.RemoteAddr = tt.clientIP + ":12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCloudflareMiddlewareChain(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.DebugMode)

	// Test the full middleware chain
	router := gin.New()
	router.Use(CloudflareOnly())
	router.Use(CloudflareRealIP())
	router.Use(CloudflareHeaders())
	router.GET("/test", func(c *gin.Context) {
		cfRay, _ := c.Get("cf_ray")
		country, _ := c.Get("country")
		c.JSON(http.StatusOK, gin.H{
			"remote_addr": c.Request.RemoteAddr,
			"cf_ray":      cfRay,
			"country":     country,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "173.245.48.1")
	req.RemoteAddr = "173.245.48.1:12345"
	req.Header.Set("CF-Ray", "8c9a1b2c3d4e5f6g-DFW")
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("CF-IPCountry", "US")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "203.0.113.1")      // Real IP extracted
	assert.Contains(t, w.Body.String(), "8c9a1b2c3d4e5f6g") // CF-Ray stored
	assert.Contains(t, w.Body.String(), "US")               // Country stored
}

func TestCloudflareOnly_InvalidIP(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.DebugMode)

	router := gin.New()
	router.Use(CloudflareOnly())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "invalid-ip")
	req.RemoteAddr = "invalid-ip:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid IP address")
}

// Helper function to parse IP and fail test if invalid
func parseIPHelper(t *testing.T, ipStr string) netIP {
	ip := parseIP(ipStr)
	if ip == nil {
		t.Fatalf("Failed to parse IP: %s", ipStr)
	}
	return ip
}

// Type alias for cleaner code
type netIP = net.IP

// Helper to parse IP addresses
func parseIP(ipStr string) net.IP {
	return net.ParseIP(ipStr)
}

// Benchmark tests
func BenchmarkCloudflareOnly(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CloudflareOnly())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "173.245.48.1")
	req.Header.Set("CF-Ray", "8c9a1b2c3d4e5f6g-DFW")
	req.RemoteAddr = "173.245.48.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCloudflareHeaders(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CloudflareHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-Ray", "8c9a1b2c3d4e5f6g-DFW")
	req.Header.Set("CF-Visitor", `{"scheme":"https"}`)
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("CF-IPCountry", "US")
	req.Header.Set("CF-Device-Type", "desktop")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkIsPrivateIP(b *testing.B) {
	ip := net.ParseIP("192.168.1.1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPrivateIP(ip)
	}
}
