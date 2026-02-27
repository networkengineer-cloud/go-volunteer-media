import React, { useState, useEffect } from 'react';
import { announcementsApi, groupsApi, type Group } from '../api/client';
import { useToast } from '../hooks/useToast';
import './AnnouncementForm.css';

interface AnnouncementFormProps {
  groupId: number;
  onSuccess?: () => void;
  onCancel?: () => void;
}

const AnnouncementForm: React.FC<AnnouncementFormProps> = ({ groupId, onSuccess, onCancel }) => {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [sendGroupMe, setSendGroupMe] = useState(false);
  const [group, setGroup] = useState<Group | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const toast = useToast();

  useEffect(() => {
    const loadGroup = async () => {
      try {
        const response = await groupsApi.getById(groupId);
        setGroup(response.data);
      } catch (error) {
        console.error('Failed to load group:', error);
      }
    };
    loadGroup();
  }, [groupId]);

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
      // Always send email to opted-in group members
      await announcementsApi.createGroupAnnouncement(groupId, title.trim(), content.trim(), true, sendGroupMe);
      toast.showSuccess('Announcement posted successfully!');
      
      // Reset form
      setTitle('');
      setContent('');
      setSendGroupMe(false);
      
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

        {/* GroupMe integration */}
        {group && group.groupme_enabled && group.groupme_bot_id && (
          <div className="announcement-form__field">
            <label className="announcement-form__checkbox-label">
              <input
                type="checkbox"
                checked={sendGroupMe}
                onChange={(e) => setSendGroupMe(e.target.checked)}
                disabled={submitting}
              />
              <span className="announcement-form__checkbox-text">
                Also send to GroupMe
                <span className="announcement-form__checkbox-description">
                  This announcement will be posted to the {group.name} GroupMe chat
                </span>
              </span>
            </label>
          </div>
        )}
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
          disabled={submitting || !title.trim() || !content.trim()}
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
