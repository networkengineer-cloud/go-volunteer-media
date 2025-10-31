package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// Level represents log level
type Level int

const (
	// DEBUG level for detailed debugging information
	DEBUG Level = iota
	// INFO level for general informational messages
	INFO
	// WARN level for warning messages
	WARN
	// ERROR level for error messages
	ERROR
	// FATAL level for fatal errors (will exit)
	FATAL
)

// String returns string representation of log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	level      Level
	output     io.Writer
	jsonFormat bool
	fields     map[string]interface{}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Function  string                 `json:"function,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

var defaultLogger *Logger

func init() {
	// Initialize default logger
	defaultLogger = New(INFO, os.Stdout, true)
}

// New creates a new Logger instance
func New(level Level, output io.Writer, jsonFormat bool) *Logger {
	return &Logger{
		level:      level,
		output:     output,
		jsonFormat: jsonFormat,
		fields:     make(map[string]interface{}),
	}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:      l.level,
		output:     l.output,
		jsonFormat: l.jsonFormat,
		fields:     make(map[string]interface{}),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

// WithContext extracts common fields from context and returns a new logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make(map[string]interface{})

	// Extract request ID from context
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields["request_id"] = requestID
	}

	// Extract user ID from context
	if userID, ok := ctx.Value("user_id").(uint); ok {
		fields["user_id"] = fmt.Sprintf("%d", userID)
	}

	return l.WithFields(fields)
}

// log is the internal logging method
func (l *Logger) log(level Level, msg string, err error) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Fields:    l.fields,
	}

	// Add caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		entry.File = file
		entry.Line = line
		if fn := runtime.FuncForPC(pc); fn != nil {
			// Get just the function name without package path
			parts := strings.Split(fn.Name(), "/")
			entry.Function = parts[len(parts)-1]
		}
	}

	// Add error if provided
	if err != nil {
		entry.Error = err.Error()
	}

	// Extract request_id and user_id to top level for easier querying
	if reqID, ok := l.fields["request_id"].(string); ok {
		entry.RequestID = reqID
	}
	if userID, ok := l.fields["user_id"].(string); ok {
		entry.UserID = userID
	}

	var output string
	if l.jsonFormat {
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			// Fallback to simple format if JSON marshaling fails
			output = fmt.Sprintf("[%s] %s: %s", entry.Timestamp, entry.Level, entry.Message)
		} else {
			output = string(jsonBytes)
		}
	} else {
		// Simple text format
		output = fmt.Sprintf("[%s] %s: %s", entry.Timestamp, entry.Level, entry.Message)
		if len(entry.Fields) > 0 {
			output += fmt.Sprintf(" %v", entry.Fields)
		}
		if entry.Error != "" {
			output += fmt.Sprintf(" error=%s", entry.Error)
		}
	}

	fmt.Fprintln(l.output, output)

	// Exit on FATAL
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.log(DEBUG, msg, nil)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.log(INFO, msg, nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.log(WARN, msg, nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error) {
	l.log(ERROR, msg, err)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil)
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(msg string, err error) {
	l.log(FATAL, msg, err)
}

// Fatalf logs a formatted fatal error and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...), nil)
}

// Package-level convenience functions using default logger

// Debug logs a debug message using the default logger
func Debug(msg string) {
	defaultLogger.Debug(msg)
}

// Info logs an info message using the default logger
func Info(msg string) {
	defaultLogger.Info(msg)
}

// Warn logs a warning message using the default logger
func Warn(msg string) {
	defaultLogger.Warn(msg)
}

// Error logs an error message using the default logger
func Error(msg string, err error) {
	defaultLogger.Error(msg, err)
}

// Fatal logs a fatal error and exits using the default logger
func Fatal(msg string, err error) {
	defaultLogger.Fatal(msg, err)
}

// WithFields returns a new logger with fields using the default logger
func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

// WithField returns a new logger with a field using the default logger
func WithField(key string, value interface{}) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithContext returns a new logger with context using the default logger
func WithContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}

// SetLevel sets the log level for the default logger
func SetLevel(level Level) {
	defaultLogger.level = level
}

// GetDefaultLogger returns the default logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// SetDefaultLogger sets the default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// InitFromEnv initializes logging based on environment variables
func InitFromEnv() {
	// Check log level from environment
	levelStr := os.Getenv("LOG_LEVEL")
	var level Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	default:
		level = INFO
	}

	// Check log format from environment
	formatStr := os.Getenv("LOG_FORMAT")
	jsonFormat := true
	if strings.ToLower(formatStr) == "text" {
		jsonFormat = false
	}

	// Create new default logger with environment configuration
	defaultLogger = New(level, os.Stdout, jsonFormat)

	// Replace standard library logger with our logger
	log.SetOutput(&stdLogAdapter{logger: defaultLogger})
	log.SetFlags(0) // Remove default flags since we handle formatting
}

// stdLogAdapter adapts our Logger to work with standard library log
type stdLogAdapter struct {
	logger *Logger
}

func (a *stdLogAdapter) Write(p []byte) (n int, err error) {
	a.logger.Info(strings.TrimSpace(string(p)))
	return len(p), nil
}
