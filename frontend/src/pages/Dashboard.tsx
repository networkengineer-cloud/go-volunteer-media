import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { groupsApi, authApi } from '../api/client';
import type { Group } from '../api/client';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import './Dashboard.css';

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [user]);

  const loadData = async () => {
    try {
      // Load groups
      const groupsResponse = await groupsApi.getAll();
      
      setGroups(groupsResponse.data);

      // Determine which group to show and redirect to it
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
          // Redirect to the group page instead of showing dashboard
          navigate(`/groups/${groupToShow.id}`, { replace: true });
        }
      }
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
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

  // If we have groups, the redirect will happen and this won't render
  return null;
};

export default Dashboard;
