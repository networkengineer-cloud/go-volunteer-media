import React, { useEffect, useState } from 'react';
import { settingsApi } from '../../api/client';
import '../SettingsPage.css';
import '../Home.css'; // Import Home.css to reuse hero styles

const SiteSettingsTab: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const response = await settingsApi.getAll();
      const currentUrl = response.data.hero_image_url || '';
      setPreviewUrl(currentUrl);
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

  if (loading) {
    return <div className="loading">Loading settings...</div>;
  }

  return (
    <div className="tab-panel">
      <h2>Site Configuration</h2>
      <form onSubmit={handleSave} className="settings-form">
        <div className="form-group">
          <label htmlFor="heroImage">
            Hero Image
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
