import React, { useEffect, useState } from 'react';
import { CommentHistory, SessionMetadata } from '../api/client';
import './CommentHistoryModal.css';

interface CommentHistoryModalProps {
  history: CommentHistory[];
  onClose: () => void;
}

const ratingLabels = ['', 'Poor', 'Fair', 'Okay', 'Good', 'Great'];

const CommentHistoryModal: React.FC<CommentHistoryModalProps> = ({ history, onClose }) => {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const renderMetadata = (metadata?: SessionMetadata) => {
    if (!metadata) return null;

    const fields = [
      { label: 'Session Goal', value: metadata.session_goal },
      { label: 'Session Outcome', value: metadata.session_outcome },
      { label: 'Behavior Notes', value: metadata.behavior_notes },
      { label: 'Medical Notes', value: metadata.medical_notes },
      { label: 'Other Notes', value: metadata.other_notes },
    ];

    return (
      <div className="metadata-content">
        {fields.map(
          (field) =>
            field.value && (
              <div key={field.label} className="metadata-field">
                <strong>{field.label}:</strong>
                <p>{field.value}</p>
              </div>
            )
        )}
        {metadata.session_rating && (
          <div className="metadata-field">
            <strong>Session Rating:</strong>
            <p>
              {ratingLabels[metadata.session_rating]} ({metadata.session_rating}/5)
            </p>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className={`comment-history-modal-overlay ${mounted ? 'show' : ''}`} onClick={onClose}>
      <div className={`comment-history-modal-content ${mounted ? 'show' : ''}`} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Comment Edit History</h2>
          <button className="close-button" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>
        <div className="modal-body">
          {history.length === 0 ? (
            <p className="no-history">No edit history available.</p>
          ) : (
            <div className="history-list">
              {history.map((entry, index) => (
                <div key={entry.id} className="history-entry">
                  <div className="history-header">
                    <div className="history-meta">
                      <span className="version-badge">Version {history.length - index}</span>
                      <span className="editor-info">
                        Edited by <strong>{entry.user?.username || 'Unknown'}</strong>
                      </span>
                      <span className="edit-time">{formatDate(entry.created_at)}</span>
                    </div>
                  </div>
                  <div className="history-content">
                    {entry.metadata ? (
                      renderMetadata(entry.metadata)
                    ) : (
                      <div className="quick-comment">
                        <p>{entry.content}</p>
                      </div>
                    )}
                    {entry.image_url && (
                      <div className="history-image">
                        <img src={entry.image_url} alt="Comment" />
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default CommentHistoryModal;
