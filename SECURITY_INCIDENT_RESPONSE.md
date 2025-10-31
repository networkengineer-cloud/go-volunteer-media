# Security Incident Response Plan

## Overview

This document outlines the security incident response procedures for the Haws Volunteer Media application. It provides guidance on how to identify, respond to, and recover from security incidents.

## Incident Response Team

- **Incident Commander**: System Administrator or Security Lead
- **Technical Lead**: Senior Developer
- **Communications Lead**: Project Manager
- **Legal Advisor**: (if applicable)

## Incident Severity Levels

### Critical (P0)
- Active data breach or unauthorized access to production systems
- Complete service outage affecting all users
- Discovery of actively exploited vulnerability
- Compromise of authentication systems or credentials

**Response Time**: Immediate (within 15 minutes)

### High (P1)
- Suspected unauthorized access attempts
- Discovery of high-severity vulnerability
- Partial service degradation
- Abnormal authentication patterns

**Response Time**: Within 1 hour

### Medium (P2)
- Discovery of medium-severity vulnerability
- Policy violations
- Suspicious but not confirmed security events

**Response Time**: Within 4 hours

### Low (P3)
- Minor security concerns
- Configuration issues
- Informational security findings

**Response Time**: Within 24 hours

## Incident Response Process

### 1. Detection and Identification

**Monitoring Sources:**
- Application logs (structured JSON logs)
- Audit logs (authentication events)
- Database query logs
- Rate limiting alerts
- Failed authentication attempts
- Health check failures
- External security scanning tools

**Key Security Events to Monitor:**
```bash
# Failed authentication attempts
grep "audit_event.*login_failure" logs/*.json

# Account lockouts
grep "audit_event.*account_locked" logs/*.json

# Rate limit violations
grep "audit_event.*rate_limit_exceeded" logs/*.json

# Unauthorized access attempts
grep "audit_event.*unauthorized_access" logs/*.json

# File upload rejections (potential malware)
grep "audit_event.*file_upload_rejected" logs/*.json
```

### 2. Initial Response (First 30 Minutes)

**Immediate Actions:**
1. **Confirm the incident** - Verify it's a real security event
2. **Isolate affected systems** - If compromise is suspected:
   ```bash
   # Scale down compromised services
   docker-compose down
   
   # Or in Kubernetes
   kubectl scale deployment volunteer-media-api --replicas=0
   ```
3. **Preserve evidence** - Copy logs before they rotate:
   ```bash
   # Copy logs to secure location
   mkdir -p /secure/incident-$(date +%Y%m%d-%H%M%S)
   cp -r logs/ /secure/incident-$(date +%Y%m%d-%H%M%S)/
   
   # Export database audit logs
   docker exec volunteer_media_db_prod pg_dump -U postgres -t users > /secure/users-backup-$(date +%Y%m%d-%H%M%S).sql
   ```
4. **Notify the incident response team**
5. **Create incident ticket** with all known information

### 3. Containment

**Short-term Containment:**
- Block malicious IP addresses at firewall level
- Disable compromised user accounts
- Rotate exposed credentials
- Enable additional logging and monitoring

**IP Blocking Example:**
```bash
# Add to firewall rules
iptables -A INPUT -s <malicious-ip> -j DROP

# Or in application config
BLOCKED_IPS="192.0.2.100,192.0.2.101"
```

**Account Lockout:**
```sql
-- Lock user account
UPDATE users SET locked_until = NOW() + INTERVAL '24 hours' WHERE id = <user_id>;
```

**Long-term Containment:**
- Deploy patches for exploited vulnerabilities
- Update security configurations
- Implement additional security controls

### 4. Eradication

**Remove the Threat:**
1. **Identify root cause** - Use logs and forensics
2. **Remove malicious code or backdoors**
3. **Close security gaps**
4. **Update dependencies** if vulnerability was in third-party code:
   ```bash
   # Update Go dependencies
   go get -u ./...
   go mod tidy
   
   # Update frontend dependencies
   cd frontend && npm audit fix
   ```

### 5. Recovery

**Restore Normal Operations:**
1. **Verify systems are clean** - Run security scans
2. **Restore from known-good backups** if necessary
3. **Gradually restore service**:
   ```bash
   # Start with limited capacity
   docker-compose up -d --scale api=1
   
   # Monitor for issues
   docker-compose logs -f api
   
   # Scale up if stable
   docker-compose up -d --scale api=3
   ```
4. **Monitor closely** for 24-48 hours

### 6. Post-Incident Review

**Within 7 days of resolution:**
1. **Document timeline** of events
2. **Analyze root cause**
3. **Identify what worked well and what didn't**
4. **Create action items** to prevent recurrence
5. **Update security policies** if needed
6. **Share lessons learned** with the team

## Common Security Incidents

### Brute Force Attack

**Detection:**
```bash
# Check for multiple failed login attempts
grep "login_failure" logs/*.json | grep "ip.*192.0.2.100" | wc -l
```

**Response:**
1. Verify rate limiting is working
2. Check if any accounts were compromised
3. Block attacking IP addresses
4. Review account lockout thresholds
5. Consider implementing CAPTCHA

### SQL Injection Attempt

**Detection:**
```bash
# Look for SQL keywords in request parameters
grep -i "select\|union\|drop\|insert" logs/*.json
```

**Response:**
1. **Note**: Application uses parameterized queries via GORM - SQL injection should be prevented
2. Verify no raw SQL queries exist
3. Check if any data was accessed or modified
4. Review input validation
5. Update WAF rules if applicable

### Unauthorized Data Access

**Detection:**
```bash
# Check for unauthorized access attempts
grep "unauthorized_access" logs/*.json
```

**Response:**
1. Identify what data was accessed
2. Determine how authorization was bypassed
3. Lock affected accounts
4. Review access control logic
5. Audit all recent access patterns
6. Notify affected users if data was compromised

### Malicious File Upload

**Detection:**
```bash
# Check rejected file uploads
grep "file_upload_rejected" logs/*.json
```

**Response:**
1. Verify file was properly rejected
2. Check if any malicious files made it through
3. Scan upload directory:
   ```bash
   # Use ClamAV or similar
   clamscan -r /app/public/uploads
   ```
4. Review file validation logic
5. Consider adding additional MIME type checks

### Credential Compromise

**Detection:**
- User reports unauthorized access
- Login from unusual location
- Multiple simultaneous sessions

**Response:**
1. **Immediately lock the account**:
   ```sql
   UPDATE users SET locked_until = NOW() + INTERVAL '24 hours' WHERE id = <user_id>;
   ```
2. **Invalidate all sessions** - Users must re-authenticate
3. **Force password reset**
4. **Review account activity** for unauthorized actions
5. **Check for data exfiltration**
6. **Notify the user** via verified contact method
7. **Report to legal/compliance** if required

### Denial of Service (DoS)

**Detection:**
- High rate of requests from single IP
- Service degradation or unavailability
- Abnormal resource consumption

**Response:**
1. **Identify attack pattern**
2. **Implement rate limiting** (already in place)
3. **Block attacking IPs**:
   ```bash
   iptables -A INPUT -s <attacking-ip> -j DROP
   ```
4. **Scale up resources** if legitimate traffic spike
5. **Contact hosting provider** for DDoS mitigation
6. **Enable CloudFlare or similar CDN/WAF** for protection

## Security Event Log Queries

### Authentication Events
```bash
# Successful logins
grep "audit_event.*login_success" logs/*.json | jq -r '.timestamp + " " + .fields.username + " " + .fields.ip'

# Failed logins by IP
grep "audit_event.*login_failure" logs/*.json | jq -r '.fields.ip' | sort | uniq -c | sort -rn

# Account lockouts
grep "audit_event.*account_locked" logs/*.json | jq -r '.timestamp + " " + .fields.username + " attempts:" + (.fields.failed_attempts | tostring)'
```

### Admin Actions
```bash
# All admin actions
grep "action_type.*admin" logs/*.json | jq -r '.timestamp + " " + .audit_event + " by admin_id:" + (.fields.admin_id | tostring)'

# User promotions/demotions
grep "audit_event.*(user_promoted|user_demoted)" logs/*.json
```

### File Operations
```bash
# Rejected uploads
grep "file_upload_rejected" logs/*.json | jq -r '.timestamp + " user:" + (.fields.user_id | tostring) + " file:" + .fields.filename + " reason:" + .fields.reason'

# Successful uploads
grep "image_uploaded" logs/*.json
```

## Communication Templates

### Internal Notification

**Subject**: SECURITY INCIDENT - [Severity] - [Brief Description]

**Body**:
```
Security Incident Notification

Severity: [P0/P1/P2/P3]
Status: [Detected/Contained/Resolved]
Incident ID: INC-YYYYMMDD-NNN

Summary:
[Brief description of the incident]

Impact:
[Affected systems/users/data]

Current Actions:
[What is being done]

Incident Commander: [Name]
Next Update: [Time]

Do not discuss this incident outside the response team until cleared by Incident Commander.
```

### User Notification (if required)

**Subject**: Security Notice - Account Security Update

**Body**:
```
Dear [User],

We are writing to inform you of a security incident that may have affected your account.

What happened:
[Brief, non-technical description]

What information was involved:
[Specific data types, no personal details in mass email]

What we're doing:
[Steps taken to address the issue]

What you should do:
1. Change your password immediately
2. Review your recent account activity
3. Enable email notifications for security events
4. Contact us if you notice any suspicious activity

We take the security of your information seriously and apologize for any concern this may cause.

If you have questions, please contact: [support email]

Sincerely,
The Haws Volunteers Security Team
```

## Tools and Resources

### Security Scanning Tools
```bash
# Vulnerability scanning
govulncheck ./...

# Docker image scanning
trivy image volunteer-media-api:latest

# Dependency audit
npm audit  # Frontend
go list -json -m all | nancy sleuth  # Backend

# Static analysis
gosec ./...
golangci-lint run
```

### Log Analysis Tools
```bash
# Install jq for JSON parsing
sudo apt-get install jq

# Real-time log monitoring
tail -f logs/app.log | jq 'select(.level == "ERROR" or .level == "WARN")'

# Count events by type
cat logs/*.json | jq -r '.audit_event' | sort | uniq -c | sort -rn
```

## Contact Information

### Internal Contacts
- **Security Lead**: security@yourdomain.com
- **On-Call Engineer**: +1-555-0100
- **System Administrator**: sysadmin@yourdomain.com

### External Contacts
- **Hosting Provider Support**: [Provider contact]
- **Database Provider Support**: [Provider contact]
- **Legal Counsel**: [Legal contact]
- **Law Enforcement** (if needed): Local cybercrime unit

## Regulatory Compliance

### Data Breach Notification Requirements

**GDPR** (if applicable):
- Notify supervisory authority within 72 hours
- Notify affected individuals if high risk to rights and freedoms

**State Data Breach Laws** (US):
- Requirements vary by state
- Consult legal counsel for specific requirements
- Most require notification without unreasonable delay

### Documentation Requirements
- Timeline of discovery and response
- Nature and scope of breach
- Data elements involved
- Number of affected individuals
- Remediation steps taken
- Contact information for questions

## Lessons Learned Template

**Incident ID**: INC-YYYYMMDD-NNN  
**Date**: YYYY-MM-DD  
**Incident Commander**: [Name]

### Executive Summary
[Brief overview of incident]

### Timeline
- **[Time]**: Detection
- **[Time]**: Initial response
- **[Time]**: Containment
- **[Time]**: Resolution

### Root Cause
[Technical root cause]

### Impact Assessment
- **Users affected**: [Number]
- **Data accessed**: [Types]
- **Downtime**: [Duration]
- **Financial impact**: [Estimate]

### What Went Well
- [Point 1]
- [Point 2]

### What Could Be Improved
- [Point 1]
- [Point 2]

### Action Items
| Action | Owner | Due Date | Status |
|--------|-------|----------|--------|
| [Action 1] | [Name] | [Date] | Open |
| [Action 2] | [Name] | [Date] | Open |

### Recommendations
- [Recommendation 1]
- [Recommendation 2]

---

**Document Version**: 1.0  
**Last Updated**: 2025-10-31  
**Next Review Date**: 2026-04-30  
**Owner**: Security Team
