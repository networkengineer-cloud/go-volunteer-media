package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMaxRequestBodySizeOverride verifies that applying MaxRequestBodySize a second
// time (e.g. a per-route override after a global middleware) replaces the limit
// rather than nesting it.  Before the fix a 12 MB body was rejected even when the
// per-route limit was raised to 25 MB, because the 10 MB global wrapper was still
// the innermost reader.
func TestMaxRequestBodySizeOverride(t *testing.T) {
	const (
		globalLimit   = 10 * 1024 * 1024 // 10 MB
		perRouteLimit = 25 * 1024 * 1024 // 25 MB
		bodySize      = 12 * 1024 * 1024 // 12 MB — between the two limits
	)

	router := gin.New()
	// Simulate the global middleware applied to all routes.
	router.Use(MaxRequestBodySize(globalLimit))

	// The per-route middleware raises the limit to 25 MB.
	router.POST("/upload", MaxRequestBodySize(perRouteLimit), func(c *gin.Context) {
		// Read the entire body to trigger MaxBytesReader enforcement.
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	body := strings.NewReader(strings.Repeat("x", bodySize))
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for %d-byte body with %d-byte per-route limit, got %d; "+
			"the global %d-byte limit may be incorrectly nesting the per-route override",
			bodySize, perRouteLimit, w.Code, globalLimit)
	}
}

// TestMaxRequestBodySizePerRouteEnforced verifies that the per-route limit still
// rejects bodies that exceed it, ensuring the fix did not simply disable
// per-route enforcement.
func TestMaxRequestBodySizePerRouteEnforced(t *testing.T) {
	const (
		globalLimit   = 10 * 1024 * 1024 // 10 MB
		perRouteLimit = 25 * 1024 * 1024 // 25 MB
		bodySize      = 30 * 1024 * 1024 // 30 MB — exceeds per-route limit
	)

	router := gin.New()
	router.Use(MaxRequestBodySize(globalLimit))

	router.POST("/upload", MaxRequestBodySize(perRouteLimit), func(c *gin.Context) {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	body := strings.NewReader(strings.Repeat("x", bodySize))
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("expected rejection for %d-byte body exceeding the %d-byte per-route limit, got 200",
			bodySize, perRouteLimit)
	}
}

// TestMaxRequestBodySizeGlobalEnforced verifies that the global limit still
// rejects bodies that exceed both the global and per-route thresholds.
func TestMaxRequestBodySizeGlobalEnforced(t *testing.T) {
	const (
		globalLimit = 10 * 1024 * 1024 // 10 MB
		bodySize    = 11 * 1024 * 1024 // 11 MB — exceeds global limit, no per-route override
	)

	router := gin.New()
	router.Use(MaxRequestBodySize(globalLimit))

	router.POST("/upload", func(c *gin.Context) {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	body := strings.NewReader(strings.Repeat("x", bodySize))
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("expected a non-200 response for %d-byte body exceeding the %d-byte global limit",
			bodySize, globalLimit)
	}
}
