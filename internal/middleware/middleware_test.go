package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	// Set up JWT secret for testing
	os.Setenv("JWT_SECRET", "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=")
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		setupEnv        func()
		cleanupEnv      func()
		origin          string
		method          string
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name: "allowed origin from env",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173,http://example.com")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "http://localhost:5173",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "http://localhost:5173",
		},
		{
			name: "different allowed origin from env",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173,http://example.com")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "http://example.com",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "http://example.com",
		},
		{
			name: "disallowed origin",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "http://malicious.com",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "", // Should not set CORS header
		},
		{
			name: "default origins when env not set",
			setupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			cleanupEnv:      func() {},
			origin:          "http://localhost:5173",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "http://localhost:5173",
		},
		{
			name: "wildcard origin",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "*")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "http://any-origin.com",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "*",
		},
		{
			name: "OPTIONS preflight request",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "http://localhost:5173",
			method:          "OPTIONS",
			wantStatus:      204,
			wantAllowOrigin: "http://localhost:5173",
		},
		{
			name: "no origin header",
			setupEnv: func() {
				os.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
			},
			cleanupEnv: func() {
				os.Unsetenv("ALLOWED_ORIGINS")
			},
			origin:          "",
			method:          "GET",
			wantStatus:      200,
			wantAllowOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			// Create test server
			router := gin.New()
			router.Use(CORS())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			// Create request
			req, _ := http.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.wantStatus {
				t.Errorf("CORS() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check CORS headers
			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.wantAllowOrigin {
				t.Errorf("CORS() Allow-Origin = %v, want %v", gotOrigin, tt.wantAllowOrigin)
			}

			// For successful requests, check other CORS headers
			if tt.wantStatus == 200 || tt.wantStatus == 204 {
				if tt.wantAllowOrigin != "" {
					credentials := w.Header().Get("Access-Control-Allow-Credentials")
					if credentials != "true" {
						t.Errorf("CORS() Allow-Credentials = %v, want true", credentials)
					}
				}
			}
		})
	}
}

func TestAuthRequired(t *testing.T) {
	// Generate a valid token for testing
	validToken, _ := auth.GenerateToken(1, false, false)
	adminToken, _ := auth.GenerateToken(2, true, false)
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc19hZG1pbiI6ZmFsc2UsImV4cCI6MTYwMDAwMDAwMH0.invalid"

	tests := []struct {
		name         string
		authHeader   string
		wantStatus   int
		wantError    string
		checkContext bool
		wantUserID   uint
		wantIsAdmin  bool
	}{
		{
			name:         "valid token",
			authHeader:   "Bearer " + validToken,
			wantStatus:   200,
			checkContext: true,
			wantUserID:   1,
			wantIsAdmin:  false,
		},
		{
			name:         "valid admin token",
			authHeader:   "Bearer " + adminToken,
			wantStatus:   200,
			checkContext: true,
			wantUserID:   2,
			wantIsAdmin:  true,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: 401,
			wantError:  "Authorization header required",
		},
		{
			name:       "invalid format - no Bearer",
			authHeader: validToken,
			wantStatus: 401,
			wantError:  "Invalid authorization format",
		},
		{
			name:       "invalid format - wrong prefix",
			authHeader: "Basic " + validToken,
			wantStatus: 401,
			wantError:  "Invalid authorization format",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid.token.here",
			wantStatus: 401,
			wantError:  "Invalid or expired token",
		},
		{
			name:       "expired token",
			authHeader: "Bearer " + expiredToken,
			wantStatus: 401,
			wantError:  "Invalid or expired token",
		},
		{
			name:       "malformed bearer format",
			authHeader: "Bearer",
			wantStatus: 401,
			wantError:  "Invalid authorization format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			router := gin.New()
			router.Use(AuthRequired())
			router.GET("/protected", func(c *gin.Context) {
				if tt.checkContext {
					userID, exists := c.Get("user_id")
					if !exists {
						t.Error("user_id not set in context")
					} else if userID.(uint) != tt.wantUserID {
						t.Errorf("user_id = %v, want %v", userID, tt.wantUserID)
					}

					isAdmin, exists := c.Get("is_admin")
					if !exists {
						t.Error("is_admin not set in context")
					} else if isAdmin.(bool) != tt.wantIsAdmin {
						t.Errorf("is_admin = %v, want %v", isAdmin, tt.wantIsAdmin)
					}
				}
				c.JSON(200, gin.H{"message": "ok"})
			})

			// Create request
			req, _ := http.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.wantStatus {
				t.Errorf("AuthRequired() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error message for failed auth
			if tt.wantError != "" {
				body := w.Body.String()
				if !contains([]string{body}, tt.wantError) && !containsSubstring(body, tt.wantError) {
					t.Errorf("AuthRequired() body = %v, want to contain %v", body, tt.wantError)
				}
			}
		})
	}
}

func TestAdminRequired(t *testing.T) {
	tests := []struct {
		name       string
		isAdmin    bool
		setContext bool
		wantStatus int
		wantError  string
	}{
		{
			name:       "admin user",
			isAdmin:    true,
			setContext: true,
			wantStatus: 200,
		},
		{
			name:       "regular user",
			isAdmin:    false,
			setContext: true,
			wantStatus: 403,
			wantError:  "Admin access required",
		},
		{
			name:       "no context set",
			setContext: false,
			wantStatus: 403,
			wantError:  "Admin access required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			router := gin.New()

			// Middleware to set context
			router.Use(func(c *gin.Context) {
				if tt.setContext {
					c.Set("is_admin", tt.isAdmin)
				}
				c.Next()
			})

			router.Use(AdminRequired())
			router.GET("/admin", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "admin area"})
			})

			// Create request
			req, _ := http.NewRequest("GET", "/admin", nil)

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.wantStatus {
				t.Errorf("AdminRequired() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error message for failed auth
			if tt.wantError != "" {
				body := w.Body.String()
				if !containsSubstring(body, tt.wantError) {
					t.Errorf("AdminRequired() body = %v, want to contain %v", body, tt.wantError)
				}
			}
		})
	}
}

func TestAuthRequiredAndAdminRequiredChained(t *testing.T) {
	// Generate tokens
	regularToken, _ := auth.GenerateToken(1, false, false)
	adminToken, _ := auth.GenerateToken(2, true, false)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "admin token - both middleware pass",
			authHeader: "Bearer " + adminToken,
			wantStatus: 200,
		},
		{
			name:       "regular token - fails admin check",
			authHeader: "Bearer " + regularToken,
			wantStatus: 403,
		},
		{
			name:       "no token - fails auth check",
			authHeader: "",
			wantStatus: 401,
		},
		{
			name:       "invalid token - fails auth check",
			authHeader: "Bearer invalid.token",
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server with both middleware chained
			router := gin.New()
			router.Use(AuthRequired())
			router.Use(AdminRequired())
			router.GET("/admin-only", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "admin only area"})
			})

			// Create request
			req, _ := http.NewRequest("GET", "/admin-only", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.wantStatus {
				t.Errorf("Chained middleware status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{
		{
			name:  "string found",
			slice: []string{"apple", "banana", "cherry"},
			str:   "banana",
			want:  true,
		},
		{
			name:  "string not found",
			slice: []string{"apple", "banana", "cherry"},
			str:   "grape",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			str:   "apple",
			want:  false,
		},
		{
			name:  "string with spaces - trimmed",
			slice: []string{"  apple  ", "banana"},
			str:   "apple",
			want:  true,
		},
		{
			name:  "exact match required",
			slice: []string{"apple", "banana"},
			str:   "app",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.str)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSecurityHeaders tests security headers middleware
func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name        string
		wantHeaders map[string]string
	}{
		{
			name: "security headers are set",
			wantHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-XSS-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
				"Permissions-Policy":     "geolocation=(), microphone=(), camera=()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			router := gin.New()
			router.Use(SecurityHeaders())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status
			if w.Code != 200 {
				t.Errorf("SecurityHeaders() status = %v, want 200", w.Code)
			}

			// Check all expected headers
			for header, expectedValue := range tt.wantHeaders {
				gotValue := w.Header().Get(header)
				if gotValue != expectedValue {
					t.Errorf("SecurityHeaders() %s = %v, want %v", header, gotValue, expectedValue)
				}
			}

			// Check CSP header separately as it's complex
			cspHeader := w.Header().Get("Content-Security-Policy")
			if cspHeader == "" {
				t.Error("SecurityHeaders() Content-Security-Policy header not set")
			}

			// Verify CSP contains key directives
			expectedCSPDirectives := []string{
				"default-src 'self'",
				"frame-src 'self' blob:",
				"frame-ancestors 'none'",
				"base-uri 'self'",
			}
			for _, directive := range expectedCSPDirectives {
				if !containsSubstring(cspHeader, directive) {
					t.Errorf("SecurityHeaders() CSP missing directive: %s", directive)
				}
			}
		})
	}
}

// TestRateLimit tests IP-based rate limiting
func TestRateLimit(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		// Create test server with rate limit of 5 requests per second
		router := gin.New()
		router.Use(RateLimit(5, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// First 5 requests should succeed
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345" // Same IP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		// Create test server with rate limit of 3 requests per second
		router := gin.New()
		router.Use(RateLimit(3, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.2:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}

		// 4th request should be rate limited
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 429 {
			t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
		}

		body := w.Body.String()
		if !containsSubstring(body, "Too many requests") {
			t.Errorf("Expected rate limit error message, got: %s", body)
		}
	})

	t.Run("different IPs are tracked separately", func(t *testing.T) {
		// Create test server with rate limit of 2 requests per second
		router := gin.New()
		router.Use(RateLimit(2, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// IP1: 2 requests should succeed
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.10:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("IP1 Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}

		// IP2: 2 requests should also succeed (independent tracking)
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.20:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("IP2 Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}
	})
}

// TestRateLimitByUser tests user-based rate limiting
func TestRateLimitByUser(t *testing.T) {
	t.Run("rate limits authenticated users", func(t *testing.T) {
		// Create test server with rate limit of 3 requests per second
		router := gin.New()

		// Middleware to set user context
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Next()
		})

		router.Use(RateLimitByUser(3, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}

		// 4th request should be rate limited
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 429 {
			t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
		}
	})

	t.Run("falls back to IP when no user context", func(t *testing.T) {
		// Create test server with rate limit of 2 requests per second
		router := gin.New()
		router.Use(RateLimitByUser(2, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// Should use IP-based rate limiting
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}

		// 3rd request should be rate limited
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 429 {
			t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
		}
	})

	t.Run("different users are tracked separately", func(t *testing.T) {
		// Create test server with rate limit of 2 requests per second
		router := gin.New()

		// Track which user ID to use
		var requestCount int

		router.Use(func(c *gin.Context) {
			requestCount++
			if requestCount <= 2 {
				c.Set("user_id", uint(1))
			} else {
				c.Set("user_id", uint(2))
			}
			c.Next()
		})

		router.Use(RateLimitByUser(2, 1*time.Second))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// User 1: 2 requests should succeed
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("User1 Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}

		// User 2: 2 requests should also succeed (independent tracking)
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("User2 Request %d: expected status 200, got %d", i+1, w.Code)
			}
		}
	})
}

func TestRequestID(t *testing.T) {
	t.Run("generates request ID when not provided", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			requestID := c.GetString("request_id")
			if requestID == "" {
				t.Error("Expected request_id to be set in context")
			}
			c.String(200, "ok")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check response header
		requestID := w.Header().Get(RequestIDKey)
		if requestID == "" {
			t.Error("Expected X-Request-ID header in response")
		}
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			requestID := c.GetString("request_id")
			if requestID != "test-request-id-123" {
				t.Errorf("Expected request_id to be test-request-id-123, got %s", requestID)
			}
			c.String(200, "ok")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set(RequestIDKey, "test-request-id-123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check response header matches
		requestID := w.Header().Get(RequestIDKey)
		if requestID != "test-request-id-123" {
			t.Errorf("Expected X-Request-ID to be test-request-id-123, got %s", requestID)
		}
	})
}
