# Security Checklist - Quick Reference

Use this checklist before deploying to production or when reviewing code changes.

## Pre-Deployment Security Checklist

### 🔐 Authentication & Authorization
- [ ] JWT_SECRET is at least 32 characters (verified at startup)
- [ ] JWT_SECRET is cryptographically random (not a default/test value)
- [ ] Password requirements enforced (min 8, max 72 chars)
- [ ] Account lockout configured (5 attempts, 30 min lock)
- [ ] Token expiration set appropriately (24 hours)
- [ ] All protected routes use `AuthRequired()` middleware
- [ ] Admin routes use `AdminRequired()` middleware

### 🛡️ Input Validation
- [ ] All API inputs validated with struct tags
- [ ] File uploads validated (size, type, content)
- [ ] No raw SQL queries (GORM parameterized only)
- [ ] User-generated content sanitized
- [ ] Path traversal protection in place

### 🌐 Network Security
- [ ] CORS origins explicitly whitelisted (no "*" in production)
- [ ] Rate limiting enabled on auth endpoints
- [ ] Security headers configured (CSP, X-Frame-Options, etc.)
- [ ] TLS/SSL enabled for database (DB_SSLMODE=require or verify-full)
- [ ] HTTPS enabled for application
- [ ] HTTP timeouts configured (Read/Write/Idle)

### 🔒 Data Protection
- [ ] No secrets in code or version control
- [ ] No passwords/tokens in logs
- [ ] Database connection pool limits set
- [ ] Sensitive data excluded from JSON responses
- [ ] Environment variables used for all secrets

### 🐳 Container Security
- [ ] Running as non-root user
- [ ] Multi-stage builds used
- [ ] Minimal base images (Alpine/scratch)
- [ ] No secrets in image layers
- [ ] Resource limits configured
- [ ] Image scanned for vulnerabilities

### 📊 Observability
- [ ] Structured JSON logging enabled
- [ ] Request ID tracing configured
- [ ] Audit logging for security events
- [ ] Health endpoints working (/health, /ready)
- [ ] Log aggregation configured
- [ ] Alerts set up for security events

### 🔍 Dependency Security
- [ ] Run `govulncheck ./...` (no vulnerabilities)
- [ ] Run `npm audit` in frontend (no high/critical)
- [ ] Dependencies up to date
- [ ] No known vulnerabilities in base images
- [ ] Automated dependency updates configured

### 📝 Documentation
- [ ] SECURITY.md reviewed and accurate
- [ ] .env.example has no real secrets
- [ ] Deployment documentation current
- [ ] Security incident response plan in place
- [ ] Team knows how to respond to incidents

## Code Review Security Checklist

### 🔍 Review These Patterns

#### Authentication Code
```go
// ✅ GOOD: Proper token validation
claims, err := auth.ValidateToken(token)
if err != nil {
    return unauthorized
}

// ❌ BAD: Skipping validation
// Don't trust tokens without validation
```

#### Password Handling
```go
// ✅ GOOD: Using bcrypt
hashedPassword, err := auth.HashPassword(password)

// ❌ BAD: Plain text or weak hashing
// Never store passwords in plain text
```

#### Database Queries
```go
// ✅ GOOD: Parameterized query
db.Where("username = ?", username).First(&user)

// ❌ BAD: String concatenation
// db.Raw("SELECT * FROM users WHERE username = '" + username + "'")
```

#### File Uploads
```go
// ✅ GOOD: Validation before processing
if err := upload.ValidateImageUpload(file, MaxImageSize); err != nil {
    return err
}

// ❌ BAD: Accepting files without validation
```

#### Secrets
```go
// ✅ GOOD: Environment variables
secret := os.Getenv("JWT_SECRET")

// ❌ BAD: Hardcoded
// const secret = "my-secret-key"
```

#### Error Messages
```go
// ✅ GOOD: Generic error
return gin.H{"error": "Invalid credentials"}

// ❌ BAD: Revealing information
// return gin.H{"error": "User not found"}  // Email enumeration
```

## Security Testing Checklist

### Manual Testing
- [ ] Test account lockout (6 failed login attempts)
- [ ] Test rate limiting (rapid requests to /api/login)
- [ ] Test file upload validation (wrong type, too large)
- [ ] Test JWT expiration (wait 24+ hours)
- [ ] Test authorization (access admin routes as regular user)
- [ ] Test CORS (requests from unauthorized origins)
- [ ] Test password reset flow
- [ ] Test SQL injection attempts (should fail)

### Automated Testing
```bash
# Vulnerability scanning
govulncheck ./...
npm audit

# Container scanning (if applicable)
trivy image volunteer-media-api:latest

# Dockerfile best practices
hadolint Dockerfile

# Static analysis
gosec ./...
golangci-lint run
```

### Load Testing
```bash
# Test rate limiting
for i in {1..10}; do 
    curl -X POST http://localhost:8080/api/login \
    -H "Content-Type: application/json" \
    -d '{"username":"test","password":"test"}'
done
# Should see 429 Too Many Requests after 5 attempts
```

## Incident Response Quick Reference

### Detection
```bash
# Check for suspicious activity
grep "login_failure" logs/*.json | wc -l
grep "account_locked" logs/*.json
grep "rate_limit_exceeded" logs/*.json
```

### Immediate Response
```bash
# Block an IP
iptables -A INPUT -s <malicious-ip> -j DROP

# Lock a user account
# Via database:
# UPDATE users SET locked_until = NOW() + INTERVAL '24 hours' WHERE id = <user_id>;

# View recent logs
tail -f logs/app.log | jq 'select(.level == "ERROR" or .level == "WARN")'
```

### Escalation
1. If critical: Notify security team immediately
2. If high: Notify within 1 hour
3. Document everything
4. Preserve logs
5. Follow incident response plan

## Security Monitoring Queries

### Authentication Events
```bash
# Failed logins by IP
grep "login_failure" logs/*.json | jq -r '.fields.ip' | sort | uniq -c | sort -rn

# Recent account lockouts
grep "account_locked" logs/*.json | tail -5

# Successful logins today
grep "login_success" logs/*.json | grep "$(date +%Y-%m-%d)" | wc -l
```

### Admin Actions
```bash
# All admin actions today
grep "action_type.*admin" logs/*.json | grep "$(date +%Y-%m-%d)"

# Recent user deletions
grep "user_deleted" logs/*.json | tail -5
```

### Security Events
```bash
# Unauthorized access attempts
grep "unauthorized_access" logs/*.json

# File upload rejections
grep "file_upload_rejected" logs/*.json

# Rate limit violations
grep "rate_limit_exceeded" logs/*.json
```

## Environment Variable Validation

### Required Variables
```bash
# Check all required variables are set
[ -z "$JWT_SECRET" ] && echo "ERROR: JWT_SECRET not set"
[ ${#JWT_SECRET} -lt 32 ] && echo "ERROR: JWT_SECRET too short"
[ -z "$DB_HOST" ] && echo "ERROR: DB_HOST not set"
[ -z "$DB_PASSWORD" ] && echo "ERROR: DB_PASSWORD not set"
[ "$ENV" = "production" ] && [ "$DB_SSLMODE" = "disable" ] && echo "WARNING: SSL disabled in production"
[ "$ENV" = "production" ] && [ "$ALLOWED_ORIGINS" = "*" ] && echo "ERROR: CORS wildcard in production"
```

### Generate Secure Secrets
```bash
# Generate JWT_SECRET
openssl rand -base64 32

# Generate database password
openssl rand -base64 24

# Generate random string
cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
```

## Quick Security Fixes

### Fix Common Issues

#### Issue: Weak JWT Secret
```bash
# Generate new secret
export JWT_SECRET=$(openssl rand -base64 32)
# Update .env file
echo "JWT_SECRET=$JWT_SECRET" >> .env
```

#### Issue: CORS Wildcard
```env
# ❌ BAD
ALLOWED_ORIGINS=*

# ✅ GOOD
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

#### Issue: DB SSL Disabled in Production
```env
# ❌ BAD
DB_SSLMODE=disable

# ✅ GOOD
DB_SSLMODE=require
# or
DB_SSLMODE=verify-full
```

#### Issue: Debug Mode in Production
```env
# ❌ BAD
ENV=development

# ✅ GOOD
ENV=production
```

## Security Tools Quick Start

### Install Tools
```bash
# Go vulnerability scanner
go install golang.org/x/vuln/cmd/govulncheck@latest

# Container scanner
brew install trivy  # macOS
# or
apt-get install trivy  # Linux

# Static analysis
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### Run Scans
```bash
# Full security scan
govulncheck ./...                    # Go dependencies
npm audit                           # Frontend dependencies
trivy image volunteer-media-api     # Container image
gosec ./...                         # Static analysis
hadolint Dockerfile                 # Dockerfile best practices
```

## Compliance Quick Check

### OWASP Top 10 Coverage
- [x] A01: Broken Access Control → RBAC implemented
- [x] A02: Cryptographic Failures → bcrypt, crypto/rand
- [x] A03: Injection → Parameterized queries
- [x] A04: Insecure Design → Security-first architecture
- [x] A05: Security Misconfiguration → Secure defaults
- [x] A06: Vulnerable Components → Scanning in place
- [x] A07: Authentication Failures → Strong auth, lockout
- [x] A08: Software Integrity → Signed images recommended
- [x] A09: Logging Failures → Comprehensive audit log
- [x] A10: SSRF → No user-controlled URLs

## Contact Information

### Security Team
- **Email**: security@yourdomain.com
- **Slack**: #security-alerts
- **On-Call**: +1-555-0100

### Escalation
- **Critical (P0)**: Immediate - Call on-call
- **High (P1)**: Within 1 hour - Slack + email
- **Medium (P2)**: Within 4 hours - Email
- **Low (P3)**: Within 24 hours - Ticket

## Resources

- [SECURITY.md](./SECURITY.md) - Full security documentation
- [SECURITY_INCIDENT_RESPONSE.md](./SECURITY_INCIDENT_RESPONSE.md) - Incident response plan
- [CONTAINER_SECURITY_SCANNING.md](./CONTAINER_SECURITY_SCANNING.md) - Container scanning guide
- [SECURITY_ASSESSMENT_REPORT.md](./SECURITY_ASSESSMENT_REPORT.md) - Latest assessment

---

**Last Updated**: October 31, 2025  
**Version**: 1.0  
**Review Frequency**: Monthly
