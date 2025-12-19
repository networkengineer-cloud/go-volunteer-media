import React, { useState, useEffect } from 'react';
import type { CommentTag, SessionMetadata } from '../api/client';
import './SessionReportForm.css';

interface SessionReportFormProps {
  animalId: number;
  availableTags: CommentTag[];
  onSubmit: (content: string, imageUrl: string, tagIds: number[], metadata?: SessionMetadata) => Promise<void>;
  submitting: boolean;
}

type FormMode = 'quick' | 'structured';

const FIELD_LIMITS = {
  session_goal: 200,
  session_outcome: 2000,
  behavior_notes: 1000,
  medical_notes: 1000,
  other_notes: 1000,
} as const;

const SessionReportForm: React.FC<SessionReportFormProps> = ({
  animalId,
  availableTags,
  onSubmit,
  submitting,
}) => {
  const [mode, setMode] = useState<FormMode>('quick');
  const [commentText, setCommentText] = useState('');
  const [commentImage, setCommentImage] = useState('');
  const [selectedTags, setSelectedTags] = useState<number[]>([]);
  
  // Structured session fields
  const [sessionGoal, setSessionGoal] = useState('');
  const [sessionOutcome, setSessionOutcome] = useState('');
  const [behaviorNotes, setBehaviorNotes] = useState('');
  const [medicalNotes, setMedicalNotes] = useState('');
  const [sessionRating, setSessionRating] = useState<number>(0);
  const [otherNotes, setOtherNotes] = useState('');
  const [draftSaved, setDraftSaved] = useState(false);
  const [showRestoreDraft, setShowRestoreDraft] = useState(false);

  // Draft management helper functions
  const getDraftKey = () => `session_draft_${animalId}`;

  const saveDraft = () => {
    if (mode !== 'structured') return; // Only save drafts for structured mode
    
    const draft = {
      sessionGoal,
      sessionOutcome,
      behaviorNotes,
      medicalNotes,
      sessionRating,
      otherNotes,
      savedAt: new Date().toISOString(),
    };
    
    // Only save if there's actual content
    const hasContent = sessionGoal || sessionOutcome || behaviorNotes || medicalNotes || sessionRating > 0 || otherNotes;
    if (hasContent) {
      localStorage.setItem(getDraftKey(), JSON.stringify(draft));
      setDraftSaved(true);
      setTimeout(() => setDraftSaved(false), 2000); // Hide indicator after 2s
    }
  };

  const restoreDraft = () => {
    const saved = localStorage.getItem(getDraftKey());
    if (!saved) return null;
    
    try {
      return JSON.parse(saved);
    } catch {
      return null;
    }
  };

  const clearDraft = () => {
    localStorage.removeItem(getDraftKey());
  };

  // On mount, check for existing draft
  useEffect(() => {
    const draft = restoreDraft();
    if (draft) {
      setShowRestoreDraft(true);
    }
  }, []);

  // Auto-save draft every 30 seconds
  useEffect(() => {
    if (mode !== 'structured') return;
    
    const interval = setInterval(() => {
      saveDraft();
    }, 30000); // 30 seconds
    
    return () => clearInterval(interval);
  }, [mode, sessionGoal, sessionOutcome, behaviorNotes, medicalNotes, sessionRating, otherNotes]);

  const handleRestoreDraft = () => {
    const draft = restoreDraft();
    if (draft) {
      setMode('structured');
      setSessionGoal(draft.sessionGoal || '');
      setSessionOutcome(draft.sessionOutcome || '');
      setBehaviorNotes(draft.behaviorNotes || '');
      setMedicalNotes(draft.medicalNotes || '');
      setSessionRating(draft.sessionRating || 0);
      setOtherNotes(draft.otherNotes || '');
      setShowRestoreDraft(false);
    }
  };

  const handleDismissDraft = () => {
    clearDraft();
    setShowRestoreDraft(false);
  };

  // Auto-tag logic
  useEffect(() => {
    const behaviorTag = findTagByType(availableTags, 'behavior');
    const medicalTag = findTagByType(availableTags, 'medical');
    
    const newTags = [...selectedTags];
    
    // Auto-select behavior tag if behavior notes have content
    if (behaviorNotes.trim() && behaviorTag && !newTags.includes(behaviorTag.id)) {
      newTags.push(behaviorTag.id);
    } else if (!behaviorNotes.trim() && behaviorTag && newTags.includes(behaviorTag.id)) {
      // Remove behavior tag if field is cleared (but only if user didn't manually select it)
      const index = newTags.indexOf(behaviorTag.id);
      if (index > -1) {
        newTags.splice(index, 1);
      }
    }
    
    // Auto-select medical tag if medical notes have content
    if (medicalNotes.trim() && medicalTag && !newTags.includes(medicalTag.id)) {
      newTags.push(medicalTag.id);
    } else if (!medicalNotes.trim() && medicalTag && newTags.includes(medicalTag.id)) {
      // Remove medical tag if field is cleared
      const index = newTags.indexOf(medicalTag.id);
      if (index > -1) {
        newTags.splice(index, 1);
      }
    }
    
    if (newTags.length !== selectedTags.length || !newTags.every((id, i) => id === selectedTags[i])) {
      setSelectedTags(newTags);
    }
  }, [behaviorNotes, medicalNotes, availableTags]);

  const findTagByType = (tags: CommentTag[], type: 'behavior' | 'medical'): CommentTag | null => {
    // 1. First, try exact name match (case-insensitive)
    const byName = tags.find(t => t.name.toLowerCase() === type.toLowerCase());
    if (byName) return byName;
    
    // 2. Fallback: look for system tags with matching names
    const systemTag = tags.find(t => t.is_system && t.name.toLowerCase().includes(type));
    if (systemTag) return systemTag;
    
    // 3. No matching tag found
    return null;
  };

  const handleModeToggle = () => {
    setMode(mode === 'quick' ? 'structured' : 'quick');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    let content = '';
    let metadata: SessionMetadata | undefined;
    
    if (mode === 'quick') {
      content = commentText.trim();
    } else {
      // Build structured content from all fields
      const parts: string[] = [];
      
      if (sessionGoal.trim()) {
        parts.push(`**Session Goal:** ${sessionGoal.trim()}`);
      }
      if (sessionOutcome.trim()) {
        parts.push(`**Session Outcome:** ${sessionOutcome.trim()}`);
      }
      if (behaviorNotes.trim()) {
        parts.push(`**Behavior:** ${behaviorNotes.trim()}`);
      }
      if (medicalNotes.trim()) {
        parts.push(`**Medical:** ${medicalNotes.trim()}`);
      }
      if (otherNotes.trim()) {
        parts.push(`**Other Notes:** ${otherNotes.trim()}`);
      }
      
      content = parts.join('\n\n');
      
      // Only include metadata if at least one field has content
      if (sessionGoal || sessionOutcome || behaviorNotes || medicalNotes || sessionRating > 0 || otherNotes) {
        metadata = {
          session_goal: sessionGoal.trim() || undefined,
          session_outcome: sessionOutcome.trim() || undefined,
          behavior_notes: behaviorNotes.trim() || undefined,
          medical_notes: medicalNotes.trim() || undefined,
          session_rating: sessionRating > 0 ? sessionRating : undefined,
          other_notes: otherNotes.trim() || undefined,
        };
      }
    }
    
    if (!content && !commentImage) {
      return; // Nothing to submit
    }
    
    await onSubmit(content, commentImage, selectedTags, metadata);
    
    // Clear draft on successful submission
    clearDraft();
    
    // Reset form
    setCommentText('');
    setCommentImage('');
    setSelectedTags([]);
    setSessionGoal('');
    setSessionOutcome('');
    setBehaviorNotes('');
    setMedicalNotes('');
    setSessionRating(0);
    setOtherNotes('');
  };

  const getRatingEmoji = (rating: number): string => {
    const emojis = ['', '😟', '😕', '😐', '🙂', '😄'];
    return emojis[rating] || '';
  };

  const getRatingLabel = (rating: number): string => {
    const labels = ['', 'Poor', 'Fair', 'Okay', 'Good', 'Great'];
    return labels[rating] || '';
  };

  const isFormValid = (): boolean => {
    if (mode === 'quick') {
      return commentText.trim().length > 0 || commentImage.length > 0;
    } else {
      // At least one field should have content for structured mode
      return sessionGoal.trim().length > 0 ||
        sessionOutcome.trim().length > 0 ||
        behaviorNotes.trim().length > 0 ||
        medicalNotes.trim().length > 0 ||
        otherNotes.trim().length > 0 ||
        sessionRating > 0 ||
        commentImage.length > 0;
    }
  };

  return (
    <form onSubmit={handleSubmit} className="session-report-form">
      {/* Draft Restore Prompt */}
      {showRestoreDraft && (
        <div className="draft-restore-prompt" role="alert">
          <div className="draft-prompt-content">
            <span className="draft-icon">📝</span>
            <span className="draft-text">You have an unsaved draft from a previous session.</span>
          </div>
          <div className="draft-prompt-actions">
            <button
              type="button"
              onClick={handleRestoreDraft}
              className="btn-restore-draft"
            >
              Restore Draft
            </button>
            <button
              type="button"
              onClick={handleDismissDraft}
              className="btn-dismiss-draft"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      <div className="form-mode-header">
        <div className="mode-title">
          {mode === 'quick' ? (
            <span className="form-hint">💬 Add a comment</span>
          ) : (
            <>
              <span className="form-hint">📋 Session Report</span>
              {draftSaved && (
                <span className="draft-saved-indicator" role="status" aria-live="polite">
                  ✓ Draft saved
                </span>
              )}
            </>
          )}
        </div>
        <button
          type="button"
          onClick={handleModeToggle}
          className="btn-mode-toggle"
          aria-label={mode === 'quick' ? 'Switch to session report mode' : 'Switch to quick note mode'}
        >
          {mode === 'quick' ? '📋 Session Report' : '← Quick Note'}
        </button>
      </div>

      {mode === 'quick' ? (
        // Quick Note Mode
        <div className="quick-note-mode">
          <textarea
            placeholder="Write your note..."
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            rows={3}
            disabled={submitting}
            aria-label="Comment text"
          />
        </div>
      ) : (
        // Structured Session Report Mode
        <div className="structured-mode">
          {/* Session Goal */}
          <div className="form-field">
            <label htmlFor="session-goal">
              🎯 Session Goal <span className="optional-label">(optional)</span>
            </label>
            <input
              id="session-goal"
              type="text"
              placeholder="e.g., leash training, socialization, enrichment"
              value={sessionGoal}
              onChange={(e) => setSessionGoal(e.target.value)}
              maxLength={FIELD_LIMITS.session_goal}
              disabled={submitting}
              aria-label="Session goal"
            />
            <span className="char-count">
              {sessionGoal.length} / {FIELD_LIMITS.session_goal}
            </span>
          </div>

          {/* Session Outcome */}
          <div className="form-field">
            <label htmlFor="session-outcome">
              📝 Session Outcome <span className="optional-label">(optional)</span>
            </label>
            <textarea
              id="session-outcome"
              placeholder="How did it go? What happened?"
              value={sessionOutcome}
              onChange={(e) => setSessionOutcome(e.target.value)}
              rows={3}
              maxLength={FIELD_LIMITS.session_outcome}
              disabled={submitting}
              aria-label="Session outcome"
            />
            <span className="char-count">
              {sessionOutcome.length} / {FIELD_LIMITS.session_outcome}
            </span>
          </div>

          {/* Concerns (side-by-side on desktop) */}
          <div className="concerns-row">
            {/* Behavior Concerns */}
            <div className="form-field concern-field">
              <label htmlFor="behavior-notes">
                ⚠️ Behavior Concerns <span className="optional-label">(optional)</span>
              </label>
              <textarea
                id="behavior-notes"
                data-testid="behavior-concerns"
                placeholder="Any behavior observations to note?"
                value={behaviorNotes}
                onChange={(e) => setBehaviorNotes(e.target.value)}
                rows={3}
                maxLength={FIELD_LIMITS.behavior_notes}
                disabled={submitting}
                aria-label="Behavior concerns"
              />
              <span className="char-count">
                {behaviorNotes.length} / {FIELD_LIMITS.behavior_notes}
              </span>
              {behaviorNotes.trim() && findTagByType(availableTags, 'behavior') && (
                <span className="auto-tag-indicator" role="status">
                  Auto-tag: 🏷️ behavior
                </span>
              )}
            </div>

            {/* Medical Concerns */}
            <div className="form-field concern-field">
              <label htmlFor="medical-notes">
                🏥 Medical Concerns <span className="optional-label">(optional)</span>
              </label>
              <textarea
                id="medical-notes"
                placeholder="Any health or medical observations?"
                value={medicalNotes}
                onChange={(e) => setMedicalNotes(e.target.value)}
                rows={3}
                maxLength={FIELD_LIMITS.medical_notes}
                disabled={submitting}
                aria-label="Medical concerns"
              />
              <span className="char-count">
                {medicalNotes.length} / {FIELD_LIMITS.medical_notes}
              </span>
              {medicalNotes.trim() && findTagByType(availableTags, 'medical') && (
                <span className="auto-tag-indicator" role="status">
                  Auto-tag: 🏷️ medical
                </span>
              )}
            </div>
          </div>

          {/* Session Rating */}
          <div className="form-field">
            <label>⭐ Session Success <span className="optional-label">(optional)</span></label>
            <div className="rating-selector" role="radiogroup" aria-label="Session rating">
              {[1, 2, 3, 4, 5].map((rating) => (
                <label
                  key={rating}
                  className={`rating-option ${sessionRating === rating ? 'selected' : ''}`}
                >
                  <input
                    type="radio"
                    name="session-rating"
                    value={rating}
                    checked={sessionRating === rating}
                    onChange={() => setSessionRating(rating)}
                    disabled={submitting}
                    aria-label={`${getRatingLabel(rating)} (${rating} out of 5)`}
                  />
                  <span className="rating-emoji">{getRatingEmoji(rating)}</span>
                  <span className="rating-label">{getRatingLabel(rating)}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Other Notes */}
          <div className="form-field">
            <label htmlFor="other-notes">
              💭 Other Comments <span className="optional-label">(optional)</span>
            </label>
            <textarea
              id="other-notes"
              placeholder="Anything else worth noting..."
              value={otherNotes}
              onChange={(e) => setOtherNotes(e.target.value)}
              rows={2}
              maxLength={FIELD_LIMITS.other_notes}
              disabled={submitting}
              aria-label="Other comments"
            />
            <span className="char-count">
              {otherNotes.length} / {FIELD_LIMITS.other_notes}
            </span>
          </div>
        </div>
      )}

      {/* Comment Image Preview (both modes) */}
      {commentImage && (
        <div className="comment-image-preview">
          <img src={commentImage} alt="Preview" />
          <button
            type="button"
            onClick={() => setCommentImage('')}
            className="remove-image"
            aria-label="Remove image"
          >
            ✕
          </button>
        </div>
      )}

      {/* Tags Selection (both modes) */}
      <div className="tags-section">
        <details>
          <summary className="tags-toggle">🏷️ Additional Tags {selectedTags.length > 0 && `(${selectedTags.length})`}</summary>
          <div className="tags-content">
            <div className="tags-grid">
              {availableTags.map((tag) => (
                <label key={tag.id} className="tag-checkbox-label">
                  <input
                    type="checkbox"
                    checked={selectedTags.includes(tag.id)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedTags([...selectedTags, tag.id]);
                      } else {
                        setSelectedTags(selectedTags.filter(id => id !== tag.id));
                      }
                    }}
                    disabled={submitting}
                  />
                  <span className="tag-badge" style={{ backgroundColor: tag.color }}>
                    {tag.name}
                  </span>
                </label>
              ))}
            </div>
          </div>
        </details>
        
        {selectedTags.length > 0 && (
          <div className="selected-tags-preview">
            {selectedTags.map(tagId => {
              const tag = availableTags.find(t => t.id === tagId);
              return tag ? (
                <span key={tagId} className="tag-badge" style={{ backgroundColor: tag.color }}>
                  {tag.name}
                  <button
                    type="button"
                    className="remove-tag"
                    onClick={() => setSelectedTags(selectedTags.filter(t => t !== tagId))}
                    aria-label={`Remove ${tag.name} tag`}
                  >
                    ×
                  </button>
                </span>
              ) : null;
            })}
          </div>
        )}
      </div>

      {/* Submit Button */}
      <div className="form-actions">
        <button
          type="submit"
          className="btn-submit"
          disabled={submitting || !isFormValid()}
          title={!isFormValid() ? 'Add some content first' : mode === 'quick' ? 'Post comment' : 'Post session report'}
        >
          {submitting ? 'Posting...' : mode === 'quick' ? 'Post Comment' : 'Post Session Report'}
        </button>
      </div>
    </form>
  );
};

export default SessionReportForm;
