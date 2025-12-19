import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { groupsApi, animalsApi } from '../api/client';
import type { ActivityItem, Animal } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import SessionCommentDisplay from '../components/SessionCommentDisplay';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import './ActivityFeedPage.css';

const ActivityFeedPage: React.FC = () => {
  const { id: groupId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  useAuth(); // Ensure user is authenticated
  const toast = useToast();

  const [loading, setLoading] = useState(true);
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [total, setTotal] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');

  // Filter state
  const [filtersExpanded, setFiltersExpanded] = useState(false);
  const [filterType, setFilterType] = useState<'all' | 'comments' | 'announcements'>('all');
  const [filterAnimal, setFilterAnimal] = useState<string>('');
  const [filterTags, setFilterTags] = useState<string[]>([]);
  const [filterRating, setFilterRating] = useState<string>('');
  const [filterDateFrom, setFilterDateFrom] = useState<string>('');
  const [filterDateTo, setFilterDateTo] = useState<string>('');

  // Data for filters
  const [animals, setAnimals] = useState<Animal[]>([]);

  // Load initial data
  useEffect(() => {
    if (groupId) {
      loadActivityFeed(true);
      loadAnimals();
    }
  }, [groupId, filterType, filterAnimal, filterTags, filterRating, filterDateFrom, filterDateTo]);

  const loadActivityFeed = async (reset = false) => {
    if (!groupId) return;

    if (reset) {
      setLoading(true);
      setActivities([]);
    } else {
      setLoadingMore(true);
    }

    try {
      const offset = reset ? 0 : activities.length;
      const response = await groupsApi.getActivityFeed(Number(groupId), {
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
      
      setTotal(response.data.total);
      setHasMore(response.data.hasMore);
      setError('');
    } catch (err: unknown) {
      console.error('Failed to load activity feed:', err);
      const errorMessage = (err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to load activity feed. Please try again.';
      setError(errorMessage);
      toast.showError(errorMessage);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  const loadAnimals = async () => {
    if (!groupId) return;
    try {
      const response = await animalsApi.getAll(Number(groupId));
      setAnimals(response.data);
    } catch (err: unknown) {
      console.error('Failed to load animals:', err);
      toast.showError('Failed to load animals for filtering');
    }
  };

  const handleLoadMore = () => {
    loadActivityFeed(false);
  };

  const clearFilters = () => {
    setFilterType('all');
    setFilterAnimal('');
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

  const hasActiveFilters = filterType !== 'all' || filterAnimal || filterTags.length > 0 || filterRating || filterDateFrom || filterDateTo;

  const getActiveFilterCount = () => {
    let count = 0;
    if (filterType !== 'all') count++;
    if (filterAnimal) count++;
    if (filterTags.length > 0) count += filterTags.length;
    if (filterRating) count++;
    if (filterDateFrom || filterDateTo) count++;
    return count;
  };

  const activeFilterCount = getActiveFilterCount();

  if (loading) {
    return (
      <div className="activity-feed-page">
        <div className="page-header">
          <SkeletonLoader variant="text" width="300px" height="2rem" />
        </div>
        <div className="activity-feed-container">
          <SkeletonLoader variant="rectangular" height="80px" />
          <div className="activity-list">
            <SkeletonLoader variant="comment" count={5} />
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="activity-feed-page">
        <ErrorState
          title="Unable to Load Activity Feed"
          message={error}
          onRetry={() => loadActivityFeed(true)}
          onGoBack={() => navigate(`/groups/${groupId}`)}
          goBackLabel="Back to Group"
        />
      </div>
    );
  }

  return (
    <div className="activity-feed-page">
      <div className="page-header">
        <h1>📰 Activity Feed</h1>
        <Link to={`/groups/${groupId}`} className="btn-back">
          ← Back to Group
        </Link>
      </div>

      {/* Filter Bar */}
      <div className="filter-bar">
        <div className="filter-bar-header">
          <h2 className="filter-bar-title">Filters</h2>
          <button 
            onClick={() => setFiltersExpanded(!filtersExpanded)}
            className="filter-toggle-btn"
            aria-expanded={filtersExpanded}
            aria-controls="filter-content"
          >
            {filtersExpanded 
              ? '▲ Hide Filters' 
              : `▼ Show Filters${activeFilterCount > 0 ? ` (${activeFilterCount} active)` : ''}`
            }
          </button>
        </div>
        
        {filtersExpanded && (
          <div id="filter-content" className="filter-content">
            <div className="filter-row">
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as 'all' | 'comments' | 'announcements')}
                className="filter-select"
              >
                <option value="all">All Activity</option>
                <option value="comments">Comments Only</option>
                <option value="announcements">Announcements Only</option>
              </select>

              <select
                value={filterAnimal}
                onChange={(e) => setFilterAnimal(e.target.value)}
                className="filter-select"
              >
                <option value="">All Animals</option>
                {animals.map(animal => (
                  <option key={animal.id} value={animal.id}>{animal.name}</option>
                ))}
              </select>

              <select
                value={filterRating}
                onChange={(e) => setFilterRating(e.target.value)}
                className="filter-select"
              >
                <option value="">All Ratings</option>
                <option value="5">😄 Great (5)</option>
                <option value="4">🙂 Good (4)</option>
                <option value="3">😐 Okay (3)</option>
                <option value="2">😕 Fair (2)</option>
                <option value="1">😟 Poor (1)</option>
                <option value="poor">😟 Poor Sessions (1-2)</option>
              </select>

              <input
                type="date"
                value={filterDateFrom}
                onChange={(e) => setFilterDateFrom(e.target.value)}
                className="filter-input"
                placeholder="From date"
              />

              <input
                type="date"
                value={filterDateTo}
                onChange={(e) => setFilterDateTo(e.target.value)}
                className="filter-input"
                placeholder="To date"
              />

              {hasActiveFilters && (
                <button onClick={clearFilters} className="btn-clear-filters">
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
              >
                ⚠️ Behavior
              </button>
              <button
                className={`quick-filter-btn ${filterTags.includes('medical') ? 'active' : ''}`}
                onClick={() => toggleTag('medical')}
              >
                🏥 Medical
              </button>
              <button
                className={`quick-filter-btn ${filterRating === 'poor' ? 'active' : ''}`}
                onClick={() => setFilterRating(filterRating === 'poor' ? '' : 'poor')}
              >
                😟 Poor Sessions
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Activity List */}
      <div className="activity-list">
        {activities.length === 0 ? (
          <div className="empty-state">
            <p>No activity found</p>
            {hasActiveFilters && (
              <button onClick={clearFilters} className="btn-primary">
                Clear Filters
              </button>
            )}
          </div>
        ) : (
          <>
            {activities.map((activity) => (
              <div key={`${activity.type}-${activity.id}`} className="activity-card">
                <div className="activity-header">
                  {activity.animal && (
                    <Link to={`/groups/${groupId}/animals/${activity.animal.id}/view`} className="activity-animal">
                      🐕 {activity.animal.name}
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
                    <SessionCommentDisplay comment={activity as ActivityItem & { type: 'comment' }} />
                  ) : (
                    <p className="activity-text">{activity.content}</p>
                  )}

                  {activity.image_url && (
                    <img src={activity.image_url} alt="Activity" className="activity-image" />
                  )}
                </div>

                {activity.animal && (
                  <div className="activity-footer">
                    <Link to={`/groups/${groupId}/animals/${activity.animal.id}/view`} className="btn-view-profile">
                      View {activity.animal.name}'s Profile →
                    </Link>
                  </div>
                )}
              </div>
            ))}

            {hasMore && (
              <div className="load-more-container">
                <button
                  onClick={handleLoadMore}
                  className="btn-load-more"
                  disabled={loadingMore}
                >
                  {loadingMore ? 'Loading...' : 'Load More Activity'}
                </button>
                <span className="results-info">
                  Showing {activities.length} of {total}
                </span>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default ActivityFeedPage;
