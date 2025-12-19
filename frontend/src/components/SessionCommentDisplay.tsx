import React from 'react';
import type { AnimalComment } from '../api/client';
import './SessionCommentDisplay.css';

interface SessionCommentDisplayProps {
  comment: AnimalComment;
}

const getRatingEmoji = (rating: number): string => {
  const emojis = ['', '😟', '😕', '😐', '🙂', '😄'];
  return emojis[rating] || '';
};

const getRatingLabel = (rating: number): string => {
  const labels = ['', 'Poor', 'Fair', 'Okay', 'Good', 'Great'];
  return labels[rating] || '';
};

const SessionCommentDisplay: React.FC<SessionCommentDisplayProps> = ({ comment }) => {
  const { metadata } = comment;

  // If no metadata, render as regular comment
  if (!metadata) {
    return <p className="comment-content">{comment.content}</p>;
  }

  // Check if any structured fields have content
  const hasStructuredContent = 
    metadata.session_goal ||
    metadata.session_outcome ||
    metadata.behavior_notes ||
    metadata.medical_notes ||
    metadata.session_rating ||
    metadata.other_notes;

  // If metadata exists but all fields are empty, render as regular comment
  if (!hasStructuredContent) {
    return <p className="comment-content">{comment.content}</p>;
  }

  // Render structured session report
  return (
    <div className="session-report-display">
      {/* Session Goal */}
      {metadata.session_goal && (
        <div className="session-field">
          <span className="session-field-label">🎯 Session Goal:</span>
          {/* Safe to render - content is HTML-escaped on server before storage */}
          <p className="session-field-content">{metadata.session_goal}</p>
        </div>
      )}

      {/* Session Outcome */}
      {metadata.session_outcome && (
        <div className="session-field">
          <span className="session-field-label">📝 Session Outcome:</span>
          {/* Safe to render - content is HTML-escaped on server before storage */}
          <p className="session-field-content">{metadata.session_outcome}</p>
        </div>
      )}

      {/* Concerns Row */}
      {(metadata.behavior_notes || metadata.medical_notes) && (
        <div className="concerns-display">
          {/* Behavior Concerns */}
          {metadata.behavior_notes && (
            <div className="concern-box concern-behavior">
              <div className="concern-header">
                <span className="concern-icon">⚠️</span>
                <span className="concern-title">Behavior</span>
              </div>
              {/* Safe to render - content is HTML-escaped on server before storage */}
              <p className="concern-content">{metadata.behavior_notes}</p>
            </div>
          )}

          {/* Medical Concerns */}
          {metadata.medical_notes && (
            <div className="concern-box concern-medical">
              <div className="concern-header">
                <span className="concern-icon">🏥</span>
                <span className="concern-title">Medical</span>
              </div>
              {/* Safe to render - content is HTML-escaped on server before storage */}
              <p className="concern-content">{metadata.medical_notes}</p>
            </div>
          )}
        </div>
      )}

      {/* Session Rating */}
      {metadata.session_rating && metadata.session_rating > 0 && (
        <div className="session-rating-display">
          <span className="rating-emoji-large">{getRatingEmoji(metadata.session_rating)}</span>
          <span className="rating-text">
            Session: <strong>{getRatingLabel(metadata.session_rating)}</strong>
          </span>
        </div>
      )}

      {/* Other Notes */}
      {metadata.other_notes && (
        <div className="session-field">
          <span className="session-field-label">💭 Other Notes:</span>
          {/* Safe to render - content is HTML-escaped on server before storage */}
          <p className="session-field-content">{metadata.other_notes}</p>
        </div>
      )}
    </div>
  );
};

export default SessionCommentDisplay;
