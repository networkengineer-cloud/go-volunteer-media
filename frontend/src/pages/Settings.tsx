import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '../api/client';
import './Settings.css';

const SUCCESS_MESSAGE_TIMEOUT = 3000; // milliseconds

const Settings: React.FC = () => {
  const [emailNotificationsEnabled, setEmailNotificationsEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    loadPreferences();
  }, []);

  const loadPreferences = async () => {
    try {
      const response = await authApi.getEmailPreferences();
      setEmailNotificationsEnabled(response.data.email_notifications_enabled || false);
      setError('');
    } catch (err: unknown) {
      console.error('Failed to load preferences:', err);
      setError('Failed to load preferences');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    setSuccess('');

    try {
      await authApi.updateEmailPreferences(emailNotificationsEnabled);
      setSuccess('Preferences saved successfully!');
      setTimeout(() => setSuccess(''), SUCCESS_MESSAGE_TIMEOUT);
    } catch (err: unknown) {
      console.error('Failed to save preferences:', err);
      const axiosError = err as { response?: { data?: { error?: string } } };
      setError(axiosError.response?.data?.error || 'Failed to save preferences');
    } finally {
      setSaving(false);
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
                  disabled={saving}
                />
                <span className="toggle-slider"></span>
              </label>
            </div>
          </div>

          {error && <div className="error">{error}</div>}
          {success && <div className="success">{success}</div>}

          <div className="settings-actions">
            <button
              onClick={handleSave}
              className="btn-primary"
              disabled={saving}
            >
              {saving ? 'Saving...' : 'Save Preferences'}
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
