import React from 'react';
import './LoadingSpinner.css';

interface LoadingSpinnerProps {
  label?: string;
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  label = 'Loading',
}) => {
  return (
    <div
      className="loading-spinner"
      role="status"
      aria-live="polite"
      aria-label={label}
    >
      <div className="spinner" aria-hidden="true" />
      <p className="loading-text">{label}...</p>
    </div>
  );
};

export default LoadingSpinner;
