import React from 'react';
import './EmptyState.css';

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  primaryAction?: {
    label: string;
    onClick: () => void;
  };
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
}

const EmptyState: React.FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  primaryAction,
  secondaryAction,
}) => {
  return (
    <div className="empty-state" role="status" aria-live="polite">
      {icon && <div className="empty-state-icon">{icon}</div>}
      <h2 className="empty-state-title">{title}</h2>
      <p className="empty-state-description">{description}</p>
      {(primaryAction || secondaryAction) && (
        <div className="empty-state-actions">
          {primaryAction && (
            <button
              onClick={primaryAction.onClick}
              className="btn-primary"
              aria-label={primaryAction.label}
            >
              {primaryAction.label}
            </button>
          )}
          {secondaryAction && (
            <button
              onClick={secondaryAction.onClick}
              className="btn-secondary"
              aria-label={secondaryAction.label}
            >
              {secondaryAction.label}
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default EmptyState;
