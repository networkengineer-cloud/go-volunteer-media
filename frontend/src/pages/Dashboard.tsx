import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { groupsApi, announcementsApi } from '../api/client';
import type { Group, Announcement } from '../api/client';
import './Dashboard.css';

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [groups, setGroups] = useState<Group[]>([]);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAnnouncementForm, setShowAnnouncementForm] = useState(false);
  const [announcementTitle, setAnnouncementTitle] = useState('');
  const [announcementContent, setAnnouncementContent] = useState('');
  const [sendEmail, setSendEmail] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [groupsResponse, announcementsResponse] = await Promise.all([
        groupsApi.getAll(),
        announcementsApi.getAll(),
      ]);
      setGroups(groupsResponse.data);
      setAnnouncements(announcementsResponse.data);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAnnouncement = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!announcementTitle.trim() || !announcementContent.trim()) return;

    setSubmitting(true);
    try {
      await announcementsApi.create(announcementTitle, announcementContent, sendEmail);
      setAnnouncementTitle('');
      setAnnouncementContent('');
      setSendEmail(false);
      setShowAnnouncementForm(false);
      await loadData();
    } catch (error) {
      console.error('Failed to create announcement:', error);
      alert('Failed to create announcement');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteAnnouncement = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this announcement?')) return;

    try {
      await announcementsApi.delete(id);
      await loadData();
    } catch (error) {
      console.error('Failed to delete announcement:', error);
      alert('Failed to delete announcement');
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  return (
    <div className="dashboard">
      <h1>Welcome, {user?.username}!</h1>

      {/* Announcements Section */}
      <div className="announcements-section">
        <div className="announcements-header">
          <h2>Announcements</h2>
          {user?.is_admin && (
            <button
              onClick={() => setShowAnnouncementForm(!showAnnouncementForm)}
              className="btn-primary"
            >
              {showAnnouncementForm ? 'Cancel' : 'New Announcement'}
            </button>
          )}
        </div>

        {showAnnouncementForm && (
          <form onSubmit={handleCreateAnnouncement} className="announcement-form">
            <div className="form-group">
              <label htmlFor="title">Title</label>
              <input
                type="text"
                id="title"
                value={announcementTitle}
                onChange={(e) => setAnnouncementTitle(e.target.value)}
                placeholder="Announcement title"
                required
                maxLength={200}
              />
            </div>
            <div className="form-group">
              <label htmlFor="content">Content</label>
              <textarea
                id="content"
                value={announcementContent}
                onChange={(e) => setAnnouncementContent(e.target.value)}
                placeholder="Announcement content"
                required
                rows={4}
              />
            </div>
            <div className="form-group checkbox-group">
              <label>
                <input
                  type="checkbox"
                  checked={sendEmail}
                  onChange={(e) => setSendEmail(e.target.checked)}
                />
                Send email to users who opted in
              </label>
            </div>
            <button type="submit" disabled={submitting} className="btn-primary">
              {submitting ? 'Creating...' : 'Create Announcement'}
            </button>
          </form>
        )}

        {announcements.length === 0 ? (
          <p className="no-announcements">No announcements yet.</p>
        ) : (
          <div className="announcements-list">
            {announcements.map((announcement) => (
              <div key={announcement.id} className="announcement-card">
                <div className="announcement-header-row">
                  <h3>{announcement.title}</h3>
                  {user?.is_admin && (
                    <button
                      onClick={() => handleDeleteAnnouncement(announcement.id)}
                      className="btn-delete"
                      title="Delete announcement"
                    >
                      ×
                    </button>
                  )}
                </div>
                <p className="announcement-content">{announcement.content}</p>
                <div className="announcement-meta">
                  <span className="announcement-author">
                    By {announcement.user?.username || 'Admin'}
                  </span>
                  <span className="announcement-date">
                    {formatDate(announcement.created_at)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Groups Section */}
      <h2>Your Groups</h2>
      {groups.length === 0 ? (
        <p>You haven't been assigned to any groups yet. Contact an admin to get started!</p>
      ) : (
        <div className="groups-grid">
          {groups.map((group) => (
            <Link key={group.id} to={`/groups/${group.id}`} className="group-card">
              <h3>{group.name}</h3>
              <p>{group.description}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export default Dashboard;
