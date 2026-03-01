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

const formatTime12h = (timeStr: string): string => {
  const [h, m] = timeStr.split(':').map(Number);
  const period = h >= 12 ? 'PM' : 'AM';
  const hour12 = h % 12 || 12;
  return `${hour12}:${m.toString().padStart(2, '0')} ${period}`;
};

const formatSessionDate = (dateStr: string): string => {
  const [year, month, day] = dateStr.split('-').map(Number);
  return new Date(year, month - 1, day).toLocaleDateString('en-US', {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  });
};

const calculateDuration = (start: string, end: string): string => {
  const [startH, startM] = start.split(':').map(Number);
  const [endH, endM] = end.split(':').map(Number);
  const startTotal = startH * 60 + startM;
  const endTotal = endH * 60 + endM;
  let diff = endTotal - startTotal;
  if (diff < 0) diff += 24 * 60; // handle midnight crossing
  if (diff === 0) return '';
  const hours = Math.floor(diff / 60);
  const minutes = diff % 60;
  if (hours > 0 && minutes > 0) return `${hours}h ${minutes}min`;
  if (hours > 0) return `${hours}h`;
  return `${minutes}min`;
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
    metadata.other_notes ||
    metadata.session_start_time ||
    metadata.session_end_time;

  // If metadata exists but all fields are empty, render as regular comment
  if (!hasStructuredContent) {
    return <p className="comment-content">{comment.content}</p>;
  }

  // Render structured session report
  return (
    <div className="session-report-display">
      {/* Session Date + Time */}
      {(metadata.session_date || metadata.session_start_time || metadata.session_end_time) && (
        <div className="session-field session-time-display">
          <span className="session-field-label">📅</span>
          <span className="session-field-content">
            {metadata.session_date && (
              <span className="session-date-text">{formatSessionDate(metadata.session_date)}</span>
            )}
            {metadata.session_date && (metadata.session_start_time || metadata.session_end_time) && (
              <span className="session-time-sep"> · </span>
            )}
            {metadata.session_start_time && metadata.session_end_time ? (
              <>
                {formatTime12h(metadata.session_start_time)} – {formatTime12h(metadata.session_end_time)}
                {(() => {
                  const duration = calculateDuration(metadata.session_start_time!, metadata.session_end_time!);
                  return duration ? <span className="session-duration"> ({duration})</span> : null;
                })()}
              </>
            ) : metadata.session_start_time ? (
              <>Started: {formatTime12h(metadata.session_start_time)}</>
            ) : metadata.session_end_time ? (
              <>Ended: {formatTime12h(metadata.session_end_time)}</>
            ) : null}
          </span>
        </div>
      )}

      {/* Session Goal */}
      {metadata.session_goal && (
        <div className="session-field">
          <span className="session-field-label">🎯 Session Goal:</span>
          <p className="session-field-content">{metadata.session_goal}</p>
        </div>
      )}

      {/* Session Outcome */}
      {metadata.session_outcome && (
        <div className="session-field">
          <span className="session-field-label">📝 Session Outcome:</span>
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
          <p className="session-field-content">{metadata.other_notes}</p>
        </div>
      )}
    </div>
  );
};

export default SessionCommentDisplay;
