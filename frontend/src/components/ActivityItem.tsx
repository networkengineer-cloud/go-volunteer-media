import React from 'react';
import { Link } from 'react-router-dom';
import type { ActivityItem as ActivityItemType } from '../api/client';
import './ActivityItem.css';

interface ActivityItemProps {
  item: ActivityItemType;
  groupId: number;
  onImageClick?: (imageUrl: string) => void;
}

const ActivityItem: React.FC<ActivityItemProps> = ({ item, groupId, onImageClick }) => {
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
    });
  };

  return (
    <article className={`activity-item activity-item--${item.type}`} role="article">
      {/* Header with user info and timestamp */}
      <div className="activity-item__header">
        <div className="activity-item__user-info">
          <div className="activity-item__avatar" aria-hidden="true">
            {item.user?.username?.charAt(0).toUpperCase() || '?'}
          </div>
          <div className="activity-item__meta">
            <span 
              className="activity-item__username" 
              title={`Posted by ${item.user?.username || 'Unknown'}`}
            >
              {item.user?.username || 'Unknown'}
            </span>
            <time className="activity-item__timestamp" dateTime={item.created_at}>
              {formatDate(item.created_at)}
            </time>
          </div>
        </div>
        <span className={`activity-item__type-badge activity-item__type-badge--${item.type}`}>
          {item.type === 'announcement' ? (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
              <path d="M13.73 21a2 2 0 0 1-3.46 0" />
            </svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
            </svg>
          )}
          <span className="sr-only">{item.type === 'announcement' ? 'Announcement' : 'Comment'}</span>
        </span>
      </div>

      {/* Content */}
      <div className="activity-item__content">
        {/* Title for announcements */}
        {item.type === 'announcement' && item.title && (
          <h3 className="activity-item__title">{item.title}</h3>
        )}

        {/* Animal link for comments */}
        {item.type === 'comment' && item.animal && (
          <div className="activity-item__animal-link">
            <Link 
              to={`/groups/${groupId}/animals/${item.animal_id}/view`}
              className="activity-item__animal-name"
            >
              {item.animal.image_url && (
                <img 
                  src={item.animal.image_url} 
                  alt={item.animal.name}
                  className="activity-item__animal-thumbnail"
                />
              )}
              <span>{item.animal.name}</span>
            </Link>
          </div>
        )}

        {/* Content text */}
        <p className="activity-item__text">{item.content}</p>

        {/* Image attachment */}
        {item.image_url && (
          <div className="activity-item__image-wrapper">
            <img
              src={item.image_url}
              alt="Activity attachment"
              className="activity-item__image"
              onClick={() => onImageClick?.(item.image_url!)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  onImageClick?.(item.image_url!);
                }
              }}
              tabIndex={0}
              role="button"
              aria-label="View full size image"
            />
          </div>
        )}

        {/* Tags for comments */}
        {item.type === 'comment' && item.tags && item.tags.length > 0 && (
          <div className="activity-item__tags" role="list" aria-label="Comment tags">
            {item.tags.map(tag => (
              <span 
                key={tag.id}
                className="activity-item__tag"
                style={{ 
                  backgroundColor: tag.color,
                  color: getContrastColor(tag.color)
                }}
                role="listitem"
              >
                {tag.name}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Footer with action buttons */}
      <div className="activity-item__footer">
        {item.type === 'comment' && item.animal && (
          <Link 
            to={`/groups/${groupId}/animals/${item.animal_id}/view`}
            className="activity-item__action-link"
          >
            View animal
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </Link>
        )}
      </div>
    </article>
  );
};

// Helper function to determine text color based on background
function getContrastColor(hexColor: string): string {
  // Remove # if present
  const hex = hexColor.replace('#', '');
  
  // Convert to RGB
  const r = parseInt(hex.substr(0, 2), 16);
  const g = parseInt(hex.substr(2, 2), 16);
  const b = parseInt(hex.substr(4, 2), 16);
  
  // Calculate relative luminance
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  
  // Return white for dark backgrounds, black for light backgrounds
  return luminance > 0.5 ? '#000000' : '#FFFFFF';
}

export default ActivityItem;
