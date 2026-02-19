import React from 'react';
import './ConfirmDialog.css';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'danger' | 'warning' | 'info';
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  isOpen,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'danger',
  onConfirm,
  onCancel,
}) => {
  const [isLoading, setIsLoading] = React.useState(false);
  const isLoadingRef = React.useRef(false);
  const dialogRef = React.useRef<HTMLDivElement>(null);
  const cancelButtonRef = React.useRef<HTMLButtonElement>(null);

  // Keep ref in sync so the escape handler captures current value without re-registering
  isLoadingRef.current = isLoading;

  // Reset loading state when dialog opens; register escape key handler
  React.useEffect(() => {
    if (isOpen) {
      setIsLoading(false);
      cancelButtonRef.current?.focus();

      const handleEscape = (e: KeyboardEvent) => {
        if (e.key === 'Escape' && !isLoadingRef.current) {
          onCancel();
        }
      };
      document.addEventListener('keydown', handleEscape);

      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen, onCancel]);

  const handleConfirm = async () => {
    setIsLoading(true);
    try {
      await onConfirm();
    } finally {
      setIsLoading(false);
      onCancel();
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="confirm-dialog-backdrop"
      onClick={isLoading ? undefined : onCancel}
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-message"
    >
      <div
        ref={dialogRef}
        className={`confirm-dialog confirm-dialog--${variant}`}
        onClick={(e) => e.stopPropagation()}
        tabIndex={-1}
      >
        <div className="confirm-dialog__icon">
          {variant === 'danger' && (
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
          )}
          {variant === 'warning' && (
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          )}
          {variant === 'info' && (
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          )}
        </div>

        <h2 id="confirm-dialog-title" className="confirm-dialog__title">
          {title}
        </h2>

        <p id="confirm-dialog-message" className="confirm-dialog__message">
          {message}
        </p>

        <div className="confirm-dialog__actions">
          <button
            type="button"
            className="btn-secondary"
            onClick={onCancel}
            disabled={isLoading}
            autoFocus
            ref={cancelButtonRef}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            className={`btn-${variant}`}
            onClick={handleConfirm}
            disabled={isLoading}
            aria-busy={isLoading}
          >
            {isLoading ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ConfirmDialog;
