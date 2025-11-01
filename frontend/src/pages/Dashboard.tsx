import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { groupsApi, announcementsApi, authApi, animalsApi } from '../api/client';
import type { Group, Announcement, CommentWithAnimal, Animal } from '../api/client';
import './Dashboard.css';

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [groups, setGroups] = useState<Group[]>([]);
  const [currentGroup, setCurrentGroup] = useState<Group | null>(null);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [latestComments, setLatestComments] = useState<CommentWithAnimal[]>([]);
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAnnouncementForm, setShowAnnouncementForm] = useState(false);
  const [announcementTitle, setAnnouncementTitle] = useState('');
  const [announcementContent, setAnnouncementContent] = useState('');
  const [sendEmail, setSendEmail] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadData();
  }, [user]);

  const loadData = async () => {
    try {
      // Load groups and announcements
      const [groupsResponse, announcementsResponse] = await Promise.all([
        groupsApi.getAll(),
        announcementsApi.getAll(),
      ]);
      
      setGroups(groupsResponse.data);
      setAnnouncements(announcementsResponse.data);

      // Determine which group to show
      let groupToShow: Group | null = null;
      
      if (groupsResponse.data.length > 0) {
        // Check if user has a default group set
        if (user?.default_group_id) {
          groupToShow = groupsResponse.data.find(g => g.id === user.default_group_id) || groupsResponse.data[0];
        } else {
          // If no default group, use the first group
          groupToShow = groupsResponse.data[0];
          // Set it as default for future logins
          try {
            await authApi.setDefaultGroup(groupToShow.id);
          } catch (error) {
            console.error('Failed to set default group:', error);
          }
        }

        if (groupToShow) {
          await loadGroupData(groupToShow);
        }
      }
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadGroupData = async (group: Group) => {
    try {
      setCurrentGroup(group);
      
      // Load latest comments and animals for this group
      const [commentsResponse, animalsResponse] = await Promise.all([
        groupsApi.getLatestComments(group.id, 20),
        animalsApi.getAll(group.id, 'all'),
      ]);
      
      setLatestComments(commentsResponse.data);
      setAnimals(animalsResponse.data);
    } catch (error) {
      console.error('Failed to load group data:', error);
    }
  };

  const handleGroupSwitch = async (group: Group) => {
    setLoading(true);
    try {
      await loadGroupData(group);
      // Update default group
      await authApi.setDefaultGroup(group.id);
    } catch (error) {
      console.error('Failed to switch group:', error);
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

  // If no groups, show message
  if (groups.length === 0) {
    return (
      <div className="dashboard">
        <h1>Welcome, {user?.username}!</h1>
        <p>You haven't been assigned to any groups yet. Contact an admin to get started!</p>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <h1>Welcome, {user?.username}!</h1>
        
        {/* Group Switcher */}
        {groups.length > 1 && (
          <div className="group-switcher">
            <label htmlFor="group-select">Group:</label>
            <select
              id="group-select"
              value={currentGroup?.id || ''}
              onChange={(e) => {
                const group = groups.find(g => g.id === Number(e.target.value));
                if (group) handleGroupSwitch(group);
              }}
              className="group-select"
            >
              {groups.map(group => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {currentGroup && (
        <>
          {/* Current Group Header */}
          <div className="current-group-header">
            <h2>{currentGroup.name}</h2>
            <p>{currentGroup.description}</p>
            <div className="group-actions">
              <Link to={`/groups/${currentGroup.id}`} className="btn-secondary">
                View Full Group Page
              </Link>
            </div>
          </div>

          {/* Latest Comments Section */}
          <div className="latest-comments-section">
            <h3>Latest Comments</h3>
            {latestComments.length === 0 ? (
              <p className="no-comments">No comments yet in this group.</p>
            ) : (
              <div className="comments-list">
                {latestComments.map((comment) => (
                  <div key={comment.id} className="comment-card">
                    <div className="comment-animal-info">
                      <Link 
                        to={`/groups/${currentGroup.id}/animals/${comment.animal_id}/view`}
                        className="animal-link"
                      >
                        {comment.animal?.image_url && (
                          <img 
                            src={comment.animal.image_url} 
                            alt={comment.animal.name}
                            className="comment-animal-thumbnail"
                          />
                        )}
                        <strong>{comment.animal?.name || 'Unknown Animal'}</strong>
                      </Link>
                    </div>
                    <div className="comment-content">
                      <p>{comment.content}</p>
                      {comment.image_url && (
                        <img 
                          src={comment.image_url} 
                          alt="Comment attachment" 
                          className="comment-image"
                        />
                      )}
                      {comment.tags && comment.tags.length > 0 && (
                        <div className="comment-tags">
                          {comment.tags.map(tag => (
                            <span 
                              key={tag.id} 
                              className="comment-tag"
                              style={{ backgroundColor: tag.color }}
                            >
                              {tag.name}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="comment-meta">
                      <span className="comment-author">
                        {comment.user?.username || 'Unknown'}
                      </span>
                      <span className="comment-date">
                        {formatDate(comment.created_at)}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Animals List Section */}
          <div className="animals-section">
            <div className="section-header">
              <h3>Animals in {currentGroup.name}</h3>
              <Link to={`/groups/${currentGroup.id}`} className="btn-link">
                View All →
              </Link>
            </div>
            {animals.length === 0 ? (
              <p>No animals in this group yet.</p>
            ) : (
              <div className="animals-grid">
                {animals.slice(0, 6).map((animal) => (
                  <Link 
                    key={animal.id} 
                    to={`/groups/${currentGroup.id}/animals/${animal.id}/view`}
                    className="animal-card-mini"
                  >
                    {animal.image_url && (
                      <img src={animal.image_url} alt={animal.name} />
                    )}
                    <div className="animal-info">
                      <h4>{animal.name}</h4>
                      <span className={`status-badge status-${animal.status}`}>
                        {animal.status}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </>
      )}

      {/* Announcements Section */}
      <div className="announcements-section">
        <div className="announcements-header">
          <h3>Announcements</h3>
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
    </div>
  );
};

export default Dashboard;
