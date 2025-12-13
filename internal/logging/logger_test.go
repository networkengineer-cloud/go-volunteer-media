package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	if logger.level != INFO {
		t.Errorf("Expected level INFO, got %v", logger.level)
	}
	if logger.output != buf {
		t.Error("Expected output to be set")
	}
	if !logger.jsonFormat {
		t.Error("Expected jsonFormat to be true")
	}
	if logger.fields == nil {
		t.Error("Expected fields to be initialized")
	}
}

func TestLogger_WithField(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	newLogger := logger.WithField("key", "value")

	if val, ok := newLogger.fields["key"]; !ok || val != "value" {
		t.Error("Expected field to be set")
	}

	// Original logger should not have the field
	if _, ok := logger.fields["key"]; ok {
		t.Error("Original logger should not be modified")
	}
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	newLogger := logger.WithFields(fields)

	if val, ok := newLogger.fields["key1"]; !ok || val != "value1" {
		t.Error("Expected key1 to be set")
	}
	if val, ok := newLogger.fields["key2"]; !ok || val != 123 {
		t.Error("Expected key2 to be set")
	}
}

func TestLogger_WithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx = context.WithValue(ctx, "user_id", uint(456))

	newLogger := logger.WithContext(ctx)

	if val, ok := newLogger.fields["request_id"]; !ok || val != "req-123" {
		t.Error("Expected request_id to be extracted from context")
	}
	if val, ok := newLogger.fields["user_id"]; !ok || val != "456" {
		t.Error("Expected user_id to be extracted from context")
	}
}

func TestLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain level, got: %s", output)
	}

	// Verify JSON structure
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", entry.Message)
	}
}

func TestLogger_Debug_FilteredByLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, true)

	logger.Debug("debug message")

	// Debug should be filtered out since level is INFO
	if buf.Len() > 0 {
		t.Errorf("Expected no output for debug message, got: %s", buf.String())
	}
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(ERROR, buf, true)

	testErr := &testError{msg: "test error"}
	logger.Error("error occurred", testErr)

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("Expected output to contain error, got: %s", output)
	}

	// Verify JSON structure
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}
	if entry.Error != "test error" {
		t.Errorf("Expected error 'test error', got %s", entry.Error)
	}
}

func TestLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(WARN, buf, true)

	logger.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected output to contain level, got: %s", output)
	}
}

func TestLogger_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(INFO, buf, false) // Text format

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain level, got: %s", output)
	}

	// Should not be valid JSON
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err == nil {
		t.Error("Expected non-JSON output for text format")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Test package-level functions use default logger
	buf := &bytes.Buffer{}
	oldLogger := defaultLogger
	defaultLogger = New(INFO, buf, true)
	defer func() { defaultLogger = oldLogger }()

	Info("test info")

	if buf.Len() == 0 {
		t.Error("Expected output from package-level Info function")
	}
	if !strings.Contains(buf.String(), "test info") {
		t.Errorf("Expected output to contain message, got: %s", buf.String())
	}
}

func TestSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := defaultLogger
	defaultLogger = New(INFO, buf, true)
	defer func() { defaultLogger = oldLogger }()

	SetLevel(ERROR)

	if defaultLogger.level != ERROR {
		t.Errorf("Expected level ERROR, got %v", defaultLogger.level)
	}
}

func TestGetDefaultLogger(t *testing.T) {
	logger := GetDefaultLogger()
	if logger == nil {
		t.Error("Expected default logger to be non-nil")
	}
	if logger != defaultLogger {
		t.Error("Expected GetDefaultLogger to return defaultLogger")
	}
}

func TestSetDefaultLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	newLogger := New(DEBUG, buf, true)

	oldLogger := defaultLogger
	defer func() { defaultLogger = oldLogger }()

	SetDefaultLogger(newLogger)

	if defaultLogger != newLogger {
		t.Error("Expected default logger to be updated")
	}
}

// Helper type for testing errors
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestAuditLogger_NewAuditLogger(t *testing.T) {
	al := NewAuditLogger()
	if al == nil {
		t.Error("Expected audit logger to be non-nil")
	}
}

func TestAuditLogger_Log(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := defaultLogger
	defaultLogger = New(INFO, buf, true)
	defer func() { defaultLogger = oldLogger }()

	al := NewAuditLogger()
	ctx := context.Background()

	al.Log(ctx, AuditEventLoginSuccess, map[string]interface{}{
		"user_id":  uint(123),
		"username": "testuser",
	})

	output := buf.String()
	if !strings.Contains(output, "login_success") {
		t.Errorf("Expected output to contain event type, got: %s", output)
	}
	if !strings.Contains(output, "testuser") {
		t.Errorf("Expected output to contain username, got: %s", output)
	}
}

func TestAuditLogger_LogAuthSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := defaultLogger
	defaultLogger = New(INFO, buf, true)
	defer func() { defaultLogger = oldLogger }()

	al := NewAuditLogger()
	ctx := context.Background()

	al.LogAuthSuccess(ctx, 123, "testuser", "192.168.1.1")

	output := buf.String()
	if !strings.Contains(output, "login_success") {
		t.Errorf("Expected output to contain event type, got: %s", output)
	}
	if !strings.Contains(output, "testuser") {
		t.Errorf("Expected output to contain username, got: %s", output)
	}
	if !strings.Contains(output, "192.168.1.1") {
		t.Errorf("Expected output to contain IP, got: %s", output)
	}
}

func TestAuditLogger_LogAuthFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := defaultLogger
	defaultLogger = New(INFO, buf, true)
	defer func() { defaultLogger = oldLogger }()

	al := NewAuditLogger()
	ctx := context.Background()

	al.LogAuthFailure(ctx, "testuser", "192.168.1.1", "invalid password")

	output := buf.String()
	if !strings.Contains(output, "login_failure") {
		t.Errorf("Expected output to contain event type, got: %s", output)
	}
	if !strings.Contains(output, "invalid password") {
		t.Errorf("Expected output to contain reason, got: %s", output)
	}
}

func TestAuditEventConstants(t *testing.T) {
	// Test that audit event constants are defined correctly
	events := []AuditEvent{
		AuditEventLoginSuccess,
		AuditEventLoginFailure,
		AuditEventAccountLocked,
		AuditEventPasswordResetRequest,
		AuditEventPasswordResetSuccess,
		AuditEventRegistration,
		AuditEventUserCreated,
		AuditEventRateLimitExceeded,
		AuditEventUnauthorizedAccess,
	}

	for _, event := range events {
		if string(event) == "" {
			t.Errorf("Expected event %v to have non-empty string value", event)
		}
	}
}
