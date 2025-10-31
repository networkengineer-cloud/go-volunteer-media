# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in this application, please report it by emailing the security team. Please do **NOT** create a public GitHub issue for security vulnerabilities.

When reporting a vulnerability, please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)

We will acknowledge receipt of your vulnerability report within 48 hours and send you regular updates about our progress.

## Security Features

This application implements multiple layers of security to protect user data and prevent unauthorized access:

### Authentication & Authorization

- **JWT-based Authentication**: Secure token-based authentication using HS256 signing
- **bcrypt Password Hashing**: Passwords are hashed using bcrypt with appropriate cost factor
- **Account Lockout**: Accounts are locked for 30 minutes after 5 failed login attempts
- **Password Requirements**: Minimum 8 characters, maximum 72 characters (bcrypt limit)
- **Password Reset**: Time-limited reset tokens (1 hour expiration) with secure random generation
- **Email Enumeration Protection**: Password reset endpoints don't reveal if email exists
- **Role-Based Access Control**: Admin and user roles with appropriate permissions

### Input Validation & Sanitization

- **Request Validation**: All API inputs validated using struct tags and validators
- **SQL Injection Prevention**: Parameterized queries via GORM (no raw SQL with user input)
- **File Upload Validation**:
  - File size limits (10MB for images, 5MB for hero images)
  - File type validation (extension and MIME type checking)
  - Content verification (magic number validation)
  - Malicious file prevention
- **XSS Prevention**: HTML escaping for user-generated content in emails

### Network Security

- **CORS Configuration**: Strict origin whitelisting (no wildcards in production)
- **Rate Limiting**: Protection against brute force attacks on authentication endpoints
- **Security Headers**:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Content-Security-Policy`: Strict CSP policy
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy`: Feature restriction
- **HTTP Timeouts**: Read (15s), Write (15s), and Idle (60s) timeouts configured
- **TLS for Database**: SSL/TLS validation enforced for database connections
- **TLS for Email**: Certificate verification for SMTP connections

### Data Protection

- **No Secrets in Logs**: Passwords, tokens, and credentials never logged
- **Secure Token Storage**: Reset tokens hashed before storage
- **Environment Variables**: All secrets stored in environment variables (never hardcoded)
- **Password Masking**: Passwords excluded from JSON responses
- **Context Propagation**: Request context properly propagated for cancellation

### Container Security

- **Non-Root User**: Docker containers run as non-root user
- **Multi-Stage Builds**: Minimal production images with reduced attack surface
- **Minimal Base Images**: Using Alpine and scratch images
- **No Secrets in Layers**: Build-time secrets never committed to image layers

### Observability & Monitoring

- **Request ID Tracing**: Unique request IDs for correlation across services
- **Health Checks**: `/health` and `/ready` endpoints for monitoring
- **Structured Logging**: Security events and errors logged for audit trail
- **Graceful Shutdown**: Proper cleanup of resources and in-flight requests

## Security Best Practices for Deployment

### Environment Variables

**Required Environment Variables:**
```bash
# JWT Secret (CRITICAL - must be at least 32 characters)
JWT_SECRET="<generate-with-openssl-rand-base64-32>"

# Database Configuration
DB_HOST="your-db-host"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="<strong-password>"
DB_NAME="volunteer_media_prod"
DB_SSLMODE="require"  # Use 'require' or 'verify-full' in production

# CORS Configuration
ALLOWED_ORIGINS="https://yourdomain.com"  # Never use "*" in production

# Email Configuration (SMTP)
SMTP_HOST="smtp.gmail.com"
SMTP_PORT="587"
SMTP_USERNAME="your-email@gmail.com"
SMTP_PASSWORD="<app-specific-password>"
SMTP_FROM_EMAIL="noreply@yourdomain.com"
SMTP_FROM_NAME="Your App Name"

# Application Configuration
ENV="production"
PORT="8080"
FRONTEND_URL="https://yourdomain.com"
```

### Production Deployment Checklist

- [ ] Generate a strong JWT_SECRET (at least 32 characters, use `openssl rand -base64 32`)
- [ ] Enable TLS/SSL for database connections (`DB_SSLMODE=require` or `verify-full`)
- [ ] Configure CORS with specific origins (never use `*`)
- [ ] Use strong database passwords
- [ ] Enable HTTPS for the application (add HSTS header when enabled)
- [ ] Set up log aggregation and monitoring
- [ ] Configure backup strategy for database
- [ ] Set up automated security scanning in CI/CD
- [ ] Review and adjust rate limiting thresholds
- [ ] Ensure all uploaded files are scanned for malware
- [ ] Configure Content Security Policy for your frontend
- [ ] Set up alerts for failed authentication attempts
- [ ] Enable database audit logging
- [ ] Configure firewall rules to restrict database access
- [ ] Use secrets management service (e.g., HashiCorp Vault, AWS Secrets Manager)
- [ ] Regularly update dependencies (`go get -u ./...`)
- [ ] Run vulnerability scans (`govulncheck ./...`)
- [ ] Review security headers with tools like securityheaders.com
- [ ] Perform penetration testing before production deployment

### Docker Security Configuration

When deploying with Docker:

1. **Never use default passwords** in docker-compose.yml
2. **Set resource limits** for containers (CPU, memory)
3. **Use Docker secrets** for sensitive configuration
4. **Scan images** with tools like Trivy or Snyk
5. **Keep base images updated** regularly
6. **Use specific image tags** (not `latest`)
7. **Enable Docker Content Trust** for image signing

Example docker-compose.yml excerpt:
```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 256M
    secrets:
      - jwt_secret
      - db_password
```

### Kubernetes Security (if applicable)

- Use **NetworkPolicies** to restrict pod-to-pod communication
- Implement **Pod Security Standards** (restricted profile)
- Use **Secrets** for sensitive data (never ConfigMaps)
- Enable **RBAC** with least privilege principle
- Use **admission controllers** for policy enforcement
- Configure **resource quotas** and **limit ranges**
- Enable **audit logging** for cluster events
- Use **service meshes** (Istio, Linkerd) for mTLS between services

## Security Testing

### Manual Security Testing

1. **Authentication Testing**:
   ```bash
   # Test account lockout
   for i in {1..6}; do curl -X POST http://localhost:8080/api/login \
     -H "Content-Type: application/json" \
     -d '{"username":"test","password":"wrong"}'; done
   
   # Test rate limiting
   for i in {1..10}; do curl -X POST http://localhost:8080/api/login \
     -H "Content-Type: application/json" \
     -d '{"username":"test","password":"test"}'; done
   ```

2. **File Upload Testing**:
   ```bash
   # Test file size limit
   dd if=/dev/zero of=large.jpg bs=1M count=15
   curl -X POST http://localhost:8080/api/animals/upload-image \
     -H "Authorization: Bearer $TOKEN" \
     -F "image=@large.jpg"
   
   # Test file type validation
   curl -X POST http://localhost:8080/api/animals/upload-image \
     -H "Authorization: Bearer $TOKEN" \
     -F "image=@malicious.php"
   ```

3. **SQL Injection Testing**:
   ```bash
   # All queries use parameterization, but test anyway
   curl -X GET "http://localhost:8080/api/groups/1/animals?name=' OR '1'='1"
   ```

### Automated Security Testing

1. **Vulnerability Scanning**:
   ```bash
   # Go vulnerability check
   govulncheck ./...
   
   # Docker image scanning
   trivy image your-app:latest
   
   # Dependency audit
   go list -json -m all | nancy sleuth
   ```

2. **Static Analysis**:
   ```bash
   # Security-focused linting
   gosec ./...
   
   # General linting
   golangci-lint run
   ```

3. **Dependency Updates**:
   ```bash
   # Update dependencies
   go get -u ./...
   go mod tidy
   
   # Check for outdated packages
   go list -u -m all
   ```

## Known Security Considerations

### Current Limitations

1. **JWT Refresh Mechanism**: The application currently doesn't implement JWT token refresh. Tokens expire after 24 hours and require re-authentication.

2. **Two-Factor Authentication**: 2FA is not currently implemented but the architecture supports adding it in the future.

3. **File Cleanup**: Orphaned uploaded files are not automatically cleaned up. Consider implementing a cleanup job.

4. **Session Revocation**: JWT tokens cannot be revoked before expiration without implementing a token blacklist.

5. **Distributed Rate Limiting**: Rate limiting is per-instance. For scaled deployments, implement distributed rate limiting with Redis.

### Future Enhancements

- Implement JWT refresh token mechanism
- Add two-factor authentication (TOTP)
- Implement audit logging for all security-relevant events
- Add anomaly detection for suspicious activities
- Implement automated orphaned file cleanup
- Add support for external identity providers (OAuth 2.0, SAML)
- Implement token revocation mechanism
- Add support for API key authentication
- Implement distributed rate limiting with Redis
- Add support for IP whitelisting/blacklisting

## Compliance

This application is designed with security best practices in mind, but compliance with specific regulations (GDPR, HIPAA, SOC 2, etc.) may require additional configuration and policies. Consult with your compliance team before deploying in regulated environments.

## Security Updates

We recommend:
- Monitoring GitHub Security Advisories for Go and dependencies
- Subscribing to security mailing lists for PostgreSQL, Gin, and other dependencies
- Running vulnerability scans regularly (at least weekly)
- Updating dependencies monthly or when critical vulnerabilities are announced
- Reviewing and rotating secrets quarterly

## Contact

For security concerns or questions, please contact the security team at security@yourdomain.com.

---

**Last Updated**: 2025-10-31
**Version**: 1.0.0
