# Logging Best Practices Guide

This document describes the structured logging approach used in the go-volunteer-media application.

## Overview

The application uses a custom structured logger (`internal/logging`) that provides:

- **JSON-formatted output** for machine parsing
- **Multiple log levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Context-aware logging** with automatic request ID and user ID tracking
- **Audit logging** for security-sensitive events
- **Integration with middleware** for automatic HTTP request logging

## Using the Logger

### Basic Usage

```go
import "github.com/networkengineer-cloud/go-volunteer-media/internal/logging"

// Simple log messages
logging.Info("Server started successfully")
logging.Warn("Configuration value missing, using default")
logging.Error("Database connection failed", err)
logging.Fatal("Critical error occurred", err) // Exits with status 1
```

### Structured Logging with Fields

```go
// Add structured fields for better querying
logger := logging.WithFields(map[string]interface{}{
    "user_id": 123,
    "action": "file_upload",
    "file_size": 1024000,
})
logger.Info("File uploaded successfully")
```

### In HTTP Handlers (Gin)

```go
import "github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"

func MyHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get logger with automatic request context (request_id, user_id)
        logger := middleware.GetLogger(c)
        
        logger.Info("Processing request")
        
        // Add additional fields
        logger.WithField("group_id", groupID).Debug("Fetching group data")
        
        if err != nil {
            logger.Error("Failed to process request", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Processing failed"})
            return
        }
    }
}
```

### Context-Aware Logging

```go
// Extract context information automatically
logger := logging.WithContext(ctx)
logger.Info("Background task started")
```

## Log Levels

Choose the appropriate log level:

| Level | When to Use | Example |
|-------|-------------|---------|
| **DEBUG** | Detailed diagnostic information, typically disabled in production | Image dimensions, query parameters |
| **INFO** | General informational messages about system operation | Server started, file uploaded, user registered |
| **WARN** | Warning messages for potentially harmful situations | Missing optional config, deprecated feature used |
| **ERROR** | Error events that might still allow the application to continue | Database query failed, email send failed |
| **FATAL** | Severe error events that cause the application to exit | Missing required config, unable to bind to port |

## Security: What NOT to Log

**NEVER log sensitive information:**

❌ Passwords (plain text or hashed)
❌ Authentication tokens (JWT, API keys, session tokens)
❌ Credit card numbers or payment information
❌ Social security numbers or government IDs
❌ Private keys or secrets
❌ Full email addresses in non-audit logs (use masking)

### Safe Logging Examples

```go
// ❌ BAD - Logs sensitive data
logging.Info("User logged in with password: " + password)
logging.Info("Generated JWT token: " + token)
logging.WithField("email", user.Email).Error("Failed to send email")

// ✅ GOOD - No sensitive data
logging.Info("User login attempt")
logging.Info("JWT token generated successfully")
logging.Error("Failed to send email to user", err) // Don't include email
```

### Audit Logging

For security-sensitive events, use the dedicated audit logging functions:

```go
import "github.com/networkengineer-cloud/go-volunteer-media/internal/logging"

// Audit logs include necessary PII for security investigations
logging.LogAuthSuccess(ctx, userID, username, clientIP)
logging.LogAuthFailure(ctx, username, clientIP, "invalid_password")
logging.LogAccountLocked(ctx, userID, username, clientIP, attemptCount)
logging.LogPasswordResetRequest(ctx, email, clientIP)
logging.LogRegistration(ctx, userID, username, email, clientIP)
```

Audit logs are appropriate for:
- Authentication events (login, logout, failed attempts)
- Authorization changes (role promotions, access grants)
- Data modifications (create, update, delete of sensitive resources)
- Security events (account lockouts, suspicious activity)

## Configuration

Logging is configured via environment variables:

```bash
# Log level: DEBUG, INFO, WARN, ERROR (default: INFO)
LOG_LEVEL=INFO

# Log format: json or text (default: json)
LOG_FORMAT=json
```

### Development vs Production

**Development:**
```bash
LOG_LEVEL=DEBUG
LOG_FORMAT=text  # Easier to read in console
```

**Production:**
```bash
LOG_LEVEL=INFO
LOG_FORMAT=json  # Required for log aggregation systems
```

## Log Output Format

### JSON Format (Production)

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "message": "Request completed successfully",
  "request_id": "abc123-def456",
  "user_id": "42",
  "method": "POST",
  "path": "/api/animals",
  "status": 201,
  "latency_ms": 125,
  "file": "/app/internal/handlers/animal.go",
  "line": 156
}
```

### Text Format (Development)

```
[2024-01-15T10:30:45Z] INFO: Request completed successfully map[request_id:abc123-def456 user_id:42 method:POST path:/api/animals status:201 latency_ms:125]
```

## Best Practices

### 1. Use Appropriate Log Levels

```go
// ✅ GOOD
logger.Debug("Query parameters parsed") // Diagnostic info
logger.Info("User registration successful") // Important event
logger.Warn("Email service unavailable, notifications disabled") // Degraded functionality
logger.Error("Database connection lost", err) // Error but recoverable
logger.Fatal("Required environment variable missing", nil) // Unrecoverable

// ❌ BAD
logger.Error("User clicked submit button", nil) // Not an error
logger.Info("Database query failed", err) // Should be ERROR
```

### 2. Add Context with Structured Fields

```go
// ✅ GOOD - Structured fields enable filtering and analysis
logger.WithFields(map[string]interface{}{
    "file_size": fileSize,
    "file_type": fileType,
    "upload_path": uploadPath,
}).Info("File upload completed")

// ❌ BAD - String interpolation loses structure
logger.Info(fmt.Sprintf("Uploaded %d byte %s file to %s", fileSize, fileType, uploadPath))
```

### 3. Log Errors Properly

```go
// ✅ GOOD - Provides error context
if err := db.Create(&user).Error; err != nil {
    logger.Error("Failed to create user", err)
    return err
}

// ❌ BAD - No error context
if err := db.Create(&user).Error; err != nil {
    logger.Error("Database error", nil)
    return err
}

// ❌ WORSE - Ignoring the error
if err := db.Create(&user).Error; err != nil {
    logger.Info("Something went wrong")
}
```

### 4. Don't Over-Log or Under-Log

```go
// ❌ Over-logging (creates noise)
logger.Debug("Entering function MyHandler")
logger.Debug("Parsing request body")
logger.Debug("Validating input")
logger.Debug("Calling database")
logger.Debug("Exiting function MyHandler")

// ✅ Appropriate logging
logger.Info("Processing user registration")
if err != nil {
    logger.Error("Registration failed", err)
}
logger.Info("User registered successfully")
```

### 5. Include Request IDs for Tracing

Request IDs are automatically added by the middleware, but you can access them:

```go
requestID, _ := c.Get("request_id")
logger.WithField("request_id", requestID).Info("Starting background job")
```

## Monitoring and Alerting

### Log Aggregation

In production, logs should be:
- Collected from all instances
- Centralized in a log aggregation system (ELK, Splunk, Datadog, etc.)
- Indexed for fast searching
- Retained according to compliance requirements

### Key Metrics to Monitor

Monitor these log patterns for alerting:

```json
// High error rate
{"level": "ERROR", "message": "Database connection failed"}

// Authentication failures (potential brute force)
{"audit_event": "login_failure", "ip": "1.2.3.4"}

// Account lockouts (security event)
{"audit_event": "account_locked", "failed_attempts": 5}

// Rate limit violations (potential abuse)
{"audit_event": "rate_limit_exceeded", "endpoint": "/api/login"}
```

### Sample Alert Rules

1. **High Error Rate**: Alert if ERROR logs exceed 10 per minute
2. **Failed Logins**: Alert if same IP has >5 failed logins in 5 minutes
3. **Account Lockouts**: Alert immediately on any account lockout
4. **Service Degradation**: Alert if WARN logs about service unavailability persist >5 minutes

## Debugging with Logs

### Finding Requests by ID

All logs for a single request share the same `request_id`:

```bash
# Find all logs for a specific request
jq 'select(.request_id == "abc123-def456")' application.log
```

### Finding User Activity

```bash
# Find all actions by a specific user
jq 'select(.user_id == "42")' application.log
```

### Finding Errors

```bash
# Find all errors in the last hour
jq 'select(.level == "ERROR")' application.log | tail -n 100
```

## Testing Logging

When writing tests, you can capture and verify log output:

```go
// Create a logger that writes to a buffer for testing
var buf bytes.Buffer
testLogger := logging.New(logging.INFO, &buf, true)
logging.SetDefaultLogger(testLogger)

// Run your code that logs
myFunction()

// Verify logs
logOutput := buf.String()
if !strings.Contains(logOutput, "expected log message") {
    t.Error("Expected log message not found")
}
```

## Migration Notes

We migrated from the standard library `log` package to structured logging:

### Before (Standard Library)

```go
log.Printf("User %d uploaded file %s", userID, filename)
log.Println("Database connected")
log.Fatal("Failed to start server")
```

### After (Structured Logger)

```go
logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "filename": filename,
}).Info("User uploaded file")

logging.Info("Database connected")
logging.Fatal("Failed to start server", err)
```

## Resources

- [Twelve-Factor App: Logs](https://12factor.net/logs)
- [Structured Logging Best Practices](https://www.dataset.com/blog/the-10-commandments-of-logging/)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)

## Support

For questions or issues with logging:
1. Check this guide first
2. Review examples in `internal/logging/logger.go` and `internal/logging/audit.go`
3. Look at existing handler implementations for patterns
4. Open an issue if you discover a logging security concern
