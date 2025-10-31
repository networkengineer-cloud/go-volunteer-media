package logging

import (
	"context"
	"fmt"
)

// AuditEvent represents different types of security events
type AuditEvent string

const (
	// Authentication events
	AuditEventLoginSuccess        AuditEvent = "login_success"
	AuditEventLoginFailure        AuditEvent = "login_failure"
	AuditEventAccountLocked       AuditEvent = "account_locked"
	AuditEventPasswordResetRequest AuditEvent = "password_reset_request"
	AuditEventPasswordResetSuccess AuditEvent = "password_reset_success"
	AuditEventRegistration        AuditEvent = "user_registration"
	
	// Admin events
	AuditEventUserCreated         AuditEvent = "user_created"
	AuditEventUserDeleted         AuditEvent = "user_deleted"
	AuditEventUserRestored        AuditEvent = "user_restored"
	AuditEventUserPromoted        AuditEvent = "user_promoted"
	AuditEventUserDemoted         AuditEvent = "user_demoted"
	AuditEventGroupCreated        AuditEvent = "group_created"
	AuditEventGroupUpdated        AuditEvent = "group_updated"
	AuditEventGroupDeleted        AuditEvent = "group_deleted"
	AuditEventUserAddedToGroup    AuditEvent = "user_added_to_group"
	AuditEventUserRemovedFromGroup AuditEvent = "user_removed_from_group"
	
	// Data events
	AuditEventAnimalCreated       AuditEvent = "animal_created"
	AuditEventAnimalUpdated       AuditEvent = "animal_updated"
	AuditEventAnimalDeleted       AuditEvent = "animal_deleted"
	AuditEventAnnouncementCreated AuditEvent = "announcement_created"
	AuditEventAnnouncementDeleted AuditEvent = "announcement_deleted"
	AuditEventImageUploaded       AuditEvent = "image_uploaded"
	
	// Security events
	AuditEventRateLimitExceeded   AuditEvent = "rate_limit_exceeded"
	AuditEventUnauthorizedAccess  AuditEvent = "unauthorized_access"
	AuditEventInvalidToken        AuditEvent = "invalid_token"
	AuditEventFileUploadRejected  AuditEvent = "file_upload_rejected"
)

// AuditLogger provides structured audit logging
type AuditLogger struct {
	// logger field removed - we'll use defaultLogger directly to avoid nil issues
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// getLogger returns the current default logger
func (al *AuditLogger) getLogger() *Logger {
	return defaultLogger
}

// Log logs an audit event
func (al *AuditLogger) Log(ctx context.Context, event AuditEvent, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	// Add audit event type
	fields["audit_event"] = string(event)
	fields["event_category"] = "audit"
	
	// Create logger with context and fields using current defaultLogger
	logger := al.getLogger().WithContext(ctx).WithFields(fields)
	
	// Log with appropriate message
	message := fmt.Sprintf("Audit: %s", event)
	logger.Info(message)
}

// LogAuthSuccess logs successful authentication
func (al *AuditLogger) LogAuthSuccess(ctx context.Context, userID uint, username, ip string) {
	al.Log(ctx, AuditEventLoginSuccess, map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"ip":       ip,
	})
}

// LogAuthFailure logs failed authentication attempt
func (al *AuditLogger) LogAuthFailure(ctx context.Context, username, ip, reason string) {
	al.Log(ctx, AuditEventLoginFailure, map[string]interface{}{
		"username": username,
		"ip":       ip,
		"reason":   reason,
	})
}

// LogAccountLocked logs account lockout event
func (al *AuditLogger) LogAccountLocked(ctx context.Context, userID uint, username, ip string, attempts int) {
	al.Log(ctx, AuditEventAccountLocked, map[string]interface{}{
		"user_id":        userID,
		"username":       username,
		"ip":             ip,
		"failed_attempts": attempts,
	})
}

// LogPasswordResetRequest logs password reset request
func (al *AuditLogger) LogPasswordResetRequest(ctx context.Context, email, ip string) {
	al.Log(ctx, AuditEventPasswordResetRequest, map[string]interface{}{
		"email": email,
		"ip":    ip,
	})
}

// LogPasswordResetSuccess logs successful password reset
func (al *AuditLogger) LogPasswordResetSuccess(ctx context.Context, userID uint, ip string) {
	al.Log(ctx, AuditEventPasswordResetSuccess, map[string]interface{}{
		"user_id": userID,
		"ip":      ip,
	})
}

// LogRegistration logs new user registration
func (al *AuditLogger) LogRegistration(ctx context.Context, userID uint, username, email, ip string) {
	al.Log(ctx, AuditEventRegistration, map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"email":    email,
		"ip":       ip,
	})
}

// LogAdminAction logs administrative actions
func (al *AuditLogger) LogAdminAction(ctx context.Context, event AuditEvent, adminID uint, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["admin_id"] = adminID
	fields["action_type"] = "admin"
	al.Log(ctx, event, fields)
}

// LogRateLimitExceeded logs rate limit violations
func (al *AuditLogger) LogRateLimitExceeded(ctx context.Context, ip, endpoint string) {
	al.Log(ctx, AuditEventRateLimitExceeded, map[string]interface{}{
		"ip":       ip,
		"endpoint": endpoint,
	})
}

// LogUnauthorizedAccess logs unauthorized access attempts
func (al *AuditLogger) LogUnauthorizedAccess(ctx context.Context, ip, endpoint, reason string) {
	al.Log(ctx, AuditEventUnauthorizedAccess, map[string]interface{}{
		"ip":       ip,
		"endpoint": endpoint,
		"reason":   reason,
	})
}

// LogFileUploadRejected logs rejected file uploads
func (al *AuditLogger) LogFileUploadRejected(ctx context.Context, userID uint, filename, reason string) {
	al.Log(ctx, AuditEventFileUploadRejected, map[string]interface{}{
		"user_id":  userID,
		"filename": filename,
		"reason":   reason,
	})
}

// Default audit logger instance
var defaultAuditLogger = NewAuditLogger()

// Package-level convenience functions

// LogAuthSuccess logs successful authentication using default audit logger
func LogAuthSuccess(ctx context.Context, userID uint, username, ip string) {
	defaultAuditLogger.LogAuthSuccess(ctx, userID, username, ip)
}

// LogAuthFailure logs failed authentication using default audit logger
func LogAuthFailure(ctx context.Context, username, ip, reason string) {
	defaultAuditLogger.LogAuthFailure(ctx, username, ip, reason)
}

// LogAccountLocked logs account lockout using default audit logger
func LogAccountLocked(ctx context.Context, userID uint, username, ip string, attempts int) {
	defaultAuditLogger.LogAccountLocked(ctx, userID, username, ip, attempts)
}

// LogPasswordResetRequest logs password reset request using default audit logger
func LogPasswordResetRequest(ctx context.Context, email, ip string) {
	defaultAuditLogger.LogPasswordResetRequest(ctx, email, ip)
}

// LogPasswordResetSuccess logs successful password reset using default audit logger
func LogPasswordResetSuccess(ctx context.Context, userID uint, ip string) {
	defaultAuditLogger.LogPasswordResetSuccess(ctx, userID, ip)
}

// LogRegistration logs new user registration using default audit logger
func LogRegistration(ctx context.Context, userID uint, username, email, ip string) {
	defaultAuditLogger.LogRegistration(ctx, userID, username, email, ip)
}

// LogAdminAction logs administrative actions using default audit logger
func LogAdminAction(ctx context.Context, event AuditEvent, adminID uint, fields map[string]interface{}) {
	defaultAuditLogger.LogAdminAction(ctx, event, adminID, fields)
}

// LogRateLimitExceeded logs rate limit violations using default audit logger
func LogRateLimitExceeded(ctx context.Context, ip, endpoint string) {
	defaultAuditLogger.LogRateLimitExceeded(ctx, ip, endpoint)
}

// LogUnauthorizedAccess logs unauthorized access attempts using default audit logger
func LogUnauthorizedAccess(ctx context.Context, ip, endpoint, reason string) {
	defaultAuditLogger.LogUnauthorizedAccess(ctx, ip, endpoint, reason)
}

// LogFileUploadRejected logs rejected file uploads using default audit logger
func LogFileUploadRejected(ctx context.Context, userID uint, filename, reason string) {
	defaultAuditLogger.LogFileUploadRejected(ctx, userID, filename, reason)
}
