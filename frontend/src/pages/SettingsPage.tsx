import React, { useEffect, useState } from 'react';
import { settingsApi } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import Button from '../components/Button';
import SkeletonLoader from '../components/SkeletonLoader';
import './SettingsPage.css';
import '../pages/Home.css'; // Import Home.css to reuse hero styles

const SettingsPage: React.FC = () => {
  const toast = useToast();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

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
      toast.showError('Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      toast.showError('Please select an image file (JPG, PNG, or WebP)');
      return;
    }

    // Validate file size (max 5MB)
    if (file.size > 5 * 1024 * 1024) {
      toast.showError(`Image size must be under 5MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
      return;
    }

    setSelectedFile(file);
    
    // Create preview URL
    const objectUrl = URL.createObjectURL(file);
    setPreviewUrl(objectUrl);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!selectedFile) {
      toast.showWarning('Please select an image to upload');
      return;
    }

    setSaving(true);

    try {
      // Upload the image
      const uploadResponse = await settingsApi.uploadHeroImage(selectedFile);
      const imageUrl = uploadResponse.data.url;

      // Update the setting with the new URL
      await settingsApi.update('hero_image_url', imageUrl);
      
      setPreviewUrl(imageUrl);
      setSelectedFile(null);
      toast.showSuccess('Hero image updated successfully!');
      
      // Clear file input
      const fileInput = document.getElementById('heroImage') as HTMLInputElement;
      if (fileInput) fileInput.value = '';
    } catch (error: any) {
      console.error('Failed to save settings:', error);
      const errorMsg = error.response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="settings-page">
        <div className="settings-container">
          <SkeletonLoader variant="text" width="40%" height="2rem" />
          <SkeletonLoader variant="text" width="60%" />
          <div style={{ marginTop: '2rem' }}>
            <SkeletonLoader variant="rectangular" height="400px" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="settings-page">
      <div className="settings-container">
        <h1>Site Settings</h1>
        <p className="settings-subtitle">Configure site-wide settings</p>

        <form onSubmit={handleSave} className="settings-form">
          <div className="form-field">
            <label htmlFor="heroImage" className="form-field__label">
              Hero Image
            </label>
            <input
              type="file"
              id="heroImage"
              accept="image/jpeg,image/png,image/webp"
              onChange={handleFileSelect}
              className="form-field__input"
              disabled={saving}
              aria-label="Upload hero image"
            />
            <p className="form-field__helper">
              Upload a high-quality image (recommended: 1920x660px or larger). 
              Accepts JPG, PNG, or WebP files up to 5MB.
            </p>
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
          </div>

          <div className="form-actions">
            <Button
              type="submit"
              variant="primary"
              loading={saving}
              disabled={!selectedFile}
              fullWidth={false}
            >
              Upload & Save
            </Button>
            {selectedFile && (
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setSelectedFile(null);
                  loadSettings();
                  const fileInput = document.getElementById('heroImage') as HTMLInputElement;
                  if (fileInput) fileInput.value = '';
                }}
                disabled={saving}
              >
                Cancel
              </Button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
};

export default SettingsPage;
