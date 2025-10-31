import React, { useEffect, useState } from 'react';
import { settingsApi } from '../api/client';
import './SettingsPage.css';

const SettingsPage: React.FC = () => {
  const [heroImageUrl, setHeroImageUrl] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const response = await settingsApi.getAll();
      setHeroImageUrl(response.data.hero_image_url || '');
    } catch (error) {
      console.error('Failed to load settings:', error);
      setMessage('Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage('');

    try {
      await settingsApi.update('hero_image_url', heroImageUrl);
      setMessage('Settings saved successfully!');
      setTimeout(() => setMessage(''), 3000);
    } catch (error) {
      console.error('Failed to save settings:', error);
      setMessage('Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading settings...</div>;
  }

  return (
    <div className="settings-page">
      <div className="settings-container">
        <h1>Site Settings</h1>
        <p className="settings-subtitle">Configure site-wide settings</p>

        <form onSubmit={handleSave} className="settings-form">
          <div className="form-group">
            <label htmlFor="heroImageUrl">
              Hero Image URL
              <span className="label-hint">The background image for the home page hero section</span>
            </label>
            <input
              type="url"
              id="heroImageUrl"
              value={heroImageUrl}
              onChange={(e) => setHeroImageUrl(e.target.value)}
              placeholder="https://example.com/image.jpg"
              required
            />
            {heroImageUrl && (
              <div className="image-preview">
                <p className="preview-label">Preview:</p>
                <div 
                  className="preview-hero"
                  style={{ backgroundImage: `url('${heroImageUrl}')` }}
                >
                  <div className="preview-overlay">
                    <h2>Help Pets. Help People.</h2>
                  </div>
                </div>
              </div>
            )}
            <p className="field-help">
              Use a high-quality image (min 1920x660px). 
              Try <a href="https://unsplash.com/s/photos/dog" target="_blank" rel="noopener noreferrer">Unsplash</a> for free stock photos.
            </p>
          </div>

          {message && (
            <div className={`message ${message.includes('success') ? 'success' : 'error'}`}>
              {message}
            </div>
          )}

          <div className="form-actions">
            <button type="submit" className="btn-save" disabled={saving}>
              {saving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default SettingsPage;
