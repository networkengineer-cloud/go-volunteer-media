import React, { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { groupsApi, announcementsApi, authApi, animalsApi } from '../api/client';
import type { Group, Announcement, CommentWithAnimal, Animal } from '../api/client';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import './Dashboard.css';

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
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
    return (
      <div className="dashboard">
        <div className="dashboard-header">
          <SkeletonLoader variant="text" width="40%" height="2rem" />
        </div>
        <div className="skeleton-list">
          <SkeletonLoader variant="comment" count={5} />
        </div>
      </div>
    );
  }

  // If no groups, show empty state
  if (groups.length === 0) {
    const welcomeIcon = (
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
        <path d="M16 3.13a4 4 0 0 1 0 7.75" />
      </svg>
    );

    return (
      <div className="dashboard">
        <EmptyState
          icon={welcomeIcon}
          title={`Welcome to HAWS, ${user?.username}!`}
          description="You haven't been assigned to any volunteer groups yet. Contact an admin to get access to dog, cat, or other volunteer groups where you can share updates and connect with animals."
          primaryAction={user?.is_admin ? {
            label: 'Create a Group',
            onClick: () => navigate('/admin/groups')
          } : undefined}
          secondaryAction={{
            label: 'Learn More',
            onClick: () => window.open('https://hawspets.org/volunteer', '_blank')
          }}
        />
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
              <EmptyState
                icon={
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                  </svg>
                }
                title="No comments yet"
                description={`Be the first to share an update in the ${currentGroup.name} group! Comment on an animal to share your experiences, photos, or care notes.`}
                primaryAction={{
                  label: 'View Animals',
                  onClick: () => navigate(`/groups/${currentGroup.id}`)
                }}
              />
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
              <EmptyState
                icon={
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <circle cx="12" cy="8" r="4" />
                    <path d="M6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2" />
                    <path d="M12 12c1.657 0 3-1.343 3-3s-1.343-3-3-3-3 1.343-3 3 1.343 3 3 3z" />
                  </svg>
                }
                title={`No animals in ${currentGroup.name} yet`}
                description={user?.is_admin ? 
                  "Get started by adding your first animal to this group. Animals added here will be visible to all group members." :
                  "This group doesn't have any animals yet. An admin will need to add animals before volunteers can share updates."
                }
                primaryAction={user?.is_admin ? {
                  label: 'Add First Animal',
                  onClick: () => navigate(`/groups/${currentGroup.id}/animals/new`)
                } : undefined}
              />
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
          <EmptyState
            icon={
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
            }
            title="No announcements yet"
            description={user?.is_admin ?
              "Create the first announcement to keep volunteers informed about important updates, events, or reminders." :
              "No announcements from the shelter at this time. Check back here for important updates from the admin team."
            }
            primaryAction={user?.is_admin ? {
              label: 'Create Announcement',
              onClick: () => setShowAnnouncementForm(true)
            } : undefined}
          />
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
