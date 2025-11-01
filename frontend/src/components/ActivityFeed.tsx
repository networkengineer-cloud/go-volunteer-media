import React, { useEffect, useState, useRef, useCallback } from 'react';
import { groupsApi } from '../api/client';
import type { ActivityItem as ActivityItemType } from '../api/client';
import ActivityItem from './ActivityItem';
import SkeletonLoader from './SkeletonLoader';
import EmptyState from './EmptyState';
import Modal from './Modal';
import './ActivityFeed.css';

interface ActivityFeedProps {
  groupId: number;
  filterType?: 'all' | 'comments' | 'announcements';
}

const ActivityFeed: React.FC<ActivityFeedProps> = ({ groupId, filterType = 'all' }) => {
  const [items, setItems] = useState<ActivityItemType[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [offset, setOffset] = useState(0);
  const [error, setError] = useState<string>('');
  const [selectedImage, setSelectedImage] = useState<string | null>(null);
  const observerTarget = useRef<HTMLDivElement>(null);

  const limit = 20;

  const loadItems = useCallback(async (isLoadMore = false) => {
    try {
      const currentOffset = isLoadMore ? offset : 0;
      
      if (isLoadMore) {
        setLoadingMore(true);
      } else {
        setLoading(true);
      }

      const response = await groupsApi.getActivityFeed(groupId, {
        limit,
        offset: currentOffset,
        type: filterType,
      });

      if (isLoadMore) {
        setItems(prev => [...prev, ...response.data.items]);
      } else {
        setItems(response.data.items);
      }

      setHasMore(response.data.hasMore);
      setOffset(currentOffset + response.data.items.length);
      setError('');
    } catch (error: any) {
      console.error('Failed to load activity feed:', error);
      setError(error.response?.data?.error || 'Failed to load activity feed. Please try again.');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [groupId, filterType, offset]);

  // Load initial items
  useEffect(() => {
    setOffset(0);
    loadItems(false);
  }, [groupId, filterType]);

  // Intersection Observer for infinite scroll
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore && !loading) {
          loadItems(true);
        }
      },
      { threshold: 0.1 }
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasMore, loadingMore, loading, loadItems]);

  const handleImageClick = (imageUrl: string) => {
    setSelectedImage(imageUrl);
  };

  if (loading && items.length === 0) {
    return (
      <div className="activity-feed" role="feed" aria-busy="true" aria-label="Loading activity feed">
        <div className="activity-feed__skeleton">
          <SkeletonLoader variant="comment" count={5} />
        </div>
      </div>
    );
  }

  if (error && items.length === 0) {
    return (
      <div className="activity-feed">
        <EmptyState
          icon={
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4M12 16h.01" />
            </svg>
          }
          title="Unable to Load Activity"
          description={error}
          primaryAction={{
            label: 'Try Again',
            onClick: () => loadItems(false),
          }}
        />
      </div>
    );
  }

  if (items.length === 0) {
    const emptyStateProps = {
      all: {
        title: 'No Activity Yet',
        description: 'Be the first to share an update or comment on an animal in this group! Your posts and comments will appear here.',
      },
      comments: {
        title: 'No Comments Yet',
        description: 'No one has commented on animals in this group yet. Be the first to share your thoughts!',
      },
      announcements: {
        title: 'No Announcements Yet',
        description: 'No announcements have been posted to this group yet.',
      },
    }[filterType];

    return (
      <div className="activity-feed">
        <EmptyState
          icon={
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
              <circle cx="9" cy="10" r="1" fill="currentColor" />
              <circle cx="12" cy="10" r="1" fill="currentColor" />
              <circle cx="15" cy="10" r="1" fill="currentColor" />
            </svg>
          }
          title={emptyStateProps.title}
          description={emptyStateProps.description}
        />
      </div>
    );
  }

  return (
    <div className="activity-feed" role="feed" aria-label="Activity feed">
      <div className="activity-feed__list">
        {items.map((item) => (
          <ActivityItem
            key={`${item.type}-${item.id}`}
            item={item}
            groupId={groupId}
            onImageClick={handleImageClick}
          />
        ))}
      </div>

      {/* Infinite scroll trigger */}
      {hasMore && (
        <div
          ref={observerTarget}
          className="activity-feed__load-trigger"
          aria-hidden="true"
        />
      )}

      {/* Loading more indicator */}
      {loadingMore && (
        <div className="activity-feed__loading-more" aria-live="polite" aria-busy="true">
          <div className="activity-feed__spinner" aria-label="Loading more items">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
            </svg>
          </div>
          <span>Loading more...</span>
        </div>
      )}

      {/* End of feed message */}
      {!hasMore && items.length > 0 && (
        <div className="activity-feed__end-message" role="status">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M5 13l4 4L19 7" />
          </svg>
          <span>You've reached the end of the feed</span>
        </div>
      )}

      {/* Image modal */}
      {selectedImage && (
        <Modal
          isOpen={!!selectedImage}
          onClose={() => setSelectedImage(null)}
          title="Image Preview"
        >
          <div className="activity-feed__image-modal">
            <img src={selectedImage} alt="Full size preview" />
          </div>
        </Modal>
      )}
    </div>
  );
};

export default ActivityFeed;
