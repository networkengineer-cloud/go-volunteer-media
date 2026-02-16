import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom';
import { groupsApi, animalsApi, authApi } from '../api/client';
import type { Group, Animal, GroupMembership, ActivityItem, GroupMember } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import SessionCommentDisplay from '../components/SessionCommentDisplay';
import AnnouncementForm from '../components/AnnouncementForm';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import Modal from '../components/Modal';
import ProtocolsList from '../components/ProtocolsList';
import './GroupPage.css';

type ViewMode = 'activity' | 'animals' | 'protocols' | 'members';
type FilterType = 'all' | 'comments' | 'announcements';

const GroupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  useAuth(); // Ensure user is authenticated
  const navigate = useNavigate();
  const toast = useToast();
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
  const [showProtocolForm, setShowProtocolForm] = useState(false);
  const [activityFeedKey, setActivityFeedKey] = useState(0);
  const [showLengthOfStay, setShowLengthOfStay] = useState(false);
  
  // Activity Feed state (integrated from ActivityFeedPage)
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [activityTotal, setActivityTotal] = useState(0);
  const [activityHasMore, setActivityHasMore] = useState(false);
  const [activityLoading, setActivityLoading] = useState(false);
  const [activityLoadingMore, setActivityLoadingMore] = useState(false);
  const [activityError, setActivityError] = useState('');
  // Members state
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [membersError, setMembersError] = useState('');

  const [filterAnimal, setFilterAnimal] = useState<string>('');
  const [animalSearchQuery, setAnimalSearchQuery] = useState<string>('');
  const [showAnimalDropdown, setShowAnimalDropdown] = useState(false);
  const [filterTags, setFilterTags] = useState<string[]>([]);
  const [filterRating, setFilterRating] = useState<string>('');
  const [filterDateFrom, setFilterDateFrom] = useState<string>('');
  const [filterDateTo, setFilterDateTo] = useState<string>('');

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
    } else if (viewParam === 'members' && membership?.is_member) {
      setViewMode(viewParam);
    } else if (viewParam === 'members' && !membership?.is_member) {
      setViewMode('activity');
    }
  }, [searchParams, membership]);

  useEffect(() => {
    const loadPreferences = async () => {
      try {
        const prefsRes = await authApi.getEmailPreferences();
        setShowLengthOfStay(prefsRes.data.show_length_of_stay || false);
      } catch (error) {
        console.error('Failed to load preferences:', error);
      }
    };
    
    loadPreferences();
    
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
    // Refresh activity feed
    if (id) {
      loadActivityFeed(Number(id), true);
    }
  };

  // Load activity feed with filters (integrated from ActivityFeedPage)
  const loadActivityFeed = async (groupId: number, reset = false) => {
    if (reset) {
      setActivityLoading(true);
      setActivities([]);
    } else {
      setActivityLoadingMore(true);
    }

    try {
      const offset = reset ? 0 : activities.length;
      const response = await groupsApi.getActivityFeed(groupId, {
        limit: 20,
        offset,
        type: filterType,
        animal: filterAnimal ? Number(filterAnimal) : undefined,
        tags: filterTags.length > 0 ? filterTags.join(',') : undefined,
        rating: filterRating || undefined,
        from: filterDateFrom || undefined,
        to: filterDateTo || undefined,
      });

      if (reset) {
        setActivities(response.data.items);
      } else {
        setActivities([...activities, ...response.data.items]);
      }
      
      setActivityTotal(response.data.total);
      setActivityHasMore(response.data.hasMore);
      setActivityError('');
    } catch (err: any) {
      console.error('Failed to load activity feed:', err);
      const errorMessage = err.response?.data?.error || 'Failed to load activity feed. Please try again.';
      setActivityError(errorMessage);
      toast.showError(errorMessage);
    } finally {
      setActivityLoading(false);
      setActivityLoadingMore(false);
    }
  };

  const handleLoadMoreActivity = () => {
    if (id) {
      loadActivityFeed(Number(id), false);
    }
  };

  const clearActivityFilters = () => {
    setFilterType('all');
    setFilterAnimal('');
    setAnimalSearchQuery('');
    setFilterTags([]);
    setFilterRating('');
    setFilterDateFrom('');
    setFilterDateTo('');
  };

  const toggleTag = (tagName: string) => {
    if (filterTags.includes(tagName)) {
      setFilterTags(filterTags.filter(t => t !== tagName));
    } else {
      setFilterTags([...filterTags, tagName]);
    }
  };

  const hasActiveActivityFilters = filterType !== 'all' || filterAnimal || filterTags.length > 0 || filterRating || filterDateFrom || filterDateTo;

  // Filter animals based on search query
  const filteredAnimals = animals.filter(animal =>
    animal.name.toLowerCase().includes(animalSearchQuery.toLowerCase())
  );

  // Handle animal selection from dropdown
  const selectAnimal = (animalId: number, animalName: string) => {
    setFilterAnimal(animalId.toString());
    setAnimalSearchQuery(animalName);
    setShowAnimalDropdown(false);
  };

  // Clear animal filter
  const clearAnimalFilter = () => {
    setFilterAnimal('');
    setAnimalSearchQuery('');
    setShowAnimalDropdown(false);
  };

  // Load members when switching to members tab
  const loadMembers = async (groupId: number) => {
    setMembersLoading(true);
    setMembersError('');
    try {
      const res = await groupsApi.getMembers(groupId);
      setMembers(res.data);
    } catch (err: unknown) {
      console.error('Failed to load members:', err);
      const axiosError = err as { response?: { data?: { error?: string } } };
      setMembersError(axiosError.response?.data?.error || 'Failed to load members');
    } finally {
      setMembersLoading(false);
    }
  };

  useEffect(() => {
    if (id && viewMode === 'members') {
      loadMembers(Number(id));
    }
  }, [id, viewMode]);

  // Load activity feed when filters change or view mode switches to activity
  useEffect(() => {
    if (id && viewMode === 'activity') {
      loadActivityFeed(Number(id), true);
    }
  }, [id, viewMode, filterType, filterAnimal, filterTags, filterRating, filterDateFrom, filterDateTo]);

  // Close animal dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest('.animal-filter-autocomplete')) {
        setShowAnimalDropdown(false);
      }
    };

    if (showAnimalDropdown) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showAnimalDropdown]);

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
        {membership?.is_member && (
          <button
            role="tab"
            aria-selected={viewMode === 'members'}
            aria-controls="members-panel"
            id="members-tab"
            className={`group-tab ${viewMode === 'members' ? 'group-tab--active' : ''}`}
            onClick={() => setViewMode('members')}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
            <span>Members</span>
          </button>
        )}
      </div>

      {/* Group Admin Quick Links - shown only to group admins (site admins already have nav bar) */}
      {membership?.is_group_admin && !membership?.is_site_admin && (
        <div className="group-admin-links" role="navigation" aria-label="Group administration links">
          <span className="group-admin-links__title">Quick Actions:</span>
          <Link to={`/groups/${id}/animals/new`} className="group-admin-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="16" />
              <line x1="8" y1="12" x2="16" y2="12" />
            </svg>
            <span>Add Animal</span>
          </Link>
          <button 
            type="button"
            className="group-admin-link"
            onClick={() => setShowAnnouncementForm(true)}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
              <path d="M13.73 21a2 2 0 0 1-3.46 0" />
            </svg>
            <span>Announcement</span>
          </button>
          <Link to="/admin/animal-tags" className="group-admin-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z" />
              <line x1="7" y1="7" x2="7.01" y2="7" />
            </svg>
            <span>Manage Tags</span>
          </Link>
          {group.has_protocols && (
            <button 
              type="button"
              className="group-admin-link"
              onClick={() => {
                setViewMode('protocols');
                setShowProtocolForm(true);
              }}
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                <path d="M14 2v6h6M16 13H8M16 17H8M10 9H8" />
              </svg>
              <span>Add Protocol</span>
            </button>
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
          {/* Enhanced Activity Filter Bar */}
          <div className="activity-filter-bar">
            <div className="filter-row">
              <select
                id="activity-filter"
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as FilterType)}
                className="filter-select"
                aria-label="Filter activity by type"
              >
                <option value="all">All Activity</option>
                <option value="comments">Comments Only</option>
                <option value="announcements">Announcements Only</option>
              </select>

              {/* Searchable Animal Filter with Autocomplete */}
              <div className="animal-filter-autocomplete">
                <input
                  type="text"
                  value={animalSearchQuery}
                  onChange={(e) => {
                    setAnimalSearchQuery(e.target.value);
                    setShowAnimalDropdown(true);
                  }}
                  onFocus={() => setShowAnimalDropdown(true)}
                  placeholder="Search animals..."
                  className="filter-input"
                  aria-label="Search and filter by animal"
                />
                {animalSearchQuery && (
                  <button
                    type="button"
                    onClick={clearAnimalFilter}
                    className="animal-filter-clear"
                    aria-label="Clear animal filter"
                  >
                    ×
                  </button>
                )}
                {showAnimalDropdown && filteredAnimals.length > 0 && (
                  <div className="animal-dropdown">
                    {filteredAnimals.slice(0, 10).map(animal => (
                      <button
                        key={animal.id}
                        type="button"
                        onClick={() => selectAnimal(animal.id, animal.name)}
                        className="animal-dropdown-item"
                      >
                        🐕 {animal.name}
                      </button>
                    ))}
                    {filteredAnimals.length > 10 && (
                      <div className="animal-dropdown-more">
                        +{filteredAnimals.length - 10} more animals...
                      </div>
                    )}
                  </div>
                )}
              </div>

              <select
                value={filterRating}
                onChange={(e) => setFilterRating(e.target.value)}
                className="filter-select"
                aria-label="Filter by rating"
              >
                <option value="">All Ratings</option>
                <option value="5">😄 Great (5)</option>
                <option value="4">🙂 Good (4)</option>
                <option value="3">😐 Okay (3)</option>
                <option value="2">😕 Fair (2)</option>
                <option value="1">😟 Poor (1)</option>
                <option value="poor">😟 Poor Sessions (1-2)</option>
              </select>

              <div className="date-range-group" aria-label="Date range">
                <div className="date-range-inputs">
                  <input
                    type="date"
                    value={filterDateFrom}
                    onChange={(e) => setFilterDateFrom(e.target.value)}
                    className="filter-input"
                    placeholder="From"
                    aria-label="Filter from date"
                  />
                  <span className="date-range-separator" aria-hidden="true">–</span>
                  <input
                    type="date"
                    value={filterDateTo}
                    onChange={(e) => setFilterDateTo(e.target.value)}
                    className="filter-input"
                    placeholder="To"
                    aria-label="Filter to date"
                  />
                </div>
              </div>

              {hasActiveActivityFilters && (
                <button onClick={clearActivityFilters} className="btn-clear-filters">
                  Clear Filters
                </button>
              )}
            </div>

            {/* Quick Filters */}
            <div className="quick-filters">
              <span className="quick-filters-label">Quick Filters:</span>
              <button
                className={`quick-filter-btn ${filterTags.includes('behavior') ? 'active' : ''}`}
                onClick={() => toggleTag('behavior')}
                aria-pressed={filterTags.includes('behavior')}
              >
                ⚠️ Behavior
              </button>
              <button
                className={`quick-filter-btn ${filterTags.includes('medical') ? 'active' : ''}`}
                onClick={() => toggleTag('medical')}
                aria-pressed={filterTags.includes('medical')}
              >
                🏥 Medical
              </button>
              <button
                className={`quick-filter-btn ${filterRating === 'poor' ? 'active' : ''}`}
                onClick={() => setFilterRating(filterRating === 'poor' ? '' : 'poor')}
                aria-pressed={filterRating === 'poor'}
              >
                😟 Poor Sessions
              </button>
            </div>
          </div>

          {/* Activity Feed Content */}
          {activityLoading ? (
            <div className="activity-list">
              <SkeletonLoader variant="comment" count={5} />
            </div>
          ) : activityError ? (
            <ErrorState
              title="Unable to Load Activity Feed"
              message={activityError}
              onRetry={() => id && loadActivityFeed(Number(id), true)}
            />
          ) : activities.length === 0 ? (
            <div className="empty-state">
              <p>No activity found</p>
              {hasActiveActivityFilters && (
                <button onClick={clearActivityFilters} className="btn-primary">
                  Clear Filters
                </button>
              )}
            </div>
          ) : (
            <div className="activity-list">
              {activities.map((activity) => (
                <div key={`${activity.type}-${activity.id}`} className="activity-card">
                  <div className="activity-header">
                    {activity.animal && (
                      <Link to={`/groups/${id}/animals/${activity.animal.id}/view`} className="activity-animal">
                        <span
                          className={`activity-animal-avatar ${activity.animal.image_url ? '' : 'avatar-missing'}`}
                          aria-hidden="true"
                        >
                          {activity.animal.image_url && (
                            <img
                              src={activity.animal.image_url}
                              alt=""
                              loading="lazy"
                              onError={(e) => {
                                e.currentTarget.style.display = 'none';
                                e.currentTarget.parentElement?.classList.add('avatar-missing');
                              }}
                            />
                          )}
                          <span className="avatar-fallback">
                            {(activity.animal.name?.trim()[0] || '?').toUpperCase()}
                          </span>
                        </span>
                        <span className="activity-animal-name">{activity.animal.name}</span>
                      </Link>
                    )}
                    <span className="activity-meta">
                      by {activity.user?.username} • {new Date(activity.created_at).toLocaleDateString()} at{' '}
                      {new Date(activity.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </div>

                  {activity.tags && activity.tags.length > 0 && (
                    <div className="activity-tags">
                      {activity.tags.map((tag) => (
                        <span key={tag.id} className="tag-badge" style={{ backgroundColor: tag.color }}>
                          🏷️ {tag.name}
                        </span>
                      ))}
                      {activity.metadata?.session_rating && (
                        <span className="session-rating-badge">
                          Session: {['', '😟', '😕', '😐', '🙂', '😄'][activity.metadata.session_rating]}
                        </span>
                      )}
                    </div>
                  )}

                  <div className="activity-content">
                    {activity.type === 'announcement' && activity.title && (
                      <h3 className="activity-title">{activity.title}</h3>
                    )}

                    {activity.type === 'comment' ? (
                      <SessionCommentDisplay comment={activity as any} />
                    ) : (
                      <p className="activity-text">{activity.content}</p>
                    )}

                    {activity.image_url && (
                      <img src={activity.image_url} alt="Activity" className="activity-image" />
                    )}
                  </div>

                  {activity.animal && (
                    <div className="activity-footer">
                      <Link to={`/groups/${id}/animals/${activity.animal.id}/view`} className="btn-view-profile">
                        View {activity.animal.name}'s Profile →
                      </Link>
                    </div>
                  )}
                </div>
              ))}

              {activityHasMore && (
                <div className="load-more-container">
                  <button
                    onClick={handleLoadMoreActivity}
                    className="btn-load-more"
                    disabled={activityLoadingMore}
                  >
                    {activityLoadingMore ? 'Loading...' : 'Load More Activity'}
                  </button>
                  <span className="results-info">
                    Showing {activities.length} of {activityTotal}
                  </span>
                </div>
              )}
            </div>
          )}
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
                        {showLengthOfStay && animal.arrival_date && (
                          <p className="length-of-stay">
                            {(() => {
                              const arrivalDate = new Date(animal.arrival_date);
                              const now = new Date();
                              const diffTime = Math.abs(now.getTime() - arrivalDate.getTime());
                              const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                              return `${diffDays} day${diffDays !== 1 ? 's' : ''} at shelter`;
                            })()}
                          </p>
                        )}
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
          <ProtocolsList 
            groupId={Number(id)} 
            isGroupAdmin={membership?.is_group_admin || membership?.is_site_admin}
            showFormExternal={showProtocolForm}
            onShowFormChange={setShowProtocolForm}
            hideAddButton={true}
          />
        </div>
      )}

      {/* Members View */}
      {viewMode === 'members' && membership?.is_member && (
        <div
          role="tabpanel"
          id="members-panel"
          aria-labelledby="members-tab"
          className="group-content"
        >
          <div className="members-section">
            <div className="section-header">
              <h2>Members</h2>
            </div>

            {membersLoading ? (
              <SkeletonLoader variant="card" count={4} />
            ) : membersError ? (
              <ErrorState
                title="Unable to Load Members"
                message={membersError}
                onRetry={() => id && loadMembers(Number(id))}
              />
            ) : members.length === 0 ? (
              <EmptyState
                icon={
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                  </svg>
                }
                title="No members found"
                description="This group doesn't have any members yet."
              />
            ) : (
              <div className="members-list">
                {members.map((member) => {
                  const displayName = [member.first_name, member.last_name].filter(Boolean).join(' ');
                  const initial = (displayName || member.username).charAt(0).toUpperCase();
                  return (
                    <div key={member.user_id} className="member-card">
                      <div className="member-avatar">{initial}</div>
                      <div className="member-info">
                        <div className="member-name">
                          {displayName || member.username}
                          {displayName && <span className="member-username">@{member.username}</span>}
                        </div>
                        <div className="member-roles">
                          {member.is_site_admin && <span className="member-badge member-badge--site-admin">Site Admin</span>}
                          {member.is_group_admin && <span className="member-badge member-badge--group-admin">Group Admin</span>}
                        </div>
                        {(member.email || member.phone_number) && (
                          <div className="member-contact">
                            {member.email && <span>{member.email}</span>}
                            {member.phone_number && <span>{member.phone_number}</span>}
                          </div>
                        )}
                      </div>
                      <Link to={`/users/${member.user_id}/profile`} className="btn-view-profile">
                        View Profile
                      </Link>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
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
