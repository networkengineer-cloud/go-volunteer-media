package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response helpers for standardized HTTP responses within the handlers package.
// Use these incrementally when touching a handler — do not mass-replace existing c.JSON calls.

func respondOK(c *gin.Context, data any)              { c.JSON(http.StatusOK, data) }
func respondCreated(c *gin.Context, data any)         { c.JSON(http.StatusCreated, data) }
func respondNoContent(c *gin.Context)                 { c.Status(http.StatusNoContent) }
func respondBadRequest(c *gin.Context, msg string)    { c.JSON(http.StatusBadRequest, gin.H{"error": msg}) }
func respondUnauthorized(c *gin.Context, msg string)  { c.JSON(http.StatusUnauthorized, gin.H{"error": msg}) }
func respondForbidden(c *gin.Context, msg string)     { c.JSON(http.StatusForbidden, gin.H{"error": msg}) }
func respondNotFound(c *gin.Context, msg string)      { c.JSON(http.StatusNotFound, gin.H{"error": msg}) }
func respondInternalError(c *gin.Context, msg string) { c.JSON(http.StatusInternalServerError, gin.H{"error": msg}) }
