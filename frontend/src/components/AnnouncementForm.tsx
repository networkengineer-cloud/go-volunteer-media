import React, { useState, useRef } from 'react';
import { updatesApi, groupsApi } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import './AnnouncementForm.css';

interface AnnouncementFormProps {
  groupId: number;
  onSuccess?: () => void;
  onCancel?: () => void;
}

const AnnouncementForm: React.FC<AnnouncementFormProps> = ({ groupId, onSuccess, onCancel }) => {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [imageUrl, setImageUrl] = useState('');
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      toast.showError('Please select an image file (JPG, PNG, or GIF)');
      return;
    }

    // Validate file size (max 10MB)
    if (file.size > 10 * 1024 * 1024) {
      toast.showError(`Image size must be under 10MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
      return;
    }

    setUploading(true);
    try {
      const res = await groupsApi.uploadImage(file);
      setImageUrl(res.data.url);
      toast.showSuccess('Image uploaded successfully!');
    } catch (error) {
      console.error('Upload error:', error);
      const err = error as { response?: { data?: { error?: string } } };
      const errorMsg = err.response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUploading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      toast.showWarning('Please enter a title for your announcement');
      return;
    }

    if (!content.trim()) {
      toast.showWarning('Please enter content for your announcement');
      return;
    }

    setSubmitting(true);
    try {
      await updatesApi.create(groupId, title.trim(), content.trim(), imageUrl || undefined);
      toast.showSuccess('Announcement posted successfully!');
      
      // Reset form
      setTitle('');
      setContent('');
      setImageUrl('');
      
      if (onSuccess) {
        onSuccess();
      }
    } catch (error) {
      console.error('Failed to post announcement:', error);
      const err = error as { response?: { data?: { error?: string } } };
      const errorMsg = err.response?.data?.error || 'Failed to post announcement. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveImage = () => {
    setImageUrl('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <form className="announcement-form" onSubmit={handleSubmit}>
      <div className="announcement-form__header">
        <h3>Post Announcement</h3>
        <p className="announcement-form__description">
          Share important updates with everyone in the group
        </p>
      </div>

      <div className="announcement-form__fields">
        {/* Title field */}
        <div className="announcement-form__field">
          <label htmlFor="announcement-title" className="announcement-form__label">
            Title <span className="announcement-form__required">*</span>
          </label>
          <input
            id="announcement-title"
            type="text"
            className="announcement-form__input"
            placeholder="e.g., Group Meeting This Saturday"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
            required
            aria-required="true"
            disabled={submitting}
          />
          <span className="announcement-form__char-count" aria-live="polite">
            {title.length}/200 characters
          </span>
        </div>

        {/* Content field */}
        <div className="announcement-form__field">
          <label htmlFor="announcement-content" className="announcement-form__label">
            Content <span className="announcement-form__required">*</span>
          </label>
          <textarea
            id="announcement-content"
            className="announcement-form__textarea"
            placeholder="Share details about your announcement..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={5}
            required
            aria-required="true"
            disabled={submitting}
          />
        </div>

        {/* Image upload */}
        <div className="announcement-form__field">
          <label className="announcement-form__label">
            Image (optional)
          </label>
          
          {!imageUrl && (
            <div className="announcement-form__upload-section">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleImageUpload}
                disabled={uploading || submitting}
                className="announcement-form__file-input"
                id="announcement-image"
                aria-label="Upload announcement image"
              />
              <label htmlFor="announcement-image" className="announcement-form__upload-button">
                {uploading ? (
                  <>
                    <div className="announcement-form__spinner" aria-hidden="true" />
                    <span>Uploading...</span>
                  </>
                ) : (
                  <>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                      <circle cx="8.5" cy="8.5" r="1.5" />
                      <path d="M21 15l-5-5L5 21" />
                    </svg>
                    <span>Upload Image</span>
                  </>
                )}
              </label>
              <p className="announcement-form__upload-hint">
                JPG, PNG or GIF. Max 10MB.
              </p>
            </div>
          )}

          {imageUrl && (
            <div className="announcement-form__image-preview">
              <img src={imageUrl} alt="Announcement preview" />
              <button
                type="button"
                className="announcement-form__remove-image"
                onClick={handleRemoveImage}
                disabled={submitting}
                aria-label="Remove image"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Form actions */}
      <div className="announcement-form__actions">
        {onCancel && (
          <button
            type="button"
            className="announcement-form__button announcement-form__button--secondary"
            onClick={onCancel}
            disabled={submitting}
          >
            Cancel
          </button>
        )}
        <button
          type="submit"
          className="announcement-form__button announcement-form__button--primary"
          disabled={submitting || uploading || !title.trim() || !content.trim()}
          aria-busy={submitting}
        >
          {submitting ? (
            <>
              <div className="announcement-form__spinner" aria-hidden="true" />
              <span>Posting...</span>
            </>
          ) : (
            <>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
                <path d="M13.73 21a2 2 0 0 1-3.46 0" />
              </svg>
              <span>Post Announcement</span>
            </>
          )}
        </button>
      </div>
    </form>
  );
};

export default AnnouncementForm;
