# Security and Observability Agent - Usage Examples

This document demonstrates practical usage of the Security and Observability Expert Agent for common scenarios in the go-volunteer-media repository.

## Table of Contents

1. [Security Review Scenarios](#security-review-scenarios)
2. [Observability Implementation Scenarios](#observability-implementation-scenarios)
3. [Code Review Examples](#code-review-examples)
4. [Enhancement Recommendations](#enhancement-recommendations)

---

## Security Review Scenarios

### Scenario 1: Authentication Handler Security Audit

**Task:** Review the authentication system for vulnerabilities

**Agent Invocation:**
```
@security-observability-expert Please conduct a comprehensive security audit of 
the authentication handlers in internal/handlers/auth.go and internal/auth/auth.go
```

**Expected Agent Analysis:**

```markdown
## Security Audit: Authentication System

### ✅ Strengths Identified

1. **Password Hashing**
   - Location: `internal/auth/auth.go:34-37`
   - Uses bcrypt with DefaultCost (currently 10)
   - Status: ✅ SECURE
   
2. **JWT Secret Validation**
   - Location: `internal/auth/auth.go:16-23`
   - Enforces minimum 32 characters
   - Status: ✅ SECURE with enhancement opportunity

3. **Account Lockout**
   - Location: `internal/handlers/auth.go:98-111`
   - Implements 5 failed attempts lockout
   - 30-minute lockout period
   - Status: ✅ SECURE

### 🟡 Enhancement Opportunities

1. **JWT Secret Entropy Validation**
   - Priority: MEDIUM
   - Current: Only checks length (32 chars)
   - Issue: "11111111111111111111111111111111" would pass
   - Recommendation: Add entropy checking or provide generation utility
   
2. **Token Expiration**
   - Priority: LOW
   - Current: 24-hour expiration
   - Consider: Shorter expiration with refresh token mechanism
   
3. **Rate Limiting**
   - Priority: HIGH
   - Current: Account-based limiting only
   - Recommendation: Add IP-based rate limiting to prevent distributed attacks

### 🔴 Critical Findings

None - No critical vulnerabilities detected

### Recommended Actions

1. Implement JWT secret generation utility
2. Add request-level rate limiting middleware
3. Consider refresh token implementation
4. Add authentication event logging
```

---

### Scenario 2: API Endpoint Security Review

**Task:** Review a new API endpoint before deployment

**Agent Invocation:**
```
@security-observability-expert Review the new animal image upload endpoint 
in internal/handlers/animal.go for security vulnerabilities
```

**Expected Agent Checklist:**

```markdown
## API Security Review: Animal Image Upload

### Input Validation ✅
- [x] File size limits implemented
- [x] File type validation (MIME type checking)
- [x] Filename sanitization
- [ ] MISSING: File content validation (magic bytes)
- [ ] MISSING: Virus/malware scanning

### Authorization ✅
- [x] Authentication required (middleware)
- [x] Group membership verified
- [x] User permissions checked

### Storage Security ⚠️
- [x] Files stored outside web root
- [x] Unique filenames generated (UUID)
- [ ] MISSING: File size quotas per user
- [ ] MISSING: Old file cleanup policy

### Path Traversal Protection ✅
- [x] UUID-based filenames prevent traversal
- [x] No user input in file paths

### Rate Limiting ⚠️
- [ ] MISSING: Upload rate limiting
- [ ] MISSING: Storage quota enforcement

### Recommendations (Priority Order)

1. **HIGH**: Add file content validation (magic bytes)
2. **HIGH**: Implement upload rate limiting
3. **MEDIUM**: Add storage quotas
4. **MEDIUM**: Implement automatic cleanup of old files
5. **LOW**: Consider adding virus scanning for production
```

---

## Observability Implementation Scenarios

### Scenario 3: Structured Logging Implementation

**Task:** Implement structured JSON logging throughout the application

**Agent Invocation:**
```
@security-observability-expert Help me implement structured logging with 
request correlation IDs across the application
```

**Expected Agent Guidance:**

```markdown
## Structured Logging Implementation Plan

### Phase 1: Choose Logging Library

Recommendation: Use `zerolog` for high-performance structured logging

```go
// go.mod
require github.com/rs/zerolog v1.31.0
```

### Phase 2: Create Logger Package

```go
// internal/logging/logger.go
package logging

import (
    "context"
    "os"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func InitLogger() {
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    
    if os.Getenv("ENV") == "production" {
        // JSON output for production
        log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
    } else {
        // Pretty console output for development
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    }
}

type contextKey string

const requestIDKey contextKey = "request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
    if reqID, ok := ctx.Value(requestIDKey).(string); ok {
        return reqID
    }
    return ""
}

func LogFromContext(ctx context.Context) *zerolog.Logger {
    logger := log.Logger
    if reqID := GetRequestID(ctx); reqID != "" {
        logger = logger.With().Str("request_id", reqID).Logger()
    }
    return &logger
}
```

### Phase 3: Request ID Middleware

```go
// internal/middleware/requestid.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        // Set in context
        ctx := logging.WithRequestID(c.Request.Context(), requestID)
        c.Request = c.Request.WithContext(ctx)
        
        // Add to response headers
        c.Header("X-Request-ID", requestID)
        
        c.Next()
    }
}
```

### Phase 4: Update Main Application

```go
// cmd/api/main.go
func main() {
    logging.InitLogger()
    
    router := gin.Default()
    
    // Add request ID middleware FIRST
    router.Use(middleware.RequestID())
    router.Use(middleware.CORS())
    
    // ... rest of setup
}
```

### Phase 5: Update Handlers

```go
// Example: internal/handlers/auth.go
func Login(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        logger := logging.LogFromContext(c.Request.Context())
        
        var req LoginRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            logger.Error().Err(err).Msg("Invalid login request")
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        
        logger.Info().
            Str("username", req.Username).
            Msg("Login attempt")
        
        // ... rest of handler
    }
}
```

### Implementation Checklist

- [ ] Install zerolog dependency
- [ ] Create logging package
- [ ] Implement request ID middleware
- [ ] Update main.go to initialize logger
- [ ] Update all handlers to use context-aware logging
- [ ] Update middleware to use structured logging
- [ ] Update database package to log with context
- [ ] Remove all fmt.Println and log.Println calls
- [ ] Add structured logging to error cases
- [ ] Ensure no sensitive data in logs

### Benefits

1. **Correlation**: All logs for a request have the same request_id
2. **Searchability**: JSON logs are easily parsed and searched
3. **Performance**: Zerolog is zero-allocation
4. **Consistency**: Structured format across all components
```

---

### Scenario 4: Metrics and Health Checks

**Task:** Add Prometheus metrics and health check endpoints

**Agent Invocation:**
```
@security-observability-expert Implement Prometheus metrics collection and 
health check endpoints for Kubernetes deployment
```

**Expected Agent Implementation:**

```markdown
## Metrics and Health Checks Implementation

### Step 1: Add Dependencies

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### Step 2: Create Metrics Package

```go
// internal/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP metrics
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    // Authentication metrics
    LoginAttemptsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "login_attempts_total",
            Help: "Total number of login attempts",
        },
        []string{"status"}, // success, failed, locked
    )
    
    // Database metrics
    DBConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "db_connections_active",
            Help: "Number of active database connections",
        },
    )
)
```

### Step 3: Metrics Middleware

```go
// internal/middleware/metrics.go
package middleware

import (
    "strconv"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/networkengineer-cloud/go-volunteer-media/internal/metrics"
)

func Metrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        
        metrics.HTTPRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            status,
        ).Inc()
        
        metrics.HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}
```

### Step 4: Health Check Handlers

```go
// internal/handlers/health.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

// HealthCheck returns basic health status
func HealthCheck() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "version": "1.0.0",
        })
    }
}

// ReadinessCheck checks if app is ready to serve traffic
func ReadinessCheck(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Check database connection
        sqlDB, err := db.DB()
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "status": "not ready",
                "reason": "database connection error",
            })
            return
        }
        
        if err := sqlDB.Ping(); err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "status": "not ready",
                "reason": "database ping failed",
            })
            return
        }
        
        c.JSON(http.StatusOK, gin.H{
            "status": "ready",
            "database": "connected",
        })
    }
}
```

### Step 5: Update Main Application

```go
// cmd/api/main.go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // ... existing setup ...
    
    router := gin.Default()
    
    // Add metrics middleware
    router.Use(middleware.Metrics())
    router.Use(middleware.RequestID())
    router.Use(middleware.CORS())
    
    // Health check endpoints
    router.GET("/health", handlers.HealthCheck())
    router.GET("/healthz", handlers.HealthCheck()) // Kubernetes style
    router.GET("/ready", handlers.ReadinessCheck(db))
    router.GET("/readyz", handlers.ReadinessCheck(db)) // Kubernetes style
    
    // Metrics endpoint
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))
    
    // ... rest of routes ...
}
```

### Step 6: Kubernetes Configuration

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: volunteer-media
spec:
  template:
    spec:
      containers:
      - name: api
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Step 7: Prometheus Scraping Configuration

```yaml
# prometheus.yaml
scrape_configs:
  - job_name: 'volunteer-media'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Verification Checklist

- [ ] Metrics endpoint accessible at /metrics
- [ ] Health check returns 200 OK
- [ ] Readiness check validates database connection
- [ ] HTTP requests tracked with method/endpoint/status
- [ ] Request duration histogram populated
- [ ] Login attempt metrics collected
- [ ] Prometheus can scrape metrics
- [ ] Kubernetes probes configured
```

---

## Code Review Examples

### Example 1: Security-Focused PR Review

**PR Description:** "Add user profile update endpoint"

**Agent Review Request:**
```
@security-observability-expert Please review PR #123 focusing on security aspects
```

**Expected Agent Response:**

```markdown
## Security Review: User Profile Update Endpoint

### Overview
Reviewing new endpoint: `PUT /api/users/:id/profile`

### 🔴 CRITICAL Issues

**1. Authorization Bypass Vulnerability**
- Location: `handlers/user.go:45-50`
- Issue: User can update any profile by changing :id parameter
- Impact: Any authenticated user can modify other users' profiles

```go
// VULNERABLE CODE
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.Param("id")
        var updates ProfileUpdate
        // ... updates applied without checking if user owns profile
    }
}

// FIXED CODE
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        requestedUserID := c.Param("id")
        currentUserID := c.GetUint("user_id") // from auth middleware
        
        // Users can only update their own profile (unless admin)
        isAdmin := c.GetBool("is_admin")
        if requestedUserID != fmt.Sprint(currentUserID) && !isAdmin {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Cannot update other users' profiles",
            })
            return
        }
        // ... proceed with update
    }
}
```

### 🟡 HIGH Priority Issues

**1. Input Validation Missing**
- Location: `handlers/user.go:52-55`
- Issue: No validation on bio length, email format, etc.
- Recommendation: Add validation struct tags

```go
type ProfileUpdate struct {
    Bio         string `json:"bio" binding:"max=500"`
    DisplayName string `json:"display_name" binding:"required,min=2,max=50"`
    Email       string `json:"email" binding:"required,email"`
}
```

**2. Email Change Without Verification**
- Location: `handlers/user.go:60`
- Issue: Users can change email without verification
- Recommendation: Implement email verification flow

### ✅ Security Strengths

1. Authentication middleware properly applied
2. HTTPS enforced in production
3. Rate limiting exists on parent route group

### Required Changes Before Merge

1. **MUST FIX**: Add authorization check (user can only update own profile)
2. **MUST FIX**: Add input validation
3. **SHOULD FIX**: Implement email verification for email changes
4. **SHOULD ADD**: Log profile update events for audit trail

### Approval Status
❌ **BLOCKED** - Critical authorization issue must be fixed
```

---

## Enhancement Recommendations

### Current State Analysis

**Agent Invocation:**
```
@security-observability-expert Analyze the current security and observability 
state of the application and provide prioritized enhancement recommendations
```

**Expected Agent Report:**

```markdown
## Security and Observability Enhancement Roadmap

### Current State Assessment

#### Security: 7/10 ⭐⭐⭐⭐⭐⭐⭐☆☆☆

**Strengths:**
- Strong authentication foundation
- Password security implemented correctly
- Account lockout prevents brute force
- Docker security best practices followed

**Gaps:**
- No request rate limiting
- Missing security headers
- No audit logging
- Limited monitoring

#### Observability: 4/10 ⭐⭐⭐⭐☆☆☆☆☆☆

**Strengths:**
- Basic logging present
- Graceful shutdown implemented
- Database logging enabled

**Gaps:**
- No structured logging
- No request correlation
- No metrics collection
- No distributed tracing
- No health check endpoints

---

### Enhancement Roadmap (Priority Order)

#### Phase 1: Critical Security (1-2 weeks)

**1. Implement Rate Limiting** 🔴 CRITICAL
- Priority: CRITICAL
- Effort: Medium
- Impact: High
- Details: Prevent DoS and brute force attacks

```go
// Recommendation: Use gin rate limiter
import "github.com/ulule/limiter/v3"

// Apply to all routes
router.Use(rateLimit.NewMiddleware())

// Stricter limits on auth endpoints
authRoutes.Use(rateLimit.NewMiddleware(
    rateLimit.WithLimit(5, 15*time.Minute),
))
```

**2. Add Security Headers** 🔴 CRITICAL
- Priority: CRITICAL
- Effort: Low
- Impact: High

```go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'")
        if c.Request.TLS != nil {
            c.Header("Strict-Transport-Security", 
                "max-age=31536000; includeSubDomains")
        }
        c.Next()
    }
}
```

#### Phase 2: Enhanced Observability (2-3 weeks)

**3. Structured Logging with Request IDs** 🟡 HIGH
- Priority: HIGH
- Effort: Medium
- Impact: High
- See detailed implementation in Scenario 3 above

**4. Metrics Collection** 🟡 HIGH
- Priority: HIGH
- Effort: Medium
- Impact: High
- See detailed implementation in Scenario 4 above

**5. Health Check Endpoints** 🟡 HIGH
- Priority: HIGH
- Effort: Low
- Impact: Medium
- Required for Kubernetes deployments

#### Phase 3: Advanced Security (3-4 weeks)

**6. Audit Logging** 🟢 MEDIUM
- Priority: MEDIUM
- Effort: Medium
- Impact: Medium

```go
func AuditLog(db *gorm.DB, action, resource string, userID uint) {
    entry := AuditLogEntry{
        Timestamp: time.Now(),
        UserID:    userID,
        Action:    action,
        Resource:  resource,
        IPAddress: getClientIP(),
    }
    db.Create(&entry)
}
```

**7. Secret Rotation** 🟢 MEDIUM
- Priority: MEDIUM
- Effort: High
- Impact: Medium
- Implement JWT secret rotation mechanism
- Document rotation procedures

**8. File Upload Security Enhancement** 🟢 LOW
- Priority: LOW
- Effort: Medium
- Impact: Low
- Add magic byte validation
- Implement virus scanning
- Add storage quotas

#### Phase 4: Advanced Observability (4-6 weeks)

**9. Distributed Tracing** 🟢 LOW
- Priority: LOW
- Effort: High
- Impact: Medium
- Implement OpenTelemetry
- Add Jaeger/Zipkin integration

**10. Advanced Metrics** 🟢 LOW
- Priority: LOW
- Effort: Medium
- Impact: Medium
- Business metrics dashboard
- User behavior analytics
- Performance profiling

---

### Implementation Timeline

```
Week 1-2:  Rate Limiting + Security Headers
Week 3-4:  Structured Logging + Request IDs
Week 5-6:  Metrics Collection + Health Checks
Week 7-8:  Audit Logging
Week 9-12: Advanced features (as needed)
```

### Success Metrics

After implementation, we should achieve:

- **Security Score: 9/10** ⭐⭐⭐⭐⭐⭐⭐⭐⭐☆
- **Observability Score: 8/10** ⭐⭐⭐⭐⭐⭐⭐⭐☆☆
- **Incident Response Time:** < 5 minutes
- **Mean Time to Detect (MTTD):** < 2 minutes
- **Mean Time to Resolve (MTTR):** < 30 minutes
```

---

## Summary

This document demonstrates how the Security and Observability Expert Agent can be used for:

1. **Security Audits**: Comprehensive code and configuration reviews
2. **Implementation Guidance**: Step-by-step implementation of security and observability features
3. **Code Reviews**: Security-focused PR reviews with actionable feedback
4. **Strategic Planning**: Prioritized enhancement roadmaps

The agent provides:
- ✅ Specific, actionable recommendations
- ✅ Complete code examples
- ✅ Priority and impact assessments
- ✅ Implementation checklists
- ✅ Best practices and patterns
- ✅ Security vulnerability identification
- ✅ Observability implementation guidance

Use the agent proactively for security reviews and observability implementations to maintain high standards throughout the development lifecycle.
