# Email Configuration Guide

This guide explains how to configure email functionality in the Haws Volunteers application.

## Overview

The application uses a provider-based email system that supports multiple email backends:

- **Resend** (Recommended) - Modern email API with excellent developer experience
- **SMTP** (Legacy) - Traditional SMTP for services like Gmail, SendGrid, Mailgun, etc.

**⚠️ Domain Flexibility:** The email system is fully flexible and works with **any domain you control**. All domains are configured via environment variables, so you can use your organization's domain (e.g., `notifications.myhaws.org`, `noreply@yourdomain.com`, etc.) without any code changes.

## Quick Start

### Using Resend (Recommended)

1. **Sign up for Resend**
   - Visit [resend.com](https://resend.com) and create an account
   - Free tier includes: 100 emails/day, 3,000 emails/month

2. **Get your API key**
   - Go to your Resend dashboard
   - Navigate to API Keys
   - Create a new API key

3. **Configure environment variables**
   ```env
   EMAIL_PROVIDER=resend
   RESEND_API_KEY=re_your_api_key_here
   RESEND_FROM_EMAIL=noreply@notifications.myhaws.org  # Use YOUR verified domain
   RESEND_FROM_NAME=Haws Volunteers
   FRONTEND_URL=http://localhost:5173
   ```

4. **Domain verification** (for production)
   - Add your domain in Resend dashboard (e.g., `notifications.myhaws.org`)
   - Verify DNS records (SPF, DKIM, DMARC)
   - Use your verified domain in `RESEND_FROM_EMAIL`
   - **Note:** You can use any domain you control - the system is completely flexible

### Using SMTP (Gmail Example)

1. **Enable 2-Factor Authentication on Gmail**
   - Go to Google Account settings
   - Security → 2-Step Verification

2. **Generate App Password**
   - Google Account → Security → App passwords
   - Generate a new app password for "Mail"
   - Copy the 16-character password

3. **Configure environment variables**
   ```env
   EMAIL_PROVIDER=smtp
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=your-email@gmail.com
   SMTP_PASSWORD=your-16-char-app-password
   SMTP_FROM_EMAIL=your-email@gmail.com
   SMTP_FROM_NAME=Haws Volunteers
   FRONTEND_URL=http://localhost:5173
   ```

## Environment Variables

### Provider Selection
```env
EMAIL_PROVIDER=resend  # or "smtp"
```
Defaults to `smtp` if not set (for backwards compatibility).

### Resend Configuration
```env
RESEND_API_KEY=re_xxxxxxxxxxxx                      # Required: Your Resend API key
RESEND_FROM_EMAIL=noreply@notifications.myhaws.org  # Required: Verified sender email (use YOUR domain)
RESEND_FROM_NAME=Haws Volunteers                    # Optional: Sender display name
```

**Domain Flexibility:** You can use **any domain** you own and have verified in Resend. Examples:
- `noreply@notifications.myhaws.org`
- `alerts@yourdomain.com`
- `volunteers@example.org`

### SMTP Configuration
```env
SMTP_HOST=smtp.example.com                      # Required: SMTP server hostname
SMTP_PORT=587                                    # Required: Port (587 for TLS, 465 for SSL)
SMTP_USERNAME=user@example.com                  # Required: SMTP username
SMTP_PASSWORD=your-password                     # Required: SMTP password
SMTP_FROM_EMAIL=noreply@notifications.myhaws.org # Required: From email (use YOUR domain)
SMTP_FROM_NAME=Haws Volunteers                  # Optional: Sender display name
```

**Domain Flexibility:** SMTP works with any email address/domain you have access to. No hardcoded restrictions.

### Frontend URL
```env
FRONTEND_URL=http://localhost:5173      # Required: For password reset links
```
This URL is used in password reset emails to generate the reset link.

## Features Using Email

### 1. Password Reset
- Users can request password reset from login page
- System sends email with secure reset token
- Token expires after 1 hour
- Reset link format: `{FRONTEND_URL}/reset-password?token={token}`

### 2. Announcement Notifications
- Admins can send announcements to all users or specific groups
- Users opt-in/out via email preferences
- Emails include formatted HTML content
- Option to send via email, GroupMe, or both

### 3. Email Preferences
- Users can enable/disable email notifications
- Settings accessible via user profile
- API endpoints: `GET /api/email-preferences`, `PUT /api/email-preferences`

## Testing Email Configuration

### Development Testing

1. **Check service configuration**
   ```bash
   # Start the backend
   go run cmd/api/main.go
   
   # Look for log message:
   # "Email service configured and ready" or
   # "Email service not configured - password reset and email notifications will be disabled"
   ```

2. **Test password reset**
   - Go to login page
   - Click "Forgot Password?"
   - Enter a valid user email
   - Check email inbox (or logs if using test mode)

3. **Test announcement emails**
   - Login as admin
   - Create an announcement with "Send Email" checked
   - Verify users with email notifications enabled receive the email

### Production Testing

1. **Verify domain** (Resend only)
   - Add domain in Resend dashboard
   - Configure DNS records (SPF, DKIM, DMARC)
   - Test with `noreply@yourdomain.com`

2. **Test deliverability**
   - Send test emails to multiple providers (Gmail, Outlook, Yahoo)
   - Check spam folders
   - Monitor Resend dashboard for delivery status

3. **Monitor logs**
   - Check for email send errors
   - Verify successful deliveries
   - Monitor rate limits

## Architecture

### Provider Interface
```go
type Provider interface {
    SendEmail(to, subject, htmlBody string) error
    IsConfigured() bool
    GetProviderName() string
}
```

### Service Implementation
```go
type Service struct {
    provider Provider
}

// Email templates built-in:
- SendPasswordResetEmail(to, username, resetToken string) error
- SendAnnouncementEmail(to, title, content string) error
```

### Provider Selection
```go
// Determined by EMAIL_PROVIDER environment variable
// Default: smtp (backwards compatible)
provider, err := email.NewProvider()
```

## Troubleshooting

### Issue: "Email service is not configured"
**Solution:** 
- Check that environment variables are set correctly
- For Resend: Verify `RESEND_API_KEY` and `RESEND_FROM_EMAIL`
- For SMTP: Verify all SMTP_* variables are set

### Issue: Emails not being delivered
**Solutions:**
- **Resend:**
  - Check API key is valid
  - Verify domain in Resend dashboard (e.g., `notifications.myhaws.org`)
  - Check Resend logs for errors
  - Ensure sender email uses YOUR verified domain (the `RESEND_FROM_EMAIL` must match a domain you've verified)
  
- **SMTP:**
  - Verify SMTP credentials
  - Check firewall allows outbound connections on port 587/465
  - For Gmail: Ensure app password (not regular password)
  - Check SMTP server logs

### Issue: Password reset emails have wrong URL
**Solution:**
- Set `FRONTEND_URL` to your actual frontend URL
- Example production: `FRONTEND_URL=https://volunteers.example.com`
- Example development: `FRONTEND_URL=http://localhost:5173`

### Issue: Emails marked as spam
**Solutions:**
- Configure SPF, DKIM, and DMARC records (Resend helps with this)
- Use a verified domain (not Gmail for sending)
- Avoid spam trigger words in content
- Include unsubscribe link in emails
- Warm up new sending domain gradually

## Security Considerations

### API Keys and Passwords
- **Never commit secrets to git**
- Use environment variables only
- Rotate API keys regularly (especially after team member leaves)
- Use different API keys for dev/staging/production

### Email Content
- HTML is sanitized before sending
- User input is escaped to prevent XSS
- Newlines converted to `<br>` tags safely

### Rate Limiting
- Authentication endpoints rate-limited (5 requests/minute default)
- Prevents password reset email abuse
- Configurable via `AUTH_RATE_LIMIT_PER_MINUTE`

### Token Security
- Password reset tokens are cryptographically secure (32 bytes)
- Tokens are hashed before storing in database
- Tokens expire after 1 hour
- Used tokens are cleared after password reset

## Switching Providers

To switch from SMTP to Resend (or vice versa):

1. **Update environment variable**
   ```env
   EMAIL_PROVIDER=resend  # Change from smtp to resend
   ```

2. **Add new provider configuration**
   ```env
   # Add Resend config with YOUR domain
   RESEND_API_KEY=re_xxxxxxxxxxxx
   RESEND_FROM_EMAIL=noreply@notifications.myhaws.org  # Use your verified domain
   ```

3. **Restart application**
   ```bash
   # No code changes needed!
   go run cmd/api/main.go
   ```

4. **Verify new provider**
   - Check logs for "Email service configured and ready"
   - Test password reset flow
   - Send test announcement

## Cost Comparison

### Resend (Recommended)
- **Free Tier:** 100 emails/day, 3,000 emails/month
- **Pro:** $20/month for 50,000 emails
- **Pros:** Easy setup, great DX, built-in deliverability
- **Best for:** Most use cases, especially small-medium deployments

### SMTP via Gmail
- **Cost:** Free (with Gmail account)
- **Limits:** 500 emails/day
- **Pros:** No additional cost, easy to test
- **Cons:** Not recommended for production, delivery issues
- **Best for:** Development and testing only

### SMTP via SendGrid
- **Free Tier:** 100 emails/day
- **Essentials:** $15/month for 50,000 emails
- **Pros:** Reliable, established provider
- **Cons:** More complex setup than Resend
- **Best for:** Large deployments, complex requirements

## Additional Resources

- [Resend Documentation](https://resend.com/docs)
- [Gmail App Passwords Guide](https://support.google.com/accounts/answer/185833)
- [Email Deliverability Best Practices](https://www.validity.com/resource-center/email-deliverability-best-practices/)
- [SPF, DKIM, and DMARC Setup](https://www.cloudflare.com/learning/dns/dns-records/dns-spf-dkim-dmarc/)

## Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review application logs for error messages
3. Test with a simple email provider (like Resend) first
4. Create an issue on GitHub with detailed error logs
