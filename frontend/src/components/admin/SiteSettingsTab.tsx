import React, { useEffect, useState } from 'react';
import { settingsApi } from '../../api/client';
import '../../pages/SettingsPage.css';
import '../../pages/Home.css'; // Import Home.css to reuse hero styles

const SiteSettingsTab: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  
  // Site name settings
  const [siteName, setSiteName] = useState('');
  const [siteShortName, setSiteShortName] = useState('');
  const [siteDescription, setSiteDescription] = useState('');
  const [savingText, setSavingText] = useState(false);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const response = await settingsApi.getAll();
      const settings = response.data;
      setPreviewUrl(settings.hero_image_url || '');
      setSiteName(settings.site_name || 'MyHAWS');
      setSiteShortName(settings.site_short_name || 'MyHAWS');
      setSiteDescription(settings.site_description || 'MyHAWS Volunteer Portal - Internal volunteer management system');
    } catch (error) {
      console.error('Failed to load settings:', error);
      setMessage('Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      setMessage('Please select an image file');
      return;
    }

    // Validate file size (max 5MB)
    if (file.size > 5 * 1024 * 1024) {
      setMessage('Image size must be less than 5MB');
      return;
    }

    setSelectedFile(file);
    
    // Create preview URL
    const objectUrl = URL.createObjectURL(file);
    setPreviewUrl(objectUrl);
    setMessage('');
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!selectedFile) {
      setMessage('Please select an image to upload');
      return;
    }

    setSaving(true);
    setMessage('');

    try {
      // Upload the image
      const uploadResponse = await settingsApi.uploadHeroImage(selectedFile);
      const imageUrl = uploadResponse.data.url;

      // Update the setting with the new URL
      await settingsApi.update('hero_image_url', imageUrl);
      
      setPreviewUrl(imageUrl);
      setSelectedFile(null);
      setMessage('Hero image updated successfully!');
      setTimeout(() => setMessage(''), 3000);
    } catch (error) {
      console.error('Failed to save settings:', error);
      setMessage('Failed to upload image');
    } finally {
      setSaving(false);
    }
  };

  const handleSaveTextSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate inputs
    if (!siteName.trim()) {
      setMessage('Site name is required');
      return;
    }
    
    if (siteName.length > 100) {
      setMessage('Site name must be 100 characters or less');
      return;
    }
    
    if (siteDescription.length > 500) {
      setMessage('Site description must be 500 characters or less');
      return;
    }

    setSavingText(true);
    setMessage('');

    try {
      // Update all text settings
      await Promise.all([
        settingsApi.update('site_name', siteName.trim()),
        settingsApi.update('site_short_name', siteShortName.trim()),
        settingsApi.update('site_description', siteDescription.trim()),
      ]);
      
      setMessage('Site settings updated successfully!');
      setTimeout(() => setMessage(''), 3000);
      
      // Reload page after a delay to show updated branding
      setTimeout(() => window.location.reload(), 1500);
    } catch (error) {
      console.error('Failed to save text settings:', error);
      setMessage('Failed to update settings');
    } finally {
      setSavingText(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading settings...</div>;
  }

  return (
    <div className="tab-panel">
      <h2>Site Configuration</h2>
      
      {/* Site Name & Description Settings */}
      <form onSubmit={handleSaveTextSettings} className="settings-form" style={{ marginBottom: '2rem' }}>
        <h3>Branding</h3>
        <div className="form-group">
          <label htmlFor="siteName">
            Site Name
            <span className="label-hint">The full name displayed throughout the site</span>
          </label>
          <input
            type="text"
            id="siteName"
            value={siteName}
            onChange={(e) => setSiteName(e.target.value)}
            maxLength={100}
            required
          />
          <p className="field-help">
            Displayed in navigation, authentication pages, and email templates (max 100 characters)
          </p>
        </div>

        <div className="form-group">
          <label htmlFor="siteShortName">
            Short Name
            <span className="label-hint">Abbreviated name for compact displays</span>
          </label>
          <input
            type="text"
            id="siteShortName"
            value={siteShortName}
            onChange={(e) => setSiteShortName(e.target.value)}
            maxLength={50}
            required
          />
          <p className="field-help">
            Used in welcome messages and compact displays (max 50 characters)
          </p>
        </div>

        <div className="form-group">
          <label htmlFor="siteDescription">
            Site Description
            <span className="label-hint">Brief description for metadata and SEO</span>
          </label>
          <textarea
            id="siteDescription"
            value={siteDescription}
            onChange={(e) => setSiteDescription(e.target.value)}
            maxLength={500}
            rows={3}
            required
          />
          <p className="field-help">
            Used in page metadata and site descriptions (max 500 characters)
          </p>
        </div>

        <div className="form-actions">
          <button type="submit" className="btn-save" disabled={savingText}>
            {savingText ? 'Saving...' : 'Save Branding Settings'}
          </button>
        </div>
      </form>

      {/* Hero Image Settings */}
      <form onSubmit={handleSave} className="settings-form">
        <h3>Hero Image</h3>
        <div className="form-group">
          <label htmlFor="heroImage">
            Background Image
            <span className="label-hint">The background image for the home page hero section</span>
          </label>
          <input
            type="file"
            id="heroImage"
            accept="image/jpeg,image/png,image/webp"
            onChange={handleFileSelect}
          />
          {previewUrl && (
            <div className="image-preview">
              <p className="preview-label">Preview:</p>
              <div 
                className="hero preview-hero"
                style={{ backgroundImage: `url(${previewUrl})` }}
              >
                <div className="hero-overlay" />
                <div className="hero-content">
                  <h1>Help Pets. Help People.</h1>
                </div>
              </div>
            </div>
          )}
          <p className="field-help">
            Upload a high-quality image (recommended: 1920x660px or larger). 
            Accepts JPG, PNG, or WebP files up to 5MB.
          </p>
        </div>

        {message && (
          <div className={`message ${message.includes('success') ? 'success' : 'error'}`}>
            {message}
          </div>
        )}

        <div className="form-actions">
          <button type="submit" className="btn-save" disabled={saving || !selectedFile}>
            {saving ? 'Uploading...' : 'Upload & Save'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default SiteSettingsTab;
