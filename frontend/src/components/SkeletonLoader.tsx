import React from 'react';
import './SkeletonLoader.css';

interface SkeletonLoaderProps {
  variant?: 'text' | 'circular' | 'rectangular' | 'card' | 'comment';
  width?: string | number;
  height?: string | number;
  count?: number;
  className?: string;
}

const SkeletonLoader: React.FC<SkeletonLoaderProps> = ({
  variant = 'text',
  width,
  height,
  count = 1,
  className = '',
}) => {
  const getVariantClass = () => {
    switch (variant) {
      case 'circular':
        return 'skeleton-circular';
      case 'rectangular':
        return 'skeleton-rectangular';
      case 'card':
        return 'skeleton-card';
      case 'comment':
        return 'skeleton-comment';
      default:
        return 'skeleton-text';
    }
  };

  const style: React.CSSProperties = {
    width: typeof width === 'number' ? `${width}px` : width,
    height: typeof height === 'number' ? `${height}px` : height,
  };

  const renderSkeleton = () => (
    <div
      className={`skeleton ${getVariantClass()} ${className}`}
      style={style}
      role="status"
      aria-busy="true"
      aria-label="Loading content"
    >
      <span className="sr-only">Loading...</span>
    </div>
  );

  if (variant === 'card') {
    return (
      <div className="skeleton-card-wrapper" role="status" aria-busy="true">
        <div className="skeleton skeleton-card-image" />
        <div className="skeleton-card-content">
          <div className="skeleton skeleton-text" style={{ width: '70%' }} />
          <div className="skeleton skeleton-text" style={{ width: '50%' }} />
          <div className="skeleton skeleton-text" style={{ width: '90%' }} />
        </div>
      </div>
    );
  }

  if (variant === 'comment') {
    return (
      <div className="skeleton-comment-wrapper" role="status" aria-busy="true">
        <div className="skeleton skeleton-circular" style={{ width: '2.5rem', height: '2.5rem' }} />
        <div className="skeleton-comment-content">
          <div className="skeleton skeleton-text" style={{ width: '30%' }} />
          <div className="skeleton skeleton-text" style={{ width: '95%' }} />
          <div className="skeleton skeleton-text" style={{ width: '80%' }} />
        </div>
      </div>
    );
  }

  if (count > 1) {
    return (
      <>
        {Array.from({ length: count }).map((_, index) => (
          <React.Fragment key={index}>{renderSkeleton()}</React.Fragment>
        ))}
      </>
    );
  }

  return renderSkeleton();
};

export default SkeletonLoader;
