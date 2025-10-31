# Production Deployment Security Checklist

Use this checklist before deploying the go-volunteer-media application to production.

## Pre-Deployment Security Configuration

### 1. Environment Variables ✅

- [ ] **JWT_SECRET** is set to a strong, randomly generated value (minimum 32 characters)
  ```bash
  # Generate with:
  openssl rand -base64 32
  ```
- [ ] **DB_PASSWORD** is set to a strong password (not default)
- [ ] **DB_SSLMODE** is set to `require` or `verify-full` (not `disable`)
- [ ] **ALLOWED_ORIGINS** is set to specific domain(s) (never `*`)
- [ ] **SMTP_PASSWORD** is set to app-specific password (for Gmail) or secure password
- [ ] **ENV** is set to `production`
- [ ] **FRONTEND_URL** is set to production URL with HTTPS
- [ ] All environment variables are stored in secure secret management (not in code)

### 2. Database Security ✅

- [ ] Database password changed from default
- [ ] SSL/TLS enabled for database connections
- [ ] Database accessible only from application servers (firewall rules)
- [ ] Database user has minimum required privileges
- [ ] Database backups configured and tested
- [ ] Database audit logging enabled
- [ ] Connection pooling configured appropriately
- [ ] Database access logged and monitored

### 3. Application Security ✅

- [ ] Application runs behind HTTPS (TLS 1.2+ only)
- [ ] HSTS header enabled (uncomment in security.go)
- [ ] Rate limiting thresholds reviewed and adjusted for production load
- [ ] File upload directory has appropriate permissions (no execute)
- [ ] Uploaded files are stored outside webroot or served through application
- [ ] Content Security Policy reviewed and adjusted for frontend requirements
- [ ] CORS origins explicitly whitelisted (no wildcards)
- [ ] Error messages don't leak sensitive information
- [ ] Stack traces disabled in production

### 4. Container Security (if using Docker) ✅

- [ ] Base images updated to latest stable versions
- [ ] Docker images scanned for vulnerabilities (Trivy, Snyk)
- [ ] Container runs as non-root user (already configured)
- [ ] Resource limits configured (CPU, memory)
- [ ] Health checks configured
- [ ] Container registry uses authentication
- [ ] Images signed and verified (Docker Content Trust)
- [ ] No secrets in Docker image layers
- [ ] Multi-stage builds minimize attack surface (already configured)

### 5. Network Security ✅

- [ ] Application only accessible through reverse proxy (Nginx, HAProxy)
- [ ] Reverse proxy configured with security headers
- [ ] DDoS protection enabled (Cloudflare, AWS Shield)
- [ ] Firewall rules restrict access to necessary ports only
- [ ] Internal services not exposed to public internet
- [ ] VPN or bastion host for administrative access
- [ ] Network segmentation in place

### 6. Monitoring & Logging ✅

- [ ] Log aggregation configured (ELK, Splunk, CloudWatch)
- [ ] Monitoring alerts set up for:
  - [ ] Failed authentication attempts (>10 per minute)
  - [ ] Account lockouts
  - [ ] Unusual API usage patterns
  - [ ] Error rate spikes
  - [ ] Resource exhaustion (CPU, memory, disk)
  - [ ] Health check failures
- [ ] Request tracing with correlation IDs
- [ ] Audit logs for admin actions enabled
- [ ] Log retention policy defined
- [ ] Sensitive data never logged (passwords, tokens)

### 7. Dependency Management ✅

- [ ] All dependencies updated to latest stable versions
- [ ] Vulnerability scan run (`govulncheck ./...`)
- [ ] No high/critical vulnerabilities in dependencies
- [ ] Dependency update process documented
- [ ] Regular dependency audits scheduled

### 8. Backup & Recovery ✅

- [ ] Database backup automated (daily minimum)
- [ ] Backup restoration tested
- [ ] Uploaded files backed up
- [ ] Disaster recovery plan documented
- [ ] RTO (Recovery Time Objective) defined
- [ ] RPO (Recovery Point Objective) defined
- [ ] Backup encryption enabled

### 9. Access Control ✅

- [ ] Admin users created with strong passwords
- [ ] Principle of least privilege enforced
- [ ] Service accounts use separate credentials
- [ ] SSH keys used instead of passwords
- [ ] Multi-factor authentication enabled for admin access
- [ ] Regular access review scheduled
- [ ] Offboarding process includes access revocation

### 10. Testing ✅

- [ ] Security testing performed:
  - [ ] Authentication bypass attempts
  - [ ] SQL injection testing
  - [ ] XSS testing
  - [ ] CSRF testing
  - [ ] File upload malicious file testing
  - [ ] Rate limiting verification
  - [ ] Account lockout testing
- [ ] Load testing performed
- [ ] Failover testing performed
- [ ] Backup restoration tested

## Post-Deployment Security

### Immediate (Within 24 Hours)

- [ ] Verify HTTPS is working correctly
- [ ] Verify health check endpoints responding
- [ ] Check logs for errors or security warnings
- [ ] Verify monitoring alerts are working
- [ ] Test authentication flows
- [ ] Verify file uploads working correctly
- [ ] Check resource usage (CPU, memory, disk)

### First Week

- [ ] Review security logs for anomalies
- [ ] Monitor authentication failure rates
- [ ] Review API usage patterns
- [ ] Check for any performance issues
- [ ] Verify backup jobs running successfully
- [ ] Test alert escalation procedures

### Ongoing Maintenance

- [ ] Weekly vulnerability scans
- [ ] Monthly dependency updates
- [ ] Quarterly access reviews
- [ ] Quarterly security testing
- [ ] Quarterly disaster recovery drills
- [ ] Quarterly secret rotation (JWT_SECRET, DB_PASSWORD)
- [ ] Annual penetration testing
- [ ] Annual security audit

## Security Incident Response

### Preparation

- [ ] Incident response plan documented
- [ ] Incident response team identified
- [ ] Communication plan established
- [ ] Backup contact information available

### Detection & Analysis

- [ ] Security monitoring in place
- [ ] Anomaly detection configured
- [ ] Log analysis tools available
- [ ] Incident classification criteria defined

### Containment, Eradication & Recovery

- [ ] Isolation procedures documented
- [ ] Rollback procedures tested
- [ ] Recovery time objectives defined
- [ ] Communication templates prepared

### Post-Incident

- [ ] Post-mortem process defined
- [ ] Lessons learned documentation
- [ ] Remediation tracking
- [ ] Continuous improvement process

## Compliance Considerations

If your deployment must comply with specific regulations, ensure:

### GDPR (if applicable)

- [ ] Data protection impact assessment completed
- [ ] User consent mechanisms implemented
- [ ] Data deletion capability implemented
- [ ] Data export capability implemented
- [ ] Privacy policy updated
- [ ] Data processing agreements in place

### HIPAA (if applicable)

- [ ] Business associate agreements signed
- [ ] PHI encryption at rest and in transit
- [ ] Audit logging comprehensive
- [ ] Access controls strict
- [ ] Disaster recovery tested

### SOC 2 (if applicable)

- [ ] Security policies documented
- [ ] Change management process defined
- [ ] Vendor risk assessment completed
- [ ] Control testing performed

## Production Environment Verification

Run these commands to verify your production configuration:

```bash
# 1. Verify TLS/HTTPS
curl -I https://yourdomain.com/health

# 2. Verify security headers
curl -I https://yourdomain.com/health | grep -E "X-Content-Type-Options|X-Frame-Options|Content-Security-Policy"

# 3. Test rate limiting
for i in {1..10}; do curl -X POST https://yourdomain.com/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"wrong"}'; done

# 4. Verify health endpoints
curl https://yourdomain.com/health
curl https://yourdomain.com/ready

# 5. Test authentication
curl -X POST https://yourdomain.com/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass"}'

# 6. Verify CORS (replace with your frontend domain)
curl -H "Origin: https://yourfrontend.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: X-Requested-With" \
  --head https://yourdomain.com/api/groups
```

## Security Tools & Resources

### Scanning Tools

- **govulncheck**: Go vulnerability scanner
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

- **Trivy**: Container vulnerability scanner
  ```bash
  trivy image your-app:latest
  ```

- **gosec**: Go security checker
  ```bash
  go install github.com/securego/gosec/v2/cmd/gosec@latest
  gosec ./...
  ```

### Security Testing Resources

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- OWASP API Security Top 10: https://owasp.org/www-project-api-security/
- CWE (Common Weakness Enumeration): https://cwe.mitre.org/
- CVE Database: https://cve.mitre.org/

## Sign-Off

This checklist should be completed and signed off by:

- [ ] Development Lead: _______________ Date: ___________
- [ ] Security Team: _______________ Date: ___________
- [ ] Operations Team: _______________ Date: ___________
- [ ] Product Owner: _______________ Date: ___________

---

**Document Version**: 1.0.0  
**Last Updated**: 2025-10-31  
**Next Review Date**: ___________
