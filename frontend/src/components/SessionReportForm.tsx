import React, { useState, useEffect, useRef } from 'react';
import type { AnimalComment, CommentTag, SessionMetadata } from '../api/client';
import './SessionReportForm.css';

interface SessionReportFormProps {
  animalId: number;
  availableTags: CommentTag[];
  onSubmit: (content: string, imageUrl: string, tagIds: number[], metadata?: SessionMetadata) => Promise<void>;
  submitting: boolean;
  editingComment?: AnimalComment | null;
  onCancelEdit?: () => void;
}

type FormMode = 'quick' | 'structured';

const FIELD_LIMITS = {
  session_goal: 200,
  session_outcome: 2000,
  behavior_notes: 1000,
  medical_notes: 1000,
  other_notes: 1000,
} as const;

const getTodayDate = (): string => {
  const d = new Date();
  return [d.getFullYear(), String(d.getMonth() + 1).padStart(2, '0'), String(d.getDate()).padStart(2, '0')].join('-');
};

const getCurrentTime = (): string => {
  const now = new Date();
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
};

const SessionReportForm: React.FC<SessionReportFormProps> = ({
  animalId,
  availableTags,
  onSubmit,
  submitting,
  editingComment,
  onCancelEdit,
}) => {
  const isEditMode = !!editingComment;
  const [mode, setMode] = useState<FormMode>('structured');
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
  const [sessionDate, setSessionDate] = useState<string>(getTodayDate);
  const [sessionStartTime, setSessionStartTime] = useState('');
  const [sessionEndTime, setSessionEndTime] = useState('');
  const [draftSaved, setDraftSaved] = useState(false);
  const [showRestoreDraft, setShowRestoreDraft] = useState(false);
  
  // Track which tags were auto-applied vs manually selected
  const [autoAppliedTags, setAutoAppliedTags] = useState<number[]>([]);
  
  // Mobile accordion state
  const [expandedSection, setExpandedSection] = useState<'timing' | 'goals' | 'concerns' | 'rating' | null>('timing');

  // Desktop tab state
  const [desktopTab, setDesktopTab] = useState<'session' | 'escalations'>('session');
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const tabOrder: Array<'session' | 'escalations'> = ['session', 'escalations'];

  const handleTabKeyDown = (e: React.KeyboardEvent, currentIndex: number) => {
    let next: number;
    if (e.key === 'ArrowRight') next = (currentIndex + 1) % tabOrder.length;
    else if (e.key === 'ArrowLeft') next = (currentIndex - 1 + tabOrder.length) % tabOrder.length;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = tabOrder.length - 1;
    else return;
    e.preventDefault();
    setDesktopTab(tabOrder[next]);
    tabRefs.current[next]?.focus();
  };

  // Use refs to access current state in interval callback
  const stateRef = useRef({
    mode,
    sessionGoal,
    sessionOutcome,
    behaviorNotes,
    medicalNotes,
    sessionRating,
    otherNotes,
    sessionDate,
    sessionStartTime,
    sessionEndTime,
  });

  // Update ref whenever state changes
  useEffect(() => {
    stateRef.current = {
      mode,
      sessionGoal,
      sessionOutcome,
      behaviorNotes,
      medicalNotes,
      sessionRating,
      otherNotes,
      sessionDate,
      sessionStartTime,
      sessionEndTime,
    };
  }, [mode, sessionGoal, sessionOutcome, behaviorNotes, medicalNotes, sessionRating, otherNotes, sessionDate, sessionStartTime, sessionEndTime]);

  // Draft management helper functions
  const getDraftKey = () => `session_draft_${animalId}`;

  const saveDraft = () => {
    const state = stateRef.current;
    if (state.mode !== 'structured') return; // Only save drafts for structured mode
    
    const draft = {
      sessionGoal: state.sessionGoal,
      sessionOutcome: state.sessionOutcome,
      behaviorNotes: state.behaviorNotes,
      medicalNotes: state.medicalNotes,
      sessionRating: state.sessionRating,
      otherNotes: state.otherNotes,
      sessionDate: state.sessionDate,
      sessionStartTime: state.sessionStartTime,
      sessionEndTime: state.sessionEndTime,
      savedAt: new Date().toISOString(),
    };
    
    // Only save if there's actual content
    const hasContent = state.sessionGoal || state.sessionOutcome || state.behaviorNotes || 
                       state.medicalNotes || state.sessionRating > 0 || state.otherNotes ||
                       state.sessionStartTime || state.sessionEndTime;
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

  // On mount, check for existing draft (but skip in edit mode)
  useEffect(() => {
    if (isEditMode) return; // Don't restore drafts when editing
    
    const draft = restoreDraft();
    if (draft) {
      setShowRestoreDraft(true);
    }
  }, [isEditMode]);

  // Pre-fill form when editingComment changes
  useEffect(() => {
    if (!editingComment) return;
    
    // Pre-fill tags
    setSelectedTags(editingComment.tags?.map(t => t.id) || []);
    
    // Pre-fill image
    setCommentImage(editingComment.image_url || '');
    
    // Check if this is a structured session report or a quick comment
    if (editingComment.metadata) {
      // Structured session report
      setMode('structured');
      setDesktopTab(editingComment.metadata.behavior_notes || editingComment.metadata.medical_notes ? 'escalations' : 'session');
      setSessionGoal(editingComment.metadata.session_goal || '');
      setSessionOutcome(editingComment.metadata.session_outcome || '');
      setBehaviorNotes(editingComment.metadata.behavior_notes || '');
      setMedicalNotes(editingComment.metadata.medical_notes || '');
      setSessionRating(editingComment.metadata.session_rating || 0);
      setOtherNotes(editingComment.metadata.other_notes || '');
      setSessionDate(editingComment.metadata.session_date || '');
      setSessionStartTime(editingComment.metadata.session_start_time || '');
      setSessionEndTime(editingComment.metadata.session_end_time || '');
      setCommentText('');
    } else {
      // Quick comment
      setMode('quick');
      setCommentText(editingComment.content || '');
      setSessionGoal('');
      setSessionOutcome('');
      setBehaviorNotes('');
      setMedicalNotes('');
      setSessionRating(0);
      setOtherNotes('');
    }
  }, [editingComment]);

  // Auto-save draft every 30 seconds - using callback to access current state (skip in edit mode)
  useEffect(() => {
    if (mode !== 'structured' || isEditMode) return;
    
    const interval = setInterval(() => {
      saveDraft();
    }, 30000); // 30 seconds
    
    return () => clearInterval(interval);
  }, [mode, isEditMode]); // Only depend on mode and isEditMode, saveDraft uses ref for current values

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
      setSessionDate(draft.sessionDate || getTodayDate());
      setSessionStartTime(draft.sessionStartTime || '');
      setSessionEndTime(draft.sessionEndTime || '');
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
    const newAutoAppliedTags = [...autoAppliedTags];
    
    // Auto-select behavior tag if behavior notes have content
    if (behaviorNotes.trim() && behaviorTag && !newTags.includes(behaviorTag.id)) {
      newTags.push(behaviorTag.id);
      newAutoAppliedTags.push(behaviorTag.id);
    } else if (!behaviorNotes.trim() && behaviorTag && newTags.includes(behaviorTag.id)) {
      // Only remove behavior tag if it was auto-applied (not manually selected)
      if (autoAppliedTags.includes(behaviorTag.id)) {
        const index = newTags.indexOf(behaviorTag.id);
        if (index > -1) {
          newTags.splice(index, 1);
        }
        const autoIndex = newAutoAppliedTags.indexOf(behaviorTag.id);
        if (autoIndex > -1) {
          newAutoAppliedTags.splice(autoIndex, 1);
        }
      }
    }
    
    // Auto-select medical tag if medical notes have content
    if (medicalNotes.trim() && medicalTag && !newTags.includes(medicalTag.id)) {
      newTags.push(medicalTag.id);
      newAutoAppliedTags.push(medicalTag.id);
    } else if (!medicalNotes.trim() && medicalTag && newTags.includes(medicalTag.id)) {
      // Only remove medical tag if it was auto-applied (not manually selected)
      if (autoAppliedTags.includes(medicalTag.id)) {
        const index = newTags.indexOf(medicalTag.id);
        if (index > -1) {
          newTags.splice(index, 1);
        }
        const autoIndex = newAutoAppliedTags.indexOf(medicalTag.id);
        if (autoIndex > -1) {
          newAutoAppliedTags.splice(autoIndex, 1);
        }
      }
    }
    
    if (newTags.length !== selectedTags.length || !newTags.every((id, i) => id === selectedTags[i])) {
      setSelectedTags(newTags);
    }
    if (newAutoAppliedTags.length !== autoAppliedTags.length || !newAutoAppliedTags.every((id, i) => id === autoAppliedTags[i])) {
      setAutoAppliedTags(newAutoAppliedTags);
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
      if (sessionGoal || sessionOutcome || behaviorNotes || medicalNotes || sessionRating > 0 || otherNotes || sessionStartTime || sessionEndTime) {
        metadata = {
          session_goal: sessionGoal.trim() || undefined,
          session_outcome: sessionOutcome.trim() || undefined,
          behavior_notes: behaviorNotes.trim() || undefined,
          medical_notes: medicalNotes.trim() || undefined,
          session_rating: sessionRating > 0 ? sessionRating : undefined,
          other_notes: otherNotes.trim() || undefined,
          session_date: sessionDate || undefined,
          session_start_time: sessionStartTime || undefined,
          session_end_time: sessionEndTime || undefined,
        };
      }
    }
    
    if (!content && !commentImage) {
      return; // Nothing to submit
    }
    
    await onSubmit(content, commentImage, selectedTags, metadata);
    
    // Clear draft on successful submission (but only if not editing)
    if (!isEditMode) {
      clearDraft();
    }
    
    // Reset form (but only if not editing - parent will handle state cleanup)
    if (!isEditMode) {
      setCommentText('');
      setCommentImage('');
      setSelectedTags([]);
      setSessionGoal('');
      setSessionOutcome('');
      setBehaviorNotes('');
      setMedicalNotes('');
      setSessionRating(0);
      setOtherNotes('');
      setSessionDate(getTodayDate());
      setSessionStartTime('');
      setSessionEndTime('');
      setDesktopTab('session');
    }
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

  const toggleSection = (section: 'timing' | 'goals' | 'concerns' | 'rating') => {
    setExpandedSection(expandedSection === section ? null : section);
  };

  const getConcernCount = () => {
    let count = 0;
    if (behaviorNotes.trim()) count++;
    if (medicalNotes.trim()) count++;
    return count;
  };

  const getRatingCount = () => {
    let count = 0;
    if (sessionRating > 0) count++;
    if (otherNotes.trim()) count++;
    return count;
  };

  const manuallySelectedTags = selectedTags.filter(id => !autoAppliedTags.includes(id));

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
          {/* Mobile Accordion Sections */}
          <div className="accordion-sections mobile-only">
            {/* Session Timing Section */}
            <div className="accordion-section">
              <button
                type="button"
                className={`accordion-header ${expandedSection === 'timing' ? 'expanded' : ''}`}
                onClick={() => toggleSection('timing')}
                aria-expanded={expandedSection === 'timing'}
                data-testid="accordion-timing"
              >
                <span className="accordion-icon">{expandedSection === 'timing' ? '▼' : '▶'}</span>
                <span className="accordion-title">⏱ Session Timing</span>
              </button>
              {expandedSection === 'timing' && (
                <div className="accordion-content" data-testid="section-timing">
                  <div className="form-field">
                    <label htmlFor="session-date-mobile">
                      📅 Date
                    </label>
                    <input
                      id="session-date-mobile"
                      type="date"
                      value={sessionDate}
                      onChange={(e) => setSessionDate(e.target.value)}
                      disabled={submitting}
                      aria-label="Session date"
                    />
                  </div>
                  <div className="session-time-row">
                    <div className="form-field">
                      <label htmlFor="session-start-time-mobile">
                        Start Time
                      </label>
                      <div className="time-input-row">
                        <input
                          id="session-start-time-mobile"
                          type="time"
                          value={sessionStartTime}
                          onChange={(e) => setSessionStartTime(e.target.value)}
                          disabled={submitting}
                          aria-label="Session start time"
                        />
                        <button
                          type="button"
                          className="btn-now"
                          onClick={() => setSessionStartTime(getCurrentTime())}
                          disabled={submitting}
                          title="Set to current time"
                        >
                          Now
                        </button>
                      </div>
                    </div>
                    <div className="form-field">
                      <label htmlFor="session-end-time-mobile">
                        End Time
                      </label>
                      <div className="time-input-row">
                        <input
                          id="session-end-time-mobile"
                          type="time"
                          value={sessionEndTime}
                          onChange={(e) => setSessionEndTime(e.target.value)}
                          disabled={submitting}
                          aria-label="Session end time"
                        />
                        <button
                          type="button"
                          className="btn-now"
                          onClick={() => setSessionEndTime(getCurrentTime())}
                          disabled={submitting}
                          title="Set to current time"
                        >
                          Now
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Goals & Outcome Section */}
            <div className="accordion-section">
              <button
                type="button"
                className={`accordion-header ${expandedSection === 'goals' ? 'expanded' : ''}`}
                onClick={() => toggleSection('goals')}
                aria-expanded={expandedSection === 'goals'}
                data-testid="accordion-goals"
              >
                <span className="accordion-icon">{expandedSection === 'goals' ? '▼' : '▶'}</span>
                <span className="accordion-title">🎯 Goals & Outcome</span>
              </button>
              {expandedSection === 'goals' && (
                <div className="accordion-content" data-testid="section-goals">
                  {/* Session Goal */}
                  <div className="form-field">
                    <label htmlFor="session-goal-mobile">
                      Session Goal
                    </label>
                    <input
                      id="session-goal-mobile"
                      type="text"
                      placeholder="e.g., leash training"
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
                    <label htmlFor="session-outcome-mobile">
                      Session Outcome
                    </label>
                    <textarea
                      id="session-outcome-mobile"
                      placeholder="How did it go?"
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
                </div>
              )}
            </div>

            {/* Rating & Notes Section */}
            <div className="accordion-section">
              <button
                type="button"
                className={`accordion-header ${expandedSection === 'rating' ? 'expanded' : ''}`}
                onClick={() => toggleSection('rating')}
                aria-expanded={expandedSection === 'rating'}
              >
                <span className="accordion-icon">{expandedSection === 'rating' ? '▼' : '▶'}</span>
                <span className="accordion-title">⭐ Rating & Notes ({getRatingCount()})</span>
              </button>
              {expandedSection === 'rating' && (
                <div className="accordion-content">
                  {/* Session Rating */}
                  <div className="form-field">
                    <label>Session Success</label>
                    <div className="rating-selector" role="radiogroup" aria-label="Session rating">
                      {[1, 2, 3, 4, 5].map((rating) => (
                        <label
                          key={rating}
                          className={`rating-option ${sessionRating === rating ? 'selected' : ''}`}
                        >
                          <input
                            type="radio"
                            name="session-rating-mobile"
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
                    <label htmlFor="other-notes-mobile">
                      Other Comments
                    </label>
                    <textarea
                      id="other-notes-mobile"
                      placeholder="Anything else..."
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
            </div>

            {/* Concerns Section */}
            <div className="accordion-section">
              <button
                type="button"
                className={`accordion-header ${expandedSection === 'concerns' ? 'expanded' : ''}`}
                onClick={() => toggleSection('concerns')}
                aria-expanded={expandedSection === 'concerns'}
                data-testid="accordion-concerns"
              >
                <span className="accordion-icon">{expandedSection === 'concerns' ? '▼' : '▶'}</span>
                <span className="accordion-title">{getConcernCount() > 0 ? '⚠️' : '📋'} Escalations ({getConcernCount()})</span>
              </button>
              {expandedSection === 'concerns' && (
                <div className="accordion-content">
                  <p className="escalations-intro">Only fill these in if there's something worth flagging — leave both blank for routine sessions.</p>
                  {/* Behavior Concerns */}
                  <div className="form-field concern-field">
                    <label htmlFor="behavior-notes-mobile">
                      🐕 Behavior
                      <small className="field-help-text">For notable behavior observations — these can be filtered and shared with the team. Leave blank if nothing to flag.</small>
                    </label>
                    <textarea
                      id="behavior-notes-mobile"
                      data-testid="behavior-concerns"
                      placeholder="Worth flagging to the behavior team?"
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
                    <label htmlFor="medical-notes-mobile">
                      🏥 Medical
                      <small className="field-help-text">For health concerns worth flagging — these can be filtered and shared with staff. Leave blank if nothing to flag.</small>
                    </label>
                    <textarea
                      id="medical-notes-mobile"
                      placeholder="Any health concerns worth noting for staff?"
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
              )}
            </div>
          </div>

          {/* Desktop Non-Accordion Layout */}
          <div className="desktop-only">
            {/* Desktop Tabs */}
            <div className="session-form-tabs" role="tablist">
              <button
                ref={(el) => { tabRefs.current[0] = el; }}
                type="button"
                role="tab"
                id="tab-session"
                aria-selected={desktopTab === 'session'}
                aria-controls="tabpanel-session"
                tabIndex={desktopTab === 'session' ? 0 : -1}
                className={`session-form-tab ${desktopTab === 'session' ? 'session-form-tab--active' : ''}`}
                onClick={() => setDesktopTab('session')}
                onKeyDown={(e) => handleTabKeyDown(e, 0)}
              >
                📝 Session
              </button>
              <button
                ref={(el) => { tabRefs.current[1] = el; }}
                type="button"
                role="tab"
                id="tab-escalations"
                aria-selected={desktopTab === 'escalations'}
                aria-controls="tabpanel-escalations"
                tabIndex={desktopTab === 'escalations' ? 0 : -1}
                className={`session-form-tab ${desktopTab === 'escalations' ? 'session-form-tab--active' : ''}`}
                onClick={() => setDesktopTab('escalations')}
                onKeyDown={(e) => handleTabKeyDown(e, 1)}
              >
                ⚠️ Escalations{getConcernCount() > 0 ? <span aria-hidden="true"> ({getConcernCount()})</span> : ''}
              </button>
            </div>

            {desktopTab === 'session' && <div role="tabpanel" id="tabpanel-session" aria-labelledby="tab-session">
            {/* Session Date + Time */}
            <div className="form-field">
              <label htmlFor="session-date">
                📅 Session Date
              </label>
              <input
                id="session-date"
                type="date"
                value={sessionDate}
                onChange={(e) => setSessionDate(e.target.value)}
                disabled={submitting}
                aria-label="Session date"
              />
            </div>
            <div className="session-time-row">
              <div className="form-field">
                <label htmlFor="session-start-time">
                  ⏱ Start Time
                </label>
                <div className="time-input-row">
                  <input
                    id="session-start-time"
                    type="time"
                    value={sessionStartTime}
                    onChange={(e) => setSessionStartTime(e.target.value)}
                    disabled={submitting}
                    aria-label="Session start time"
                  />
                  <button
                    type="button"
                    className="btn-now"
                    onClick={() => setSessionStartTime(getCurrentTime())}
                    disabled={submitting}
                    title="Set to current time"
                  >
                    Now
                  </button>
                </div>
              </div>
              <div className="form-field">
                <label htmlFor="session-end-time">
                  ⏱ End Time
                </label>
                <div className="time-input-row">
                  <input
                    id="session-end-time"
                    type="time"
                    value={sessionEndTime}
                    onChange={(e) => setSessionEndTime(e.target.value)}
                    disabled={submitting}
                    aria-label="Session end time"
                  />
                  <button
                    type="button"
                    className="btn-now"
                    onClick={() => setSessionEndTime(getCurrentTime())}
                    disabled={submitting}
                    title="Set to current time"
                  >
                    Now
                  </button>
                </div>
              </div>
            </div>
            {/* Session Goal */}
            <div className="form-field">
              <label htmlFor="session-goal">
                🎯 Session Goal
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
              📝 Session Outcome
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

          {/* Session Rating */}
          <div className="form-field">
            <label>⭐ Session Success</label>
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
              💭 Other Comments
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
          </div>}

          {desktopTab === 'escalations' && <div role="tabpanel" id="tabpanel-escalations" aria-labelledby="tab-escalations">
            <p className="escalations-intro">Only fill these in if there's something worth flagging — leave both blank for routine sessions.</p>
            {/* Concerns (side-by-side on desktop) */}
            <div className="concerns-row">
              {/* Behavior Concerns */}
              <div className="form-field concern-field">
                <label htmlFor="behavior-notes">
                  🐕 Behavior Escalation
                  <small className="field-help-text">For notable behavior observations — these can be filtered and shared with the team.</small>
                </label>
                <textarea
                  id="behavior-notes"
                  data-testid="behavior-concerns-desktop"
                  placeholder="Worth flagging to the behavior team?"
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
                  🏥 Medical Escalation
                  <small className="field-help-text">For health concerns worth flagging — these can be filtered and shared with staff.</small>
                </label>
                <textarea
                  id="medical-notes"
                  placeholder="Any health concerns worth noting for staff?"
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
          </div>}
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

      {/* Tags Selection (both modes) - only shown when non-system tags exist */}
      {availableTags.some(t => !t.is_system) && (
        <div className="tags-section">
          <details>
            <summary className="tags-toggle">
              🏷️ Additional Tags {manuallySelectedTags.length > 0 && `(${manuallySelectedTags.length})`}
            </summary>
            <div className="tags-content">
              <div className="tags-grid">
                {availableTags.filter(t => !t.is_system).map((tag) => (
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

          {manuallySelectedTags.length > 0 && (
            <div className="selected-tags-preview">
              {manuallySelectedTags.map(tagId => {
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
      )}

      {/* Submit Button */}
      <div className="form-actions">
        <button
          type="submit"
          className="btn-submit"
          disabled={submitting || !isFormValid()}
          title={!isFormValid() ? 'Add some content first' : isEditMode ? 'Save changes' : mode === 'quick' ? 'Post comment' : 'Post session report'}
        >
          {submitting ? (isEditMode ? 'Saving...' : 'Posting...') : isEditMode ? 'Save Changes' : mode === 'quick' ? 'Post Comment' : 'Post Session Report'}
        </button>
        {isEditMode && onCancelEdit && (
          <button
            type="button"
            className="btn-cancel"
            onClick={onCancelEdit}
            disabled={submitting}
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
};

export default SessionReportForm;
