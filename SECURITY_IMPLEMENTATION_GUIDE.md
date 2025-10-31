# Security Implementation Guide

This document provides a comprehensive guide to the security features implemented in go-volunteer-media and how to use them effectively.

## Table of Contents

1. [Security Overview](#security-overview)
2. [Quick Start](#quick-start)
3. [Authentication & Authorization](#authentication--authorization)
4. [File Upload Security](#file-upload-security)
5. [Rate Limiting](#rate-limiting)
6. [Structured Logging](#structured-logging)
7. [Audit Logging](#audit-logging)
8. [Security Headers](#security-headers)
9. [Monitoring & Health Checks](#monitoring--health-checks)
10. [Testing Security](#testing-security)
11. [Production Deployment](#production-deployment)

---

## Security Overview

The go-volunteer-media application implements multiple layers of security:

```
┌─────────────────────────────────────────────────────────┐
│                  Security Layers                        │
├─────────────────────────────────────────────────────────┤
│  1. Network Security (HTTPS, CORS, Security Headers)    │
│  2. Rate Limiting (Brute Force Protection)              │
│  3. Authentication (JWT, Account Lockout)               │
│  4. Authorization (Role-Based Access Control)           │
│  5. Input Validation (Request Validation, File Upload)  │
│  6. Data Protection (Encryption, Secure Storage)        │
│  7. Logging & Monitoring (Audit Trail, Health Checks)   │
└─────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/networkengineer-cloud/go-volunteer-media.git
   cd go-volunteer-media
   ```

2. **Copy environment variables**:
   ```bash
   cp .env.example .env
   ```

3. **Generate secure JWT secret**:
   ```bash
   openssl rand -base64 32
   ```
   Add this to `.env` as `JWT_SECRET`

4. **Start the development environment**:
   ```bash
   docker-compose up -d
   ```

5. **Run the application**:
   ```bash
   go run cmd/api/main.go
   ```

### Verify Security Features

```bash
# Check health endpoint
curl http://localhost:8080/health

# Test security headers
curl -I http://localhost:8080/health

# Test rate limiting (run multiple times)
for i in {1..10}; do 
  curl -X POST http://localhost:8080/api/login \
    -H "Content-Type: application/json" \
    -d '{"username":"test","password":"wrong"}'
done
```

---

## Authentication & Authorization

### JWT Token Authentication

The application uses JWT (JSON Web Tokens) for stateless authentication:

**Token Structure**:
```json
{
  "user_id": 1,
  "is_admin": false,
  "exp": 1730419179,
  "iat": 1730332779
}
```

**Token Lifetime**: 24 hours

**Example Usage**:
```bash
# Register a new user
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "email": "john@example.com",
    "password": "securepass123"
  }'

# Login and get token
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "password": "securepass123"
  }'

# Use token for authenticated requests
curl http://localhost:8080/api/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Password Security

- **Hashing**: bcrypt with default cost factor
- **Requirements**: 
  - Minimum 8 characters
  - Maximum 72 characters (bcrypt limit)
- **No special character requirements** (allows flexibility)

### Account Lockout

- **Threshold**: 5 failed login attempts
- **Lockout Duration**: 30 minutes
- **Automatic Reset**: After lockout period expires
- **Manual Reset**: Via password reset flow

**Example Response (Account Locked)**:
```json
{
  "error": "Account is temporarily locked due to too many failed login attempts",
  "locked_until": "2025-10-31T04:30:00Z",
  "retry_in_mins": 25
}
```

### Password Reset Flow

1. **Request Reset**:
   ```bash
   curl -X POST http://localhost:8080/api/request-password-reset \
     -H "Content-Type: application/json" \
     -d '{"email": "john@example.com"}'
   ```

2. **Check Email**: User receives reset link with token

3. **Reset Password**:
   ```bash
   curl -X POST http://localhost:8080/api/reset-password \
     -H "Content-Type: application/json" \
     -d '{
       "token": "RESET_TOKEN_FROM_EMAIL",
       "new_password": "newsecurepass123"
     }'
   ```

**Security Features**:
- Token expires after 1 hour
- Tokens are hashed before storage
- Email enumeration protection
- Rate limiting on reset endpoints

---

## File Upload Security

### Upload Validation

All file uploads are validated for:

1. **File Size**: 
   - Images: 10 MB max
   - Hero images: 5 MB max

2. **File Type**: 
   - Allowed: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`
   - Extension validation
   - MIME type verification
   - Magic number validation

3. **Content Validation**:
   - Image format verification
   - Prevents file type spoofing

### Usage Example

```bash
# Upload animal image (authenticated users)
curl -X POST http://localhost:8080/api/animals/upload-image \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "image=@/path/to/image.jpg"

# Response
{
  "url": "/uploads/1730419179_uuid.jpg"
}
```

### Error Handling

```json
{
  "error": "Invalid file: file size exceeds maximum limit: file size is 15728640 bytes, maximum is 10485760 bytes"
}
```

---

## Rate Limiting

### Configuration

Rate limiting protects authentication endpoints from brute force attacks:

- **Limit**: 5 requests per minute per IP
- **Window**: 1 minute rolling window
- **Algorithm**: Token bucket

### Protected Endpoints

- `/api/login` - Login attempts
- `/api/register` - User registration
- `/api/request-password-reset` - Password reset requests
- `/api/reset-password` - Password reset completion

### Response When Rate Limited

```json
{
  "error": "Too many requests. Please try again later."
}
```

**HTTP Status**: 429 Too Many Requests

### Testing Rate Limiting

```bash
# Test rate limit (will fail after 5 requests)
for i in {1..10}; do 
  echo "Request $i:"
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST http://localhost:8080/api/login \
    -H "Content-Type: application/json" \
    -d '{"username":"test","password":"wrong"}'
done
```

---

## Structured Logging

### Configuration

Set log level and format via environment variables:

```bash
# .env file
LOG_LEVEL=INFO    # DEBUG, INFO, WARN, ERROR
LOG_FORMAT=json   # json or text
```

### Log Levels

- **DEBUG**: Detailed debugging information
- **INFO**: General informational messages (default)
- **WARN**: Warning messages
- **ERROR**: Error messages
- **FATAL**: Fatal errors (exits application)

### Log Format

**JSON Format** (Production):
```json
{
  "timestamp": "2025-10-31T03:59:39Z",
  "level": "INFO",
  "message": "Request completed successfully",
  "fields": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "method": "GET",
    "path": "/api/groups",
    "status": 200,
    "latency_ms": 25,
    "ip": "192.168.1.100",
    "user_id": "1"
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "file": "middleware/logging.go",
  "line": 58
}
```

**Text Format** (Development):
```
[2025-10-31T03:59:39Z] INFO: Request completed successfully {"request_id":"550e8400...","method":"GET","status":200}
```

### Request Tracing

Every request gets a unique `request_id`:

```bash
# Request
curl -H "X-Request-ID: my-custom-id" http://localhost:8080/health

# Response includes same ID
X-Request-ID: my-custom-id
```

If not provided, a UUID is generated automatically.

---

## Audit Logging

### Security Events Tracked

1. **Authentication Events**:
   - Login success/failure
   - Account lockout
   - Password reset request/completion
   - User registration

2. **Admin Actions**:
   - User creation/deletion
   - User promotion/demotion
   - Group management
   - Permission changes

3. **Data Operations**:
   - Animal CRUD operations
   - Announcement creation/deletion
   - File uploads

4. **Security Violations**:
   - Rate limit exceeded
   - Unauthorized access attempts
   - Invalid token usage
   - File upload rejections

### Audit Log Format

```json
{
  "timestamp": "2025-10-31T03:59:39Z",
  "level": "INFO",
  "message": "Audit: login_success",
  "fields": {
    "audit_event": "login_success",
    "event_category": "audit",
    "user_id": 1,
    "username": "john",
    "ip": "192.168.1.100"
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Querying Audit Logs

```bash
# Find all login failures
cat logs/*.log | jq 'select(.fields.audit_event == "login_failure")'

# Find all admin actions
cat logs/*.log | jq 'select(.fields.action_type == "admin")'

# Find activity for specific user
cat logs/*.log | jq 'select(.fields.user_id == "1")'

# Find suspicious IP activity
cat logs/*.log | jq 'select(.fields.ip == "192.168.1.100")'
```

---

## Security Headers

### Headers Implemented

The application automatically adds these security headers to all responses:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; ...
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

### HSTS (HTTPS Only)

When running with HTTPS, uncomment this line in `internal/middleware/security.go`:

```go
c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
```

### Content Security Policy

Adjust CSP based on your frontend requirements:

```go
// Default (strict)
csp := "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; ..."

// Example: Allow Google Fonts
csp := "default-src 'self'; font-src 'self' https://fonts.gstatic.com; ..."
```

---

## Monitoring & Health Checks

### Health Check Endpoints

1. **Basic Health** (`/health`, `/healthz`):
   ```bash
   curl http://localhost:8080/health
   ```
   
   Response:
   ```json
   {
     "status": "healthy",
     "time": "2025-10-31T03:59:39Z"
   }
   ```

2. **Readiness Check** (`/ready`):
   ```bash
   curl http://localhost:8080/ready
   ```
   
   Response (healthy):
   ```json
   {
     "status": "ready",
     "time": "2025-10-31T03:59:39Z",
     "database": "connected"
   }
   ```
   
   Response (unhealthy):
   ```json
   {
     "status": "not ready",
     "error": "database ping failed"
   }
   ```

### Container Health Checks

Docker compose includes health checks:

```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### Kubernetes Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Testing Security

### Manual Security Testing

#### 1. Authentication Testing

```bash
# Test account lockout
for i in {1..6}; do 
  curl -X POST http://localhost:8080/api/login \
    -H "Content-Type: application/json" \
    -d '{"username":"test","password":"wrong"}'
done

# Expected: Account locked after 5 attempts
```

#### 2. File Upload Testing

```bash
# Test file size limit
dd if=/dev/zero of=large.jpg bs=1M count=15
curl -X POST http://localhost:8080/api/animals/upload-image \
  -H "Authorization: Bearer TOKEN" \
  -F "image=@large.jpg"

# Expected: File size exceeds limit error
```

#### 3. SQL Injection Testing

```bash
# All queries use parameterization, should not be vulnerable
curl "http://localhost:8080/api/groups/1/animals?name=' OR '1'='1"

# Expected: Empty results or proper error (not SQL error)
```

#### 4. XSS Testing

```bash
# Test XSS in registration
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "<script>alert(1)</script>",
    "email": "test@example.com",
    "password": "password123"
  }'

# Expected: Validation error (invalid characters)
```

### Automated Security Testing

#### Go Security Scanner

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...
```

#### Vulnerability Check

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan for vulnerabilities
govulncheck ./...
```

#### Container Scanning

```bash
# Build image
docker build -t go-volunteer-media:latest .

# Scan with Trivy
trivy image go-volunteer-media:latest
```

---

## Production Deployment

### Pre-Deployment Checklist

Use the comprehensive checklist: `PRODUCTION_SECURITY_CHECKLIST.md`

#### Critical Steps

1. **Generate Strong JWT Secret**:
   ```bash
   openssl rand -base64 32
   ```

2. **Enable TLS for Database**:
   ```env
   DB_SSLMODE=require
   ```

3. **Configure CORS**:
   ```env
   ALLOWED_ORIGINS=https://yourdomain.com
   ```

4. **Set Production Environment**:
   ```env
   ENV=production
   LOG_LEVEL=INFO
   LOG_FORMAT=json
   ```

5. **Enable HSTS** (after HTTPS is configured):
   Uncomment in `internal/middleware/security.go`

### Monitoring Setup

#### Log Aggregation

**ELK Stack Example**:
```yaml
# filebeat.yml
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/app/*.log
    json.keys_under_root: true
    json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

#### Alerting Rules

```yaml
# Alert on failed logins
- alert: HighFailedLoginRate
  expr: rate(login_failure_total[5m]) > 10
  annotations:
    summary: "High failed login rate detected"
```

### Backup & Recovery

1. **Database Backups**:
   ```bash
   # Daily backup
   pg_dump -h localhost -U postgres volunteer_media_prod > backup_$(date +%Y%m%d).sql
   ```

2. **Uploaded Files**:
   ```bash
   # Backup uploads directory
   tar -czf uploads_$(date +%Y%m%d).tar.gz public/uploads/
   ```

---

## Support & Contact

- **Security Issues**: See SECURITY.md for vulnerability reporting
- **Documentation**: See README.md for general documentation
- **Deployment**: See PRODUCTION_SECURITY_CHECKLIST.md

---

**Last Updated**: 2025-10-31  
**Version**: 1.0.0
