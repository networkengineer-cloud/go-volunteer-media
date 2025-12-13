import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { adminDashboardApi } from '../api/client';
import type { AdminDashboardStats } from '../api/client';
import { useToast } from '../hooks/useToast';
import './AdminDashboard.css';

const AdminDashboard: React.FC = () => {
  const { showToast } = useToast();
  const [stats, setStats] = useState<AdminDashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  const loadStats = useCallback(async () => {
    try {
      setLoading(true);
      const response = await adminDashboardApi.getStats();
      setStats(response.data);
    } catch (error: unknown) {
      console.error('Failed to load dashboard stats:', error);
      showToast('Failed to load dashboard statistics', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
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
    if (diffDays < 7) return `${diffDays}d ago`;
    return formatDate(dateString);
  };

  if (loading) {
    return (
      <div className="admin-dashboard">
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (!stats) {
    return <div className="admin-dashboard"><p>Failed to load dashboard</p></div>;
  }

  return (
    <div className="admin-dashboard">
      <div className="dashboard-header">
        <h1>Admin Dashboard</h1>
        <p className="dashboard-subtitle">System overview and key metrics</p>
      </div>

      {/* Quick Stats Overview */}
      <div className="quick-stats-grid">
        <div className="stat-card">
          <div className="stat-icon users">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <circle cx="9" cy="7" r="4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </div>
          <div className="stat-content">
            <div className="stat-value">{stats.total_users}</div>
            <div className="stat-label">Total Users</div>
            <Link to="/admin/users" className="stat-link">Manage →</Link>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon groups">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="3" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="14" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="14" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="3" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
            </svg>
          </div>
          <div className="stat-content">
            <div className="stat-value">{stats.total_groups}</div>
            <div className="stat-label">Total Groups</div>
            <Link to="/admin/groups" className="stat-link">Manage →</Link>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon animals">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2"/>
              <circle cx="9" cy="10" r="1.5" fill="currentColor"/>
              <circle cx="15" cy="10" r="1.5" fill="currentColor"/>
              <path d="M9 15c.5 1 1.5 2 3 2s2.5-1 3-2" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
          </div>
          <div className="stat-content">
            <div className="stat-value">{stats.total_animals}</div>
            <div className="stat-label">Total Animals</div>
            <Link to="/admin/animals" className="stat-link">Manage →</Link>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon comments">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </div>
          <div className="stat-content">
            <div className="stat-value">{stats.total_comments}</div>
            <div className="stat-label">Total Comments</div>
          </div>
        </div>
      </div>

      {/* System Health */}
      <div className="dashboard-section">
        <h2>System Health (Last 24 Hours)</h2>
        <div className="health-grid">
          <div className="health-card">
            <div className="health-value">{stats.system_health.active_users_last_24h}</div>
            <div className="health-label">Active Users</div>
          </div>
          <div className="health-card">
            <div className="health-value">{stats.system_health.comments_last_24h}</div>
            <div className="health-label">New Comments</div>
          </div>
          <div className="health-card">
            <div className="health-value">{stats.system_health.new_users_last_7_days}</div>
            <div className="health-label">New Users (7d)</div>
          </div>
          <div className="health-card">
            <div className="health-value">{Math.round(stats.system_health.average_comments_per_day)}</div>
            <div className="health-label">Avg Comments/Day</div>
          </div>
        </div>
      </div>

      <div className="dashboard-grid">
        {/* Recent Users */}
        <div className="dashboard-section">
          <h2>Recent Users</h2>
          {stats.recent_users.length > 0 ? (
            <div className="user-list">
              {stats.recent_users.map((user) => (
                <Link
                  key={user.id}
                  to={`/users/${user.id}/profile`}
                  className="user-item"
                >
                  <div className="user-avatar">
                    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <circle cx="12" cy="8" r="4" fill="currentColor" opacity="0.3"/>
                      <path d="M4 20C4 16.6863 6.68629 14 10 14H14C17.3137 14 20 16.6863 20 20" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
                    </svg>
                  </div>
                  <div className="user-info">
                    <div className="user-name">
                      {user.username}
                      {user.is_admin && <span className="admin-badge">Admin</span>}
                    </div>
                    <div className="user-meta">{formatRelativeTime(user.created_at)}</div>
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <p className="empty-message">No recent users</p>
          )}
        </div>

        {/* Most Active Groups */}
        <div className="dashboard-section">
          <h2>Most Active Groups</h2>
          {stats.most_active_groups.length > 0 ? (
            <div className="group-list">
              {stats.most_active_groups.map((group) => (
                <Link
                  key={group.group_id}
                  to={`/groups/${group.group_id}`}
                  className="group-item"
                >
                  <div className="group-info">
                    <div className="group-name">{group.group_name}</div>
                    <div className="group-stats">
                      <span>{group.user_count} members</span>
                      <span>•</span>
                      <span>{group.animal_count} animals</span>
                      <span>•</span>
                      <span>{group.comment_count} comments</span>
                    </div>
                  </div>
                  {group.last_activity && (
                    <div className="group-activity">
                      {formatRelativeTime(group.last_activity)}
                    </div>
                  )}
                </Link>
              ))}
            </div>
          ) : (
            <p className="empty-message">No active groups in the last 30 days</p>
          )}
        </div>
      </div>

      {/* Animals Needing Attention */}
      {stats.animals_needing_attention.length > 0 && (
        <div className="dashboard-section">
          <h2>Animals Needing Attention</h2>
          <div className="animals-grid">
            {stats.animals_needing_attention.map((animal) => (
              <Link
                key={animal.animal_id}
                to={`/groups/${animal.group_id}/animals/${animal.animal_id}/view`}
                className="animal-alert-card"
              >
                {animal.image_url ? (
                  <img src={animal.image_url} alt={animal.animal_name} className="animal-alert-image" />
                ) : (
                  <div className="animal-alert-placeholder">
                    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2"/>
                      <circle cx="9" cy="10" r="1.5" fill="currentColor"/>
                      <circle cx="15" cy="10" r="1.5" fill="currentColor"/>
                      <path d="M9 15c.5 1 1.5 2 3 2s2.5-1 3-2" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
                    </svg>
                  </div>
                )}
                <div className="animal-alert-info">
                  <h3>{animal.animal_name}</h3>
                  <p className="animal-alert-group">{animal.group_name}</p>
                  {animal.alert_tags.length > 0 && (
                    <div className="alert-tags">
                      {animal.alert_tags.map((tag, index) => (
                        <span key={index} className="alert-tag">{tag}</span>
                      ))}
                    </div>
                  )}
                  {animal.last_comment && (
                    <p className="animal-alert-time">{formatRelativeTime(animal.last_comment)}</p>
                  )}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Quick Links */}
      <div className="dashboard-section">
        <h2>Quick Links</h2>
        <div className="quick-links">
          <Link to="/admin/users" className="quick-link-card">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <circle cx="9" cy="7" r="4" stroke="currentColor" strokeWidth="2"/>
            </svg>
            <span>Manage Users</span>
          </Link>
          <Link to="/admin/groups" className="quick-link-card">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="3" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="14" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="14" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
              <rect x="3" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="2"/>
            </svg>
            <span>Manage Groups</span>
          </Link>
          <Link to="/admin/animals" className="quick-link-card">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2"/>
            </svg>
            <span>Bulk Edit Animals</span>
          </Link>
          <Link to="/admin/animal-tags" className="quick-link-card">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              <line x1="7" y1="7" x2="7.01" y2="7" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
            <span>Animal Tags</span>
          </Link>
          <Link to="/admin/site-settings" className="quick-link-card">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="2"/>
              <path d="M12 1v6m0 6v6M23 12h-6m-6 0H1" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
            <span>Site Settings</span>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default AdminDashboard;
