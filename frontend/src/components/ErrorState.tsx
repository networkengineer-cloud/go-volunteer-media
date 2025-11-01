import React from 'react';
import './ErrorState.css';

interface ErrorStateProps {
  title?: string;
  message: string;
  icon?: React.ReactNode;
  onRetry?: () => void;
  onGoBack?: () => void;
  retryLabel?: string;
  goBackLabel?: string;
}

const ErrorState: React.FC<ErrorStateProps> = ({
  title = 'Something went wrong',
  message,
  icon,
  onRetry,
  onGoBack,
  retryLabel = 'Try Again',
  goBackLabel = 'Go Back',
}) => {
  const defaultIcon = (
    <svg
      width="64"
      height="64"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="8" x2="12" y2="12" />
      <line x1="12" y1="16" x2="12.01" y2="16" />
    </svg>
  );

  return (
    <div className="error-state" role="alert" aria-live="assertive">
      <div className="error-state-icon">{icon || defaultIcon}</div>
      <h2 className="error-state-title">{title}</h2>
      <p className="error-state-message">{message}</p>
      {(onRetry || onGoBack) && (
        <div className="error-state-actions">
          {onRetry && (
            <button
              onClick={onRetry}
              className="btn-primary"
              aria-label={retryLabel}
            >
              {retryLabel}
            </button>
          )}
          {onGoBack && (
            <button
              onClick={onGoBack}
              className="btn-secondary"
              aria-label={goBackLabel}
            >
              {goBackLabel}
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default ErrorState;
