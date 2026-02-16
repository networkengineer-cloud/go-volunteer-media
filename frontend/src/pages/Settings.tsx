import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { getPasswordStrength } from '../utils/passwordStrength';
import './Settings.css';

const Settings: React.FC = () => {
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [hideEmail, setHideEmail] = useState(false);
  const [hidePhoneNumber, setHidePhoneNumber] = useState(false);
  const [emailNotificationsEnabled, setEmailNotificationsEnabled] = useState(false);
  const [showLengthOfStay, setShowLengthOfStay] = useState(false);
  const [loading, setLoading] = useState(true);
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingNotifications, setSavingNotifications] = useState(false);
  const [savingDisplay, setSavingDisplay] = useState(false);
  const [error, setError] = useState('');
  const [phoneError, setPhoneError] = useState('');
  const [success, setSuccess] = useState('');
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [savingPassword, setSavingPassword] = useState(false);
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const navigate = useNavigate();
  const { user } = useAuth();
  const { showToast } = useToast();

  const validatePhoneNumber = (phone: string): boolean => {
    if (!phone.trim()) {
      setPhoneError('');
      return true; // Phone is optional
    }
    
    // Accept various US phone formats: (123) 456-7890, 123-456-7890, 1234567890, +1-123-456-7890, etc.
    const phoneRegex = /^(\+?1[-.]?)?(\([0-9]{3}\)|[0-9]{3})[-.]?[0-9]{3}[-.]?[0-9]{4}$/;
    if (!phoneRegex.test(phone)) {
      setPhoneError('Phone number must be in format (123) 456-7890 or similar');
      return false;
    }
    setPhoneError('');
    return true;
  };

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [userRes, prefsRes] = await Promise.all([
        authApi.getCurrentUser(),
        authApi.getEmailPreferences(),
      ]);
      setFirstName(userRes.data.first_name || '');
      setLastName(userRes.data.last_name || '');
      setEmail(userRes.data.email);
      setPhoneNumber(userRes.data.phone_number || '');
      setHideEmail(userRes.data.hide_email || false);
      setHidePhoneNumber(userRes.data.hide_phone_number || false);
      setEmailNotificationsEnabled(prefsRes.data.email_notifications_enabled || false);
      setShowLengthOfStay(prefsRes.data.show_length_of_stay || false);
      setError('');
    } catch (err: unknown) {
      console.error('Failed to load settings:', err);
      setError('Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveProfile = async () => {
    if (!email.trim()) {
      showToast('Email cannot be empty', 'error');
      return;
    }

    if (!validatePhoneNumber(phoneNumber)) {
      return;
    }

    setSavingProfile(true);
    setError('');
    setSuccess('');

    try {
      await authApi.updateCurrentUserProfile({
        first_name: firstName,
        last_name: lastName,
        email,
        phone_number: phoneNumber,
        hide_email: hideEmail,
        hide_phone_number: hidePhoneNumber,
      });
      showToast('Profile updated successfully!', 'success');
    } catch (err: unknown) {
      console.error('Failed to save profile:', err);
      const axiosError = err as { response?: { data?: { error?: string } } };
      showToast(axiosError.response?.data?.error || 'Failed to save profile', 'error');
    } finally {
      setSavingProfile(false);
    }
  };

  const handleSaveNotifications = async () => {
    setSavingNotifications(true);
    setError('');
    setSuccess('');

    try {
      await authApi.updateEmailPreferences(emailNotificationsEnabled, showLengthOfStay);
      showToast('Email preferences saved successfully!', 'success');
    } catch (err: unknown) {
      console.error('Failed to save preferences:', err);
      const axiosError = err as { response?: { data?: { error?: string } } };
      showToast(axiosError.response?.data?.error || 'Failed to save preferences', 'error');
    } finally {
      setSavingNotifications(false);
    }
  };

  const handleSaveDisplayPreferences = async () => {
    setSavingDisplay(true);
    setError('');
    setSuccess('');

    try {
      await authApi.updateEmailPreferences(emailNotificationsEnabled, showLengthOfStay);
      showToast('Display preferences saved successfully!', 'success');
    } catch (err: unknown) {
      console.error('Failed to save display preferences:', err);
      const axiosError = err as { response?: { data?: { error?: string } } };
      showToast(axiosError.response?.data?.error || 'Failed to save display preferences', 'error');
    } finally {
      setSavingDisplay(false);
    }
  };

  const handleChangePassword = async () => {
    if (!user) return;

    if (!currentPassword) {
      showToast('Please enter your current password', 'error');
      return;
    }
    if (!newPassword) {
      showToast('Please enter a new password', 'error');
      return;
    }
    if (newPassword.length < 8) {
      showToast('New password must be at least 8 characters', 'error');
      return;
    }
    if (newPassword === currentPassword) {
      showToast('New password must be different from current password', 'error');
      return;
    }
    if (newPassword !== confirmPassword) {
      showToast('New passwords do not match', 'error');
      return;
    }

    setSavingPassword(true);
    try {
      await authApi.changePassword(user.id, currentPassword, newPassword);
      showToast('Password changed successfully!', 'success');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: unknown) {
      console.error('Failed to change password:', err);
      const axiosError = err as { response?: { status?: number; data?: { error?: string } } };
      if (axiosError.response?.status === 401) {
        showToast('Current password is incorrect', 'error');
      } else {
        showToast(axiosError.response?.data?.error || 'Failed to change password', 'error');
      }
    } finally {
      setSavingPassword(false);
    }
  };

  const passwordStrength = newPassword ? getPasswordStrength(newPassword) : null;

  if (loading) {
    return (
      <div className="settings-container">
        <div className="settings-card">
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="settings-container">
      <div className="settings-card">
        <div className="settings-header">
          <h1>Settings</h1>
          <button onClick={() => navigate('/dashboard')} className="btn-back">
            ← Back to Dashboard
          </button>
        </div>

        {/* Profile Information Section */}
        <div className="settings-section">
          <h2>Profile Information</h2>
          <p className="settings-description">
            Manage your account information, email, phone number, and control who can see this information.
          </p>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="firstName">
                <strong>First Name</strong>
              </label>
              <p className="setting-help">
                Your first name as you'd like it to appear.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <input
                id="firstName"
                type="text"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                disabled={savingProfile}
                className="setting-input"
                placeholder="First name"
                maxLength={100}
              />
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="lastName">
                <strong>Last Name</strong>
              </label>
              <p className="setting-help">
                Your last name as you'd like it to appear.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <input
                id="lastName"
                type="text"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                disabled={savingProfile}
                className="setting-input"
                placeholder="Last name"
                maxLength={100}
              />
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="email">
                <strong>Email Address</strong>
              </label>
              <p className="setting-help">
                Your email is used for account recovery and notifications.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={savingProfile}
                className="setting-input"
                placeholder="your.email@example.com"
              />
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="phone">
                <strong>Phone Number</strong>
              </label>
              <p className="setting-help">
                Your phone number helps other volunteers contact you (optional).
              </p>
            </div>
            <div className="setting-input-wrapper">
              <input
                id="phone"
                type="tel"
                value={phoneNumber}
                onChange={(e) => {
                  setPhoneNumber(e.target.value);
                  validatePhoneNumber(e.target.value);
                }}
                disabled={savingProfile}
                className="setting-input"
                placeholder="(123) 456-7890"
              />
              {phoneError && <div className="error" style={{ marginTop: '8px' }}>{phoneError}</div>}
            </div>
          </div>

          {error && <div className="error">{error}</div>}
          {success && <div className="success">{success}</div>}

          <div className="settings-actions">
            <button
              onClick={handleSaveProfile}
              className="btn-primary"
              disabled={savingProfile}
            >
              {savingProfile ? 'Saving Profile...' : 'Save Profile'}
            </button>
          </div>
        </div>

        {/* Privacy Settings Section */}
        <div className="settings-section">
          <h2>Privacy Settings</h2>
          <p className="settings-description">
            Control who can see your email and phone number. Administrators and group admins will always have access.
          </p>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="hide-email">
                <strong>Hide Email from Other Users</strong>
              </label>
              <p className="setting-help">
                When enabled, only admins can see your email address.
              </p>
            </div>
            <div className="toggle-wrapper">
              <label className="toggle">
                <input
                  id="hide-email"
                  type="checkbox"
                  checked={hideEmail}
                  onChange={(e) => setHideEmail(e.target.checked)}
                  disabled={savingProfile}
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="hide-phone">
                <strong>Hide Phone from Other Users</strong>
              </label>
              <p className="setting-help">
                When enabled, only admins and group admins can see your phone number.
              </p>
            </div>
            <div className="toggle-wrapper">
              <label className="toggle">
                <input
                  id="hide-phone"
                  type="checkbox"
                  checked={hidePhoneNumber}
                  onChange={(e) => setHidePhoneNumber(e.target.checked)}
                  disabled={savingProfile}
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>

          <div className="settings-actions">
            <button
              onClick={handleSaveProfile}
              className="btn-primary"
              disabled={savingProfile}
            >
              {savingProfile ? 'Saving...' : 'Update Privacy Settings'}
            </button>
          </div>
        </div>

        {/* Email Notifications Section */}
        <div className="settings-section">
          <h2>Email Notifications</h2>
          <p className="settings-description">
            Control how you receive email notifications from Haws Volunteers.
          </p>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="email-notifications">
                <strong>Receive Announcement Emails</strong>
              </label>
              <p className="setting-help">
                Get notified via email when new announcements are posted to your groups.
              </p>
            </div>
            <div className="toggle-wrapper">
              <label className="toggle">
                <input
                  id="email-notifications"
                  type="checkbox"
                  checked={emailNotificationsEnabled}
                  onChange={(e) => setEmailNotificationsEnabled(e.target.checked)}
                  disabled={savingNotifications}
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>

          <div className="settings-actions">
            <button
              onClick={handleSaveNotifications}
              className="btn-primary"
              disabled={savingNotifications}
            >
              {savingNotifications ? 'Saving...' : 'Save Email Preferences'}
            </button>
          </div>
        </div>

        {/* Display Preferences Section */}
        <div className="settings-section">
          <h2>Display Preferences</h2>
          <p className="settings-description">
            Customize how information is displayed on the animals page.
          </p>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="show-length-of-stay">
                <strong>Show Length of Stay</strong>
              </label>
              <p className="setting-help">
                Display how long each animal has been at the shelter on the animals page.
              </p>
            </div>
            <div className="toggle-wrapper">
              <label className="toggle">
                <input
                  id="show-length-of-stay"
                  type="checkbox"
                  checked={showLengthOfStay}
                  onChange={(e) => setShowLengthOfStay(e.target.checked)}
                  disabled={savingDisplay}
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>

          <div className="settings-actions">
            <button
              onClick={handleSaveDisplayPreferences}
              className="btn-primary"
              disabled={savingDisplay}
            >
              {savingDisplay ? 'Saving...' : 'Save Display Preferences'}
            </button>
          </div>
        </div>

        {/* Change Password Section */}
        <div className="settings-section">
          <h2>Change Password</h2>
          <p className="settings-description">
            Update your account password. You'll need to enter your current password for verification.
          </p>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="current-password">
                <strong>Current Password</strong>
              </label>
              <p className="setting-help">
                Enter your existing password to verify your identity.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <div className="password-input-wrapper">
                <input
                  id="current-password"
                  type={showCurrentPassword ? 'text' : 'password'}
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  disabled={savingPassword}
                  className="setting-input"
                  placeholder="Current password"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  className="password-toggle"
                  onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                  aria-label={showCurrentPassword ? 'Hide password' : 'Show password'}
                >
                  {showCurrentPassword ? (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" /><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" /><line x1="1" y1="1" x2="23" y2="23" /></svg>
                  ) : (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
                  )}
                </button>
              </div>
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="new-password">
                <strong>New Password</strong>
              </label>
              <p className="setting-help">
                Choose a strong password with at least 8 characters.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <div className="password-input-wrapper">
                <input
                  id="new-password"
                  type={showNewPassword ? 'text' : 'password'}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={savingPassword}
                  className="setting-input"
                  placeholder="New password"
                  autoComplete="new-password"
                />
                <button
                  type="button"
                  className="password-toggle"
                  onClick={() => setShowNewPassword(!showNewPassword)}
                  aria-label={showNewPassword ? 'Hide password' : 'Show password'}
                >
                  {showNewPassword ? (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" /><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" /><line x1="1" y1="1" x2="23" y2="23" /></svg>
                  ) : (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
                  )}
                </button>
              </div>
              {passwordStrength && (
                <div className="strength-bar">
                  <div
                    className={`strength-fill strength-${passwordStrength.strength}`}
                  />
                  <span className="strength-label" style={{ color: passwordStrength.color }}>
                    {passwordStrength.label}
                  </span>
                </div>
              )}
            </div>
          </div>

          <div className="setting-item">
            <div className="setting-info">
              <label htmlFor="confirm-password">
                <strong>Confirm New Password</strong>
              </label>
              <p className="setting-help">
                Re-enter your new password to confirm.
              </p>
            </div>
            <div className="setting-input-wrapper">
              <div className="password-input-wrapper">
                <input
                  id="confirm-password"
                  type={showConfirmPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={savingPassword}
                  className="setting-input"
                  placeholder="Confirm new password"
                  autoComplete="new-password"
                />
                <button
                  type="button"
                  className="password-toggle"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
                >
                  {showConfirmPassword ? (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" /><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" /><line x1="1" y1="1" x2="23" y2="23" /></svg>
                  ) : (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
                  )}
                </button>
              </div>
              {confirmPassword && newPassword !== confirmPassword && (
                <div className="error" style={{ marginTop: '8px' }}>Passwords do not match</div>
              )}
            </div>
          </div>

          <div className="settings-actions">
            <button
              onClick={handleChangePassword}
              className="btn-primary"
              disabled={savingPassword || !currentPassword || !newPassword || !confirmPassword}
            >
              {savingPassword ? 'Changing Password...' : 'Change Password'}
            </button>
          </div>
        </div>

        <div className="settings-section">
          <h2>About Email Notifications</h2>
          <div className="info-box">
            <p>
              <strong>When enabled:</strong> You will receive email notifications when:
            </p>
            <ul>
              <li>New announcements are posted to groups you're a member of</li>
              <li>Important updates are shared by administrators</li>
            </ul>
            <p>
              <strong>Note:</strong> You will always receive password reset emails and account security notifications regardless of this setting.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Settings;
