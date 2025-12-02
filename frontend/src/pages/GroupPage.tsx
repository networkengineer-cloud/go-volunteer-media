import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom';
import { groupsApi, animalsApi, authApi } from '../api/client';
import type { Group, Animal, GroupMembership } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import ActivityFeed from '../components/ActivityFeed';
import AnnouncementForm from '../components/AnnouncementForm';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import Modal from '../components/Modal';
import ProtocolsList from '../components/ProtocolsList';
import './GroupPage.css';

type ViewMode = 'activity' | 'animals' | 'protocols';
type FilterType = 'all' | 'comments' | 'announcements';

const GroupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  useAuth(); // Ensure user is authenticated
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  
  // Get initial view mode from URL parameter (default to 'activity')
  const initialViewMode = (searchParams.get('view') as ViewMode) || 'activity';
  
  const [group, setGroup] = useState<Group | null>(null);
  const [groups, setGroups] = useState<Group[]>([]);
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [membership, setMembership] = useState<GroupMembership | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [viewMode, setViewMode] = useState<ViewMode>(initialViewMode);
  const [filterType, setFilterType] = useState<FilterType>('all');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [nameSearch, setNameSearch] = useState<string>('');
  const [showAnnouncementForm, setShowAnnouncementForm] = useState(false);
  const [activityFeedKey, setActivityFeedKey] = useState(0);

  // Load group data and membership info
  const loadGroupData = async (groupId: number) => {
    try {
      setError('');
      const [groupRes, membershipRes] = await Promise.all([
        groupsApi.getById(groupId),
        groupsApi.getMembership(groupId)
      ]);
      setGroup(groupRes.data);
      setMembership(membershipRes.data);
    } catch (error) {
      console.error('Failed to load group data:', error);
      const err = error as { response?: { data?: { error?: string } } };
      setError(err.response?.data?.error || 'Failed to load group information. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  // Load all groups for the switcher
  const loadAllGroups = async () => {
    try {
      const groupsRes = await groupsApi.getAll();
      setGroups(groupsRes.data);
    } catch (error) {
      console.error('Failed to load groups:', error);
    }
  };

  // Load animals with filters
  const loadAnimals = useCallback(async (groupId: number) => {
    try {
      const animalsRes = await animalsApi.getAll(groupId, statusFilter, nameSearch);
      setAnimals(animalsRes.data);
    } catch (error) {
      console.error('Failed to load animals:', error);
    }
  }, [statusFilter, nameSearch]);

  // Update view mode when URL search params change
  useEffect(() => {
    const viewParam = searchParams.get('view') as ViewMode;
    if (viewParam && (viewParam === 'activity' || viewParam === 'animals' || viewParam === 'protocols')) {
      setViewMode(viewParam);
    }
  }, [searchParams]);

  useEffect(() => {
    if (id) {
      loadGroupData(Number(id));
      // Load animals immediately to show count in tab
      loadAnimals(Number(id));
    }
    // Load all groups for the switcher
    loadAllGroups();
  }, [id, loadAnimals]);

  useEffect(() => {
    // Reload animals when filters change
    if (id) {
      loadAnimals(Number(id));
    }
  }, [id, loadAnimals]);

  const handleGroupSwitch = async (newGroupId: number) => {
    if (newGroupId !== Number(id)) {
      // Update default group preference
      try {
        await authApi.setDefaultGroup(newGroupId);
      } catch (error) {
        console.error('Failed to set default group:', error);
      }
      // Navigate to the new group
      navigate(`/groups/${newGroupId}`);
    }
  };

  const handleAnnouncementSuccess = () => {
    setShowAnnouncementForm(false);
    // Refresh activity feed by changing key
    setActivityFeedKey(prev => prev + 1);
  };

  if (loading) {
    return (
      <div className="group-page">
        <SkeletonLoader variant="rectangular" height="300px" />
        <div className="group-header">
          <SkeletonLoader variant="text" width="150px" />
          <SkeletonLoader variant="text" width="60%" height="2rem" />
          <SkeletonLoader variant="text" width="80%" />
        </div>
        <div className="skeleton-grid">
          <SkeletonLoader variant="card" count={6} />
        </div>
      </div>
    );
  }

  if (error || !group) {
    return (
      <div className="group-page">
        <ErrorState
          title={error ? 'Unable to Load Group' : 'Group Not Found'}
          message={error || 'The group you\'re looking for doesn\'t exist or you don\'t have permission to view it.'}
          onRetry={error ? () => {
            setLoading(true);
            loadGroupData(Number(id));
          } : undefined}
          onGoBack={() => navigate('/dashboard')}
          goBackLabel="Back to Dashboard"
        />
      </div>
    );
  }

  return (
    <div className="group-page">
      {/* Hero Image */}
      {group.hero_image_url && (
        <div 
          className="group-hero-image" 
          style={{ backgroundImage: `url(${group.hero_image_url})` }}
          role="img"
          aria-label={`${group.name} hero image`}
        />
      )}

      {/* Group Header */}
      <div className="group-header">
        <div className="group-header-title">
          <h1>{group.name}</h1>
          {/* Group Switcher */}
          {groups.length > 1 && (
            <div className="group-switcher">
              <label htmlFor="group-select" className="sr-only">Switch group:</label>
              <select
                id="group-select"
                value={id}
                onChange={(e) => handleGroupSwitch(Number(e.target.value))}
                className="group-select"
                aria-label="Switch to a different group"
              >
                {groups.map(g => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>
        <p className="group-description">{group.description}</p>
      </div>

      {/* View Mode Tabs */}
      <div className="group-tabs" role="tablist" aria-label="Group view modes">
        <button
          role="tab"
          aria-selected={viewMode === 'activity'}
          aria-controls="activity-panel"
          id="activity-tab"
          className={`group-tab ${viewMode === 'activity' ? 'group-tab--active' : ''}`}
          onClick={() => setViewMode('activity')}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
          <span>Activity Feed</span>
        </button>
        <button
          role="tab"
          aria-selected={viewMode === 'animals'}
          aria-controls="animals-panel"
          id="animals-tab"
          className={`group-tab ${viewMode === 'animals' ? 'group-tab--active' : ''}`}
          onClick={() => setViewMode('animals')}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <circle cx="12" cy="8" r="4" />
            <path d="M6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2" />
          </svg>
          <span>Animals ({animals.length})</span>
        </button>
        {group.has_protocols && (
          <button
            role="tab"
            aria-selected={viewMode === 'protocols'}
            aria-controls="protocols-panel"
            id="protocols-tab"
            className={`group-tab ${viewMode === 'protocols' ? 'group-tab--active' : ''}`}
            onClick={() => setViewMode('protocols')}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <path d="M14 2v6h6M16 13H8M16 17H8M10 9H8" />
            </svg>
            <span>Protocols</span>
          </button>
        )}
      </div>

      {/* Group Admin Quick Links - shown to group admins and site admins */}
      {(membership?.is_group_admin || membership?.is_site_admin) && (
        <div className="group-admin-links" role="navigation" aria-label="Group administration links">
          <span className="group-admin-links__title">Quick Links:</span>
          <Link to={`/groups/${id}/animals/new`} className="group-admin-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="16" />
              <line x1="8" y1="12" x2="16" y2="12" />
            </svg>
            <span>Add Animal</span>
          </Link>
          {group?.name === 'modsquad' && (
            <Link to="/admin/animal-tags" className="group-admin-link">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z" />
                <line x1="7" y1="7" x2="7.01" y2="7" />
              </svg>
              <span>Manage Tags</span>
            </Link>
          )}
          {membership?.is_site_admin && (
            <>
              <Link to="/admin/users" className="group-admin-link">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                </svg>
                <span>Manage Users</span>
              </Link>
              <Link to="/admin/groups" className="group-admin-link">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <rect x="3" y="3" width="7" height="7" rx="1" />
                  <rect x="14" y="3" width="7" height="7" rx="1" />
                  <rect x="14" y="14" width="7" height="7" rx="1" />
                  <rect x="3" y="14" width="7" height="7" rx="1" />
                </svg>
                <span>Manage Groups</span>
              </Link>
            </>
          )}
        </div>
      )}

      {/* Activity Feed View */}
      {viewMode === 'activity' && (
        <div 
          role="tabpanel"
          id="activity-panel"
          aria-labelledby="activity-tab"
          className="group-content"
        >
          {/* Quick Actions Bar */}
          <div className="quick-actions-bar">
            <div className="quick-actions-bar__filters">
              <label htmlFor="activity-filter" className="sr-only">Filter activity type</label>
              <select
                id="activity-filter"
                className="quick-actions-bar__select"
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as FilterType)}
                aria-label="Filter activity by type"
              >
                <option value="all">All Activity</option>
                <option value="comments">Comments Only</option>
                <option value="announcements">Announcements Only</option>
              </select>
            </div>
            <div className="quick-actions-bar__actions">
              <button
                className="quick-actions-bar__button quick-actions-bar__button--primary"
                onClick={() => setShowAnnouncementForm(true)}
                aria-label="Post new announcement"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
                  <path d="M13.73 21a2 2 0 0 1-3.46 0" />
                </svg>
                <span>New Announcement</span>
              </button>
            </div>
          </div>

          {/* Activity Feed Component */}
          <ActivityFeed 
            key={activityFeedKey}
            groupId={Number(id)} 
            filterType={filterType} 
          />
        </div>
      )}

      {/* Animals Grid View */}
      {viewMode === 'animals' && (
        <div 
          role="tabpanel"
          id="animals-panel"
          aria-labelledby="animals-tab"
          className="group-content"
        >
          <div className="animals-section">
            <div className="section-header">
              <h2>Animals</h2>
              {(membership?.is_group_admin || membership?.is_site_admin) && (
                <Link to={`/groups/${id}/animals/new`} className="btn-primary">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  <span>Add Animal</span>
                </Link>
              )}
            </div>

            {/* Filters */}
            <div className="filters-section">
              <div className="filter-group">
                <label htmlFor="status-filter">Status:</label>
                <select
                  id="status-filter"
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  aria-label="Filter animals by status"
                >
                  <option value="">Available & Bite Quarantine (default)</option>
                  <option value="all">All Animals</option>
                  <option value="available">Available Only</option>
                  <option value="foster">Foster</option>
                  <option value="bite_quarantine">Bite Quarantine Only</option>
                  <option value="archived">Archived</option>
                </select>
              </div>
              <div className="filter-group">
                <label htmlFor="name-search">Search by name:</label>
                <input
                  id="name-search"
                  type="text"
                  placeholder="Enter animal name..."
                  value={nameSearch}
                  onChange={(e) => setNameSearch(e.target.value)}
                  aria-label="Search animals by name"
                />
              </div>
            </div>

            {/* Animals Grid */}
            {animals.length === 0 ? (
              <EmptyState
                icon={
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                }
                title={statusFilter || nameSearch ? 'No animals match your filters' : `No animals in ${group.name} yet`}
                description={
                  statusFilter || nameSearch
                    ? 'Try adjusting your search or filter to see more results.'
                    : (membership?.is_group_admin || membership?.is_site_admin)
                      ? `Get started by adding your first animal to ${group.name}. Animals added here will be visible to all group members.`
                      : `This group doesn't have any animals yet. An admin will need to add animals before volunteers can share updates.`
                }
                primaryAction={
                  (statusFilter || nameSearch)
                    ? {
                        label: 'Clear Filters',
                        onClick: () => {
                          setStatusFilter('');
                          setNameSearch('');
                        },
                      }
                    : (membership?.is_group_admin || membership?.is_site_admin)
                      ? {
                          label: 'Add First Animal',
                          onClick: () => navigate(`/groups/${id}/animals/new`),
                        }
                      : undefined
                }
              />
            ) : (
              <div className="animals-grid">
                {animals.map((animal) => {
                  // Calculate duplicate name info
                  const duplicates = animals.filter(a => 
                    a.name.toLowerCase() === animal.name.toLowerCase()
                  );
                  const hasDuplicates = duplicates.length > 1;
                  const duplicateIndex = hasDuplicates 
                    ? duplicates.sort((a, b) => a.id - b.id).findIndex(a => a.id === animal.id) + 1
                    : 0;
                  
                  return (
                    <Link
                      key={animal.id}
                      to={`/groups/${id}/animals/${animal.id}/view`}
                      className="animal-card"
                    >
                      {animal.image_url && (
                        <img
                          src={animal.image_url}
                          alt={animal.name}
                          className="animal-card-image"
                        />
                      )}
                      <div className="animal-info">
                        <div className="animal-header-badges">
                          <h3>{animal.name}</h3>
                          {hasDuplicates && (
                            <span 
                              className="badge badge-duplicate" 
                              title={`${duplicates.length} animals named "${animal.name}"`}
                              aria-label={`${duplicateIndex} of ${duplicates.length} animals named ${animal.name}`}
                            >
                              {duplicateIndex} of {duplicates.length}
                            </span>
                          )}
                          {animal.return_count > 0 && (
                            <span 
                              className="badge badge-returning" 
                              title={`This animal has returned to the shelter ${animal.return_count} time${animal.return_count > 1 ? 's' : ''}`}
                              aria-label={`Returning animal - ${animal.return_count} return${animal.return_count > 1 ? 's' : ''}`}
                            >
                              ↩ Returned {animal.return_count}×
                            </span>
                          )}
                        </div>
                        {animal.breed && <p className="breed">{animal.breed}</p>}
                        <p className="age">{animal.age} years old</p>
                        <span className={`status ${animal.status}`}>{animal.status}</span>
                        {animal.tags && animal.tags.length > 0 && (
                          <div className="animal-tags">
                            {animal.tags.map((tag) => (
                              <span
                                key={tag.id}
                                className="animal-tag"
                                style={{ 
                                  backgroundColor: `${tag.color}15`,
                                  color: tag.color,
                                  borderColor: tag.color
                                }}
                                title={tag.name}
                              >
                                {tag.name}
                              </span>
                            ))}
                          </div>
                        )}
                        {animal.status === 'bite_quarantine' && animal.quarantine_start_date && (
                          <p className="quarantine-end-date">
                            Ends: {(() => {
                              const startDate = new Date(animal.quarantine_start_date);
                              const endDate = new Date(startDate);
                              endDate.setDate(endDate.getDate() + 10);
                              while (endDate.getDay() === 0 || endDate.getDay() === 6) {
                                endDate.setDate(endDate.getDate() + 1);
                              }
                              return endDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                            })()}
                          </p>
                        )}
                      </div>
                    </Link>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Protocols View */}
      {viewMode === 'protocols' && group.has_protocols && (
        <div 
          role="tabpanel"
          id="protocols-panel"
          aria-labelledby="protocols-tab"
          className="group-content"
        >
          <ProtocolsList groupId={Number(id)} />
        </div>
      )}

      {/* Announcement Form Modal */}
      <Modal
        isOpen={showAnnouncementForm}
        onClose={() => setShowAnnouncementForm(false)}
        title="Post Announcement"
      >
        <AnnouncementForm
          groupId={Number(id)}
          onSuccess={handleAnnouncementSuccess}
          onCancel={() => setShowAnnouncementForm(false)}
        />
      </Modal>
    </div>
  );
};

export default GroupPage;
