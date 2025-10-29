# Login Security and Email Features - Testing Guide

## Overview
This implementation adds three major security and communication features:
1. **Login Attempt Restrictions** - Accounts lock after 5 failed login attempts
2. **Password Reset via Email** - Users can reset forgotten passwords
3. **Email Notification Preferences** - Users can opt in/out of announcement emails

## Features Implemented

### 1. Login Restrictions (5 Failed Attempts → Account Lock)

**How it works:**
- Each failed login attempt is recorded
- After 5 failed attempts, the account is locked for 30 minutes
- Users see how many attempts remain before lockout
- Locked accounts display the time until unlock
- Successful login resets the counter

**Testing:**
1. Try logging in with wrong password
2. Note the error message shows "X attempts remaining"
3. After 5 failed attempts, account locks for 30 minutes
4. Error message shows "retry_in_mins" and suggests password reset
5. After 30 minutes, lock expires automatically

**API Response Examples:**
```json
// Failed attempt with remaining tries
{
  "error": "Invalid credentials",
  "attempts_remaining": 3
}

// Account locked
{
  "error": "Account has been locked due to too many failed login attempts...",
  "locked_until": "2024-01-20T15:30:00Z",
  "retry_in_mins": 30
}
```

### 2. Password Reset Flow

**How it works:**
1. User clicks "Forgot Password?" on login page
2. Enters their email address in the modal
3. System generates a secure token (32-byte random, hashed with bcrypt)
4. Token is valid for 1 hour
5. Email is sent with reset link (if SMTP is configured)
6. User clicks link → redirected to reset page with token
7. User enters new password → account updated, failed attempts cleared

**Testing WITHOUT Email (Default):**
- Click "Forgot Password?" → Modal appears
- Enter any email → Success message appears
- No email sent (graceful degradation)
- Backend logs: "Password reset requested but email service is not configured"

**Testing WITH Email (Configure SMTP):**
1. Add SMTP settings to `.env`:
   ```
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=your-email@gmail.com
   SMTP_PASSWORD=your-app-password
   SMTP_FROM_EMAIL=noreply@hawsvolunteers.org
   SMTP_FROM_NAME=Haws Volunteers
   FRONTEND_URL=http://localhost:5173
   ```
2. Request password reset
3. Check email for reset link
4. Click link → opens `/reset-password?token=...`
5. Enter new password (twice)
6. Submit → redirected to login
7. Login with new password → success

**Security Features:**
- Token never stored in plaintext (hashed with bcrypt)
- Tokens expire after 1 hour
- Email responses don't reveal if account exists (anti-enumeration)
- Reset clears failed login attempts and account locks

### 3. Email Notification Preferences

**How it works:**
- New Settings page at `/settings`
- Users can toggle email notifications on/off
- Default: OFF (opt-in)
- Preference saved to database
- Used by admin when sending announcements

**Testing:**
1. Login to application
2. Click "Settings" in navigation
3. See "Email Notifications" section
4. Toggle "Receive Announcement Emails"
5. Click "Save Preferences"
6. Success message appears
7. Refresh page → preference persists

**API Endpoints:**
- `GET /api/email-preferences` - Get current setting
- `PUT /api/email-preferences` - Update setting

**Note:** Security emails (password reset) are ALWAYS sent regardless of this preference.

## Database Schema Changes

New fields added to `users` table:
```sql
failed_login_attempts  INTEGER DEFAULT 0
locked_until           TIMESTAMP NULL
reset_token            TEXT
reset_token_expiry     TIMESTAMP NULL
email_notifications_enabled BOOLEAN DEFAULT false
```

These are automatically migrated on server start.

## Configuration

### Minimal Configuration (No Email)
The application works without email configuration:
- Login restrictions work
- Password reset UI works (but no email sent)
- Settings page works
- Graceful degradation with clear logging

### Full Configuration (With Email)
Add to `.env` file:
```bash
# Email Configuration (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=noreply@hawsvolunteers.org
SMTP_FROM_NAME=Haws Volunteers

# Frontend URL (for password reset links)
FRONTEND_URL=http://localhost:5173
```

**For Gmail:**
1. Use App Password (not regular password)
2. Enable 2FA in Google Account
3. Generate App Password: https://myaccount.google.com/apppasswords
4. Use that password in SMTP_PASSWORD

**For SendGrid/Other Providers:**
- Use their SMTP settings
- Port 587 (STARTTLS) or 465 (SSL/TLS)
- Use API key as password

## UI Screenshots

### Login Page with "Forgot Password?" Link
- New link appears below password field when in login mode
- Opens modal for email entry

### Password Reset Modal
- Clean, centered modal
- Email input field
- Success/error messages
- Cancel and Submit buttons

### Password Reset Page
- Accessed via email link
- Two password fields (new + confirm)
- Validates passwords match
- Shows token expiry errors
- Auto-redirects to login on success

### Settings Page
- New page at `/settings`
- Toggle switch for email notifications
- Clear descriptions
- Info box explaining what notifications include
- Save button with loading state

### Enhanced Login Errors
- Shows "X attempts remaining" on failed login
- Shows "Account locked for Y minutes" when locked
- Suggests password reset when locked

## Testing Checklist

### Login Restrictions
- [ ] Failed login increments counter
- [ ] Error shows attempts remaining
- [ ] Account locks after 5 attempts
- [ ] Lock message shows time remaining
- [ ] Successful login resets counter
- [ ] Lock expires after 30 minutes

### Password Reset (Without Email)
- [ ] "Forgot Password?" link appears on login
- [ ] Modal opens with email input
- [ ] Success message appears on submit
- [ ] No email sent (graceful)
- [ ] Backend logs show "not configured"

### Password Reset (With Email) 
- [ ] Email is received with reset link
- [ ] Link opens reset page with token
- [ ] Can reset password successfully
- [ ] Passwords must match
- [ ] Minimum 8 characters enforced
- [ ] Token expiry is validated
- [ ] Redirect to login after success
- [ ] Can login with new password
- [ ] Failed attempts cleared after reset

### Email Preferences
- [ ] Settings page loads
- [ ] Current preference displayed
- [ ] Toggle changes state
- [ ] Save updates database
- [ ] Success message appears
- [ ] Preference persists on refresh
- [ ] Settings link in navigation works

## Code Structure

### Backend
```
internal/
├── email/
│   └── email.go               # SMTP service, email templates
├── handlers/
│   ├── auth.go                # Login with attempt tracking
│   └── password_reset.go      # Reset handlers
└── models/
    └── models.go              # User model with new fields
```

### Frontend
```
frontend/src/
├── pages/
│   ├── Login.tsx              # Enhanced with forgot password
│   ├── Login.css              # Styles for modal
│   ├── ResetPassword.tsx      # Password reset page
│   ├── Settings.tsx           # Email preferences
│   └── Settings.css           # Settings page styles
├── components/
│   └── Navigation.tsx         # Added Settings link
└── App.tsx                    # New routes
```

## API Endpoints

### Public (No Auth Required)
- `POST /api/login` - Enhanced with attempt tracking
- `POST /api/request-password-reset` - Request reset email
- `POST /api/reset-password` - Reset password with token

### Protected (Auth Required)
- `GET /api/email-preferences` - Get user's email preference
- `PUT /api/email-preferences` - Update email preference

## Security Considerations

1. **Token Security**
   - 32-byte cryptographically secure random tokens
   - Hashed with bcrypt before storage
   - Never stored or transmitted in plaintext (except in email)

2. **Email Enumeration Protection**
   - Password reset always returns success message
   - Doesn't reveal if email exists in system

3. **Rate Limiting**
   - Login attempts limited per user
   - 30-minute lockout prevents brute force

4. **Token Expiry**
   - Reset tokens expire after 1 hour
   - Expired tokens return clear error message

5. **Password Requirements**
   - Minimum 8 characters
   - Maximum 72 characters (bcrypt limit)
   - Validated on frontend and backend

## Future Enhancements

Possible improvements:
1. Admin panel to manually unlock accounts
2. Email notifications when account is locked
3. Configurable lockout duration
4. IP-based rate limiting
5. 2FA/MFA support
6. Password strength requirements
7. Password history (prevent reuse)
8. Account recovery via security questions

## Troubleshooting

### Email not sending
- Check SMTP credentials in `.env`
- Verify firewall allows outbound SMTP
- Check backend logs for detailed errors
- Test with telnet: `telnet smtp.gmail.com 587`

### Token validation fails
- Token may have expired (1 hour limit)
- Request new reset email
- Check system time is synchronized

### Account won't unlock
- Wait full 30 minutes
- Check LockedUntil timestamp in database
- Restart application if needed (locks persist in DB)

### Settings not saving
- Check browser console for errors
- Verify authentication token is valid
- Check backend logs for database errors
