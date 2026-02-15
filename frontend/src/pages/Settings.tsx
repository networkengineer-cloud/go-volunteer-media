import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '../api/client';
import { useToast } from '../hooks/useToast';
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
  const navigate = useNavigate();
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
