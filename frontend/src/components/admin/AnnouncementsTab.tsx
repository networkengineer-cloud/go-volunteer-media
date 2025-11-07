import React, { useState, useEffect } from 'react';
import { announcementsApi, groupsApi, type Announcement, type Group } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import './AnnouncementsTab.css';

const AnnouncementsTab: React.FC = () => {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [sendEmail, setSendEmail] = useState(false);
  const [sendGroupMe, setSendGroupMe] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [announcementsResponse, groupsResponse] = await Promise.all([
        announcementsApi.getAll(),
        groupsApi.getAll(),
      ]);
      setAnnouncements(announcementsResponse.data);
      setGroups(groupsResponse.data);
    } catch (error) {
      console.error('Failed to load data:', error);
      toast.showError('Failed to load announcements and groups');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim() || title.trim().length < 2) {
      toast.showWarning('Title must be at least 2 characters long');
      return;
    }

    if (!content.trim() || content.trim().length < 10) {
      toast.showWarning('Content must be at least 10 characters long');
      return;
    }

    if (!sendEmail && !sendGroupMe) {
      toast.showWarning('Please select at least one delivery method (Email or GroupMe)');
      return;
    }

    setSubmitting(true);
    try {
      await announcementsApi.create(title.trim(), content.trim(), sendEmail, sendGroupMe);
      toast.showSuccess('Announcement created successfully!');
      
      // Reset form
      setTitle('');
      setContent('');
      setSendEmail(false);
      setSendGroupMe(false);
      
      // Reload announcements
      loadData();
    } catch (error) {
      console.error('Failed to create announcement:', error);
      const err = error as { response?: { data?: { error?: string } } };
      const errorMsg = err.response?.data?.error || 'Failed to create announcement. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this announcement?')) {
      return;
    }

    try {
      await announcementsApi.delete(id);
      toast.showSuccess('Announcement deleted successfully');
      loadData();
    } catch (error) {
      console.error('Failed to delete announcement:', error);
      toast.showError('Failed to delete announcement');
    }
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    });
  };

  const groupMeEnabledGroups = groups.filter(g => g.groupme_enabled && g.groupme_bot_id);

  if (loading) {
    return (
      <div className="announcements-tab">
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading announcements...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="announcements-tab">
      <h2>Create Site-Wide Announcement</h2>
      <p className="tab-description">
        Send important updates to all volunteers via email and/or GroupMe
      </p>

      <form className="announcement-form" onSubmit={handleSubmit}>
        <div className="form-field">
          <label htmlFor="title">
            Title <span className="required">*</span>
          </label>
          <input
            id="title"
            type="text"
            className="form-input"
            placeholder="e.g., Emergency Meeting This Saturday"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
            required
            disabled={submitting}
          />
          <span className="char-count">{title.length}/200 characters</span>
        </div>

        <div className="form-field">
          <label htmlFor="content">
            Content <span className="required">*</span>
          </label>
          <textarea
            id="content"
            className="form-textarea"
            placeholder="Write your announcement message..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={6}
            required
            disabled={submitting}
          />
          <span className="char-count">{content.length} characters (min 10)</span>
        </div>

        <div className="delivery-options">
          <h3>Delivery Methods</h3>
          <p className="delivery-description">Select how to send this announcement</p>
          
          <div className="checkbox-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={sendEmail}
                onChange={(e) => setSendEmail(e.target.checked)}
                disabled={submitting}
              />
              <span className="checkbox-text">
                Send via Email
                <span className="checkbox-description">
                  Send to all users who have enabled email notifications
                </span>
              </span>
            </label>

            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={sendGroupMe}
                onChange={(e) => setSendGroupMe(e.target.checked)}
                disabled={submitting}
              />
              <span className="checkbox-text">
                Send via GroupMe
                <span className="checkbox-description">
                  {groupMeEnabledGroups.length > 0
                    ? `Send to ${groupMeEnabledGroups.length} GroupMe-enabled group${groupMeEnabledGroups.length > 1 ? 's' : ''}`
                    : 'No groups have GroupMe enabled yet'}
                </span>
              </span>
            </label>
          </div>

          {groupMeEnabledGroups.length > 0 && sendGroupMe && (
            <div className="groupme-info">
              <strong>Will be sent to:</strong>
              <ul>
                {groupMeEnabledGroups.map(group => (
                  <li key={group.id}>{group.name}</li>
                ))}
              </ul>
            </div>
          )}

          {groupMeEnabledGroups.length === 0 && (
            <div className="groupme-notice">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
              <span>
                To enable GroupMe integration, configure a GroupMe Bot ID for each group in the Groups admin page.
              </span>
            </div>
          )}
        </div>

        <div className="form-actions">
          <button
            type="submit"
            className="btn btn-primary"
            disabled={submitting || (!sendEmail && !sendGroupMe)}
          >
            {submitting ? (
              <>
                <div className="spinner-small" />
                <span>Sending...</span>
              </>
            ) : (
              <>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
                  <path d="M13.73 21a2 2 0 0 1-3.46 0" />
                </svg>
                <span>Send Announcement</span>
              </>
            )}
          </button>
        </div>
      </form>

      <div className="announcements-list">
        <h2>Recent Announcements</h2>
        {announcements.length === 0 ? (
          <p className="empty-message">No announcements yet</p>
        ) : (
          <div className="announcements-grid">
            {announcements.map((announcement) => (
              <div key={announcement.id} className="announcement-card">
                <div className="announcement-header">
                  <h3>{announcement.title}</h3>
                  <button
                    className="btn-delete"
                    onClick={() => handleDelete(announcement.id)}
                    aria-label="Delete announcement"
                  >
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                    </svg>
                  </button>
                </div>
                <p className="announcement-content">{announcement.content}</p>
                <div className="announcement-meta">
                  <span className="announcement-date">{formatDate(announcement.created_at)}</span>
                  <div className="announcement-delivery">
                    {announcement.send_email && (
                      <span className="delivery-badge email">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
                          <polyline points="22,6 12,13 2,6" />
                        </svg>
                        Email
                      </span>
                    )}
                    {announcement.send_groupme && (
                      <span className="delivery-badge groupme">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                        </svg>
                        GroupMe
                      </span>
                    )}
                  </div>
                </div>
                {announcement.user && (
                  <span className="announcement-author">by {announcement.user.username}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default AnnouncementsTab;
