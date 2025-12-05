import React, { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { userProfileApi } from '../api/client';
import type { UserProfile } from '../api/client';
import { useToast } from '../hooks/useToast';
import './UserProfilePage.css';

const UserProfilePage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);

  const loadProfile = useCallback(async () => {
    if (!id) {
      showToast('Invalid user ID', 'error');
      navigate('/admin/users');
      return;
    }

    try {
      setLoading(true);
      const response = await userProfileApi.getProfile(parseInt(id));
      setProfile(response.data);
    } catch (error: unknown) {
      console.error('Failed to load user profile:', error);
      if (error.response?.status === 403) {
        showToast('You do not have permission to view this profile', 'error');
      } else if (error.response?.status === 404) {
        showToast('User not found', 'error');
      } else {
        showToast('Failed to load user profile', 'error');
      }
      navigate('/admin/users');
    } finally {
      setLoading(false);
    }
  }, [id, showToast, navigate]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  const formatRelativeTime = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 30) return `${diffDays}d ago`;
    return formatDate(dateString);
  };

  if (loading) {
    return (
      <div className="user-profile-page">
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading profile...</p>
        </div>
      </div>
    );
  }

  if (!profile) {
    return null;
  }

  // Check if we have a full profile (with statistics) or limited profile
  const hasFullProfile = !!profile.statistics;
  const hasExtendedProfile = !!profile.email; // Group admin view includes email

  return (
    <div className="user-profile-page">
      {/* Header */}
      <div className="profile-header">
        <div className="container">
          <button onClick={() => navigate(-1)} className="back-button" aria-label="Go back">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12.5 15L7.5 10L12.5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            Back
          </button>

          <div className="profile-info">
            <div className="profile-avatar">
              <svg width="60" height="60" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <circle cx="12" cy="8" r="4" fill="currentColor" opacity="0.3"/>
                <path d="M4 20C4 16.6863 6.68629 14 10 14H14C17.3137 14 20 16.6863 20 20" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
              </svg>
            </div>
            <div className="profile-details">
              <h1>{profile.username}</h1>
              {profile.email && <p className="profile-email">{profile.email}</p>}
              {profile.phone_number && <p className="profile-phone">{profile.phone_number}</p>}
              {profile.is_admin && <span className="admin-badge">Admin</span>}
              {profile.created_at && <p className="member-since">Member since {formatDate(profile.created_at)}</p>}
            </div>
          </div>
        </div>
      </div>

      <div className="container">
        {/* Limited Profile Notice */}
        {!hasFullProfile && !hasExtendedProfile && (
          <div className="limited-profile-notice">
            <p>You are viewing a limited profile. Only basic information is available.</p>
          </div>
        )}

        {/* Statistics Overview - only for full profiles */}
        {hasFullProfile && profile.statistics && (
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-value">{profile.statistics.total_comments}</div>
              <div className="stat-label">Comments</div>
            </div>
          </div>

          {profile.is_admin && (
            <div className="stat-card">
              <div className="stat-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  <path d="M13.73 21a2 2 0 0 1-3.46 0" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <div className="stat-content">
                <div className="stat-value">{profile.statistics.total_announcements}</div>
                <div className="stat-label">Announcements</div>
              </div>
            </div>
          )}

          <div className="stat-card">
            <div className="stat-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M21.5 8.5v5c0 1.66-1.34 3-3 3h-1.5v2.48c0 1.13-1.22 1.84-2.2 1.28l-3.47-1.97c-.39-.22-.82-.33-1.26-.33h-2.1c-1.66 0-3-1.34-3-3v-5c0-1.66 1.34-3 3-3h10.03c1.66 0 3 1.34 3 3z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-value">{profile.statistics.animals_interacted}</div>
              <div className="stat-label">Animals</div>
            </div>
          </div>

          {profile.statistics.last_active_date && (
            <div className="stat-card">
              <div className="stat-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2"/>
                  <path d="M12 6v6l4 2" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
                </svg>
              </div>
              <div className="stat-content">
                <div className="stat-value">{formatRelativeTime(profile.statistics.last_active_date)}</div>
                <div className="stat-label">Last Active</div>
              </div>
            </div>
          )}
        </div>
        )}

        {/* Groups */}
        {profile.groups && profile.groups.length > 0 && (
          <div className="profile-section">
            <h2>Groups ({profile.groups.length})</h2>
            <div className="groups-list">
              {profile.groups.map((group) => (
                <Link
                  key={group.id}
                  to={`/groups/${group.id}`}
                  className="group-card"
                  title={`View ${group.name} group`}
                >
                  {group.image_url && (
                    <img src={group.image_url} alt={group.name} className="group-image" />
                  )}
                  <div className="group-info">
                    <h3>{group.name}</h3>
                    {group.description && <p>{group.description}</p>}
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* Most Active Group */}
        {profile.statistics?.most_active_group && (
          <div className="profile-section">
            <h2>Most Active Group</h2>
            <div className="most-active-group">
              <Link
                to={`/groups/${profile.statistics.most_active_group.group_id}`}
                className="group-link"
              >
                <strong>{profile.statistics.most_active_group.group_name}</strong>
              </Link>
              <span className="comment-count">
                {profile.statistics.most_active_group.comment_count} comments
              </span>
            </div>
          </div>
        )}

        {/* Animals Interacted With */}
        {profile.animals_interacted_with && profile.animals_interacted_with.length > 0 && (
          <div className="profile-section">
            <h2>Animals Interacted With ({profile.animals_interacted_with.length})</h2>
            <div className="animals-grid">
              {profile.animals_interacted_with.map((animal) => (
                <Link
                  key={animal.animal_id}
                  to={`/groups/${animal.group_id}/animals/${animal.animal_id}/view`}
                  className="animal-card"
                  title={`View ${animal.animal_name}`}
                >
                  {animal.image_url ? (
                    <img src={animal.image_url} alt={animal.animal_name} className="animal-image" />
                  ) : (
                    <div className="animal-image-placeholder">
                      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M21.5 8.5v5c0 1.66-1.34 3-3 3h-1.5v2.48c0 1.13-1.22 1.84-2.2 1.28l-3.47-1.97c-.39-.22-.82-.33-1.26-.33h-2.1c-1.66 0-3-1.34-3-3v-5c0-1.66 1.34-3 3-3h10.03c1.66 0 3 1.34 3 3z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                      </svg>
                    </div>
                  )}
                  <div className="animal-info">
                    <h3>{animal.animal_name}</h3>
                    <p className="group-name">{animal.group_name}</p>
                    <div className="animal-stats">
                      <span>{animal.comment_count} comment{animal.comment_count !== 1 ? 's' : ''}</span>
                      <span>·</span>
                      <span>{formatRelativeTime(animal.last_comment_at)}</span>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* Recent Comments */}
        {profile.recent_comments && profile.recent_comments.length > 0 && (
          <div className="profile-section">
            <h2>Recent Comments ({profile.recent_comments.length})</h2>
            <div className="activity-list">
              {profile.recent_comments.map((comment) => (
                <div key={comment.id} className="activity-item">
                  <div className="activity-meta">
                    <Link to={`/groups/${comment.group_id}/animals/${comment.animal_id}/view`} className="animal-link">
                      {comment.animal_name}
                    </Link>
                    <span className="activity-group">in {comment.group_name}</span>
                    <span className="activity-time">{formatRelativeTime(comment.created_at)}</span>
                  </div>
                  <div className="activity-content">
                    <p>{comment.content}</p>
                    {comment.image_url && (
                      <img src={comment.image_url} alt="Comment attachment" className="activity-image" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Recent Announcements (if admin) */}
        {profile.is_admin && profile.recent_announcements && profile.recent_announcements.length > 0 && (
          <div className="profile-section">
            <h2>Recent Announcements ({profile.recent_announcements.length})</h2>
            <div className="activity-list">
              {profile.recent_announcements.map((announcement) => (
                <div key={announcement.id} className="activity-item">
                  <div className="activity-meta">
                    <span className="activity-type">Site-wide announcement</span>
                    <span className="activity-time">{formatRelativeTime(announcement.created_at)}</span>
                  </div>
                  <div className="activity-content">
                    <p>{announcement.content}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Empty State - only for full profiles with no activity */}
        {hasFullProfile && (!profile.recent_comments || profile.recent_comments.length === 0) && (!profile.animals_interacted_with || profile.animals_interacted_with.length === 0) && (
          <div className="empty-state">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="8" r="4" stroke="currentColor" strokeWidth="2"/>
              <path d="M4 20C4 16.6863 6.68629 14 10 14H14C17.3137 14 20 16.6863 20 20" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
            <h3>No Activity Yet</h3>
            <p>This user hasn't posted any comments or interacted with animals yet.</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default UserProfilePage;
