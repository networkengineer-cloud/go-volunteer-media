import React, { useEffect, useState, useRef, useCallback, Suspense, lazy } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, commentTagsApi, groupsApi, scriptsApi } from '../api/client';
import type { Animal, AnimalComment, CommentTag, CommentHistory, Group, GroupMembership, SessionMetadata, Script } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { calculateQuarantineEndDate, calculateAge, formatAge } from '../utils/dateUtils';
import { formatDisplayName } from '../utils/formatName';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import ConfirmDialog from '../components/ConfirmDialog';
import { useConfirmDialog } from '../hooks/useConfirmDialog';
import ProtocolViewerErrorBoundary from '../components/ProtocolViewerErrorBoundary';
import SessionReportForm from '../components/SessionReportForm';
import SessionCommentDisplay from '../components/SessionCommentDisplay';
import CommentHistoryModal from '../components/CommentHistoryModal';
import './AnimalDetailPage.css';

// Lazy load ProtocolViewer to reduce initial bundle size (~350KB savings)
const ProtocolViewer = lazy(() => import('../components/ProtocolViewer'));

const AnimalDetailPage: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const { isAdmin, user } = useAuth();
  const toast = useToast();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [group, setGroup] = useState<Group | null>(null);
  const [membership, setMembership] = useState<GroupMembership | null>(null);
  const [comments, setComments] = useState<AnimalComment[]>([]);
  const [deletedComments, setDeletedComments] = useState<AnimalComment[]>([]);
  const [showDeleted, setShowDeleted] = useState(false);
  const [availableTags, setAvailableTags] = useState<CommentTag[]>([]);
  const [filterTags, setFilterTags] = useState<string[]>([]); // Tags selected for filtering
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [showCommentForm, setShowCommentForm] = useState(true);
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null);
  const [error, setError] = useState<string>('');
  const [commentHistory, setCommentHistory] = useState<CommentHistory[]>([]);
  const [showHistoryModal, setShowHistoryModal] = useState(false);
  
  // Pagination state
  const [totalComments, setTotalComments] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc'); // Default: newest first
  const [showScrollToTop, setShowScrollToTop] = useState(false);
  const { confirmDialog, openConfirmDialog, closeConfirmDialog } = useConfirmDialog();
  const [showProtocolModal, setShowProtocolModal] = useState(false);
  const [viewingScript, setViewingScript] = useState<Script | null>(null);
  const [manageScriptsOpen, setManageScriptsOpen] = useState(false);
  const [groupScripts, setGroupScripts] = useState<Script[]>([]);
  const [savingScripts, setSavingScripts] = useState(false);
  const [selectedScriptIds, setSelectedScriptIds] = useState<Set<number>>(new Set());
  const [trainerNotesExpanded, setTrainerNotesExpanded] = useState(false);
  const commentsTopRef = useRef<HTMLDivElement>(null);
  const COMMENTS_PER_PAGE = 10;

  const loadComments = useCallback(async (
    gId: number, 
    animalId: number, 
    filter: string, 
    offset: number = 0,
    order: 'asc' | 'desc' = 'desc',
    append: boolean = false
  ) => {
    try {
      const commentsRes = await animalCommentsApi.getAll(gId, animalId, {
        tagFilter: filter || undefined,
        limit: COMMENTS_PER_PAGE,
        offset,
        order
      });
      
      if (append) {
        setComments(prev => [...prev, ...commentsRes.data.comments]);
      } else {
        setComments(commentsRes.data.comments);
      }
      
      setTotalComments(commentsRes.data.total);
      setHasMore(commentsRes.data.hasMore);
    } catch (error) {
      console.error('Failed to load comments:', error);
      toast.showError('Failed to load comments. Please try again.');
    }
  }, [toast, COMMENTS_PER_PAGE]);

  const loadTags = useCallback(async (gId: number) => {
    try {
      const res = await commentTagsApi.getAll(gId);
      setAvailableTags(res.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
    }
  }, []);

  const loadDeletedComments = useCallback(async (gId: number) => {
    if (!isAdmin) return;
    try {
      const res = await animalCommentsApi.getDeleted(gId);
      console.log('Deleted comments loaded:', res.data);
      setDeletedComments(res.data || []);
    } catch (error) {
      console.error('Failed to load deleted comments:', error);
      setDeletedComments([]);
    }
  }, [isAdmin]);

  const handleDeleteComment = (commentId: number) => {
    if (!groupId || !id) return;
    openConfirmDialog(
      'Delete Comment',
      'Are you sure you want to delete this comment?',
      async () => {
        try {
          await animalCommentsApi.delete(Number(groupId), Number(id), commentId);
          toast.showSuccess('Comment deleted successfully');
          await loadComments(Number(groupId), Number(id), filterTags.join(','), 0, sortOrder, false);
          if (isAdmin) {
            await loadDeletedComments(Number(groupId));
          }
        } catch (error) {
          console.error('Failed to delete comment:', error);
          toast.showError('Failed to delete comment. Please try again.');
        }
      },
    );
  };

  const loadGroupData = useCallback(async (gId: number) => {
    try {
      const [groupRes, membershipRes] = await Promise.all([
        groupsApi.getById(gId),
        groupsApi.getMembership(gId)
      ]);
      setGroup(groupRes.data);
      setMembership(membershipRes.data);
    } catch (error) {
      console.error('Failed to load group data:', error);
    }
  }, []);

  const loadAnimalData = useCallback(async (gId: number, animalId: number) => {
    try {
      setError('');
      const animalRes = await animalsApi.getById(gId, animalId);
      setAnimal(animalRes.data);
      await loadComments(gId, animalId, '', 0, sortOrder);
    } catch (error: unknown) {
      console.error('Failed to load animal data:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to load animal information. Please try again.';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [loadComments, sortOrder]);

  useEffect(() => {
    if (groupId && id) {
      loadAnimalData(Number(groupId), Number(id));
      loadGroupData(Number(groupId));
      loadTags(Number(groupId));
      if (isAdmin) {
        loadDeletedComments(Number(groupId));
      }
    }
  }, [groupId, id, loadAnimalData, loadGroupData, loadTags, isAdmin, loadDeletedComments]);

  useEffect(() => {
    if (groupId && id) {
      const tagFilterString = filterTags.join(',');
      loadComments(Number(groupId), Number(id), tagFilterString, 0, sortOrder);
    }
  }, [filterTags, groupId, id, loadComments, sortOrder]);

  useEffect(() => {
    const handleScroll = () => {
      setShowScrollToTop(window.scrollY > 500);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const handleSubmitComment = async (content: string, imageUrl: string, tagIds: number[], metadata?: SessionMetadata) => {
    if (!content.trim() && !imageUrl) {
      toast.showWarning('Please write a comment');
      return;
    }

    setSubmitting(true);
    try {
      const newComment = await animalCommentsApi.create(
        Number(groupId),
        Number(id),
        content,
        imageUrl,
        tagIds.length > 0 ? tagIds : undefined,
        metadata
      );
      
      // Add new comment to the beginning or end based on sort order
      if (sortOrder === 'desc') {
        setComments([newComment.data, ...comments]);
      } else {
        setComments([...comments, newComment.data]);
      }
      
      setTotalComments(prev => prev + 1);
      toast.showSuccess('Comment posted successfully!');
    } catch (error: unknown) {
      console.error('Failed to post comment:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to post comment. Please try again.';
      toast.showError(errorMsg);
      throw error; // Re-throw so SessionReportForm doesn't reset on error
    } finally {
      setSubmitting(false);
    }
  };

  const handleEditComment = (commentId: number) => {
    setEditingCommentId(commentId);
    setShowCommentForm(false); // Hide the new comment form when editing
  };

  const handleUpdateComment = async (content: string, imageUrl: string, tagIds: number[], metadata?: SessionMetadata) => {
    if (!editingCommentId) return;
    
    if (!content.trim() && !imageUrl) {
      toast.showWarning('Please write a comment');
      return;
    }

    setSubmitting(true);
    try {
      const updatedComment = await animalCommentsApi.update(
        Number(groupId),
        Number(id),
        editingCommentId,
        content,
        imageUrl,
        tagIds.length > 0 ? tagIds : undefined,
        metadata
      );
      
      // Update the comment in the local state
      setComments(comments.map(c => c.id === editingCommentId ? updatedComment.data : c));
      
      // Exit edit mode
      setEditingCommentId(null);
      setShowCommentForm(true);
      
      toast.showSuccess('Comment updated successfully!');
    } catch (error: unknown) {
      console.error('Failed to update comment:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to update comment. Please try again.';
      toast.showError(errorMsg);
      throw error;
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelEdit = () => {
    setEditingCommentId(null);
    setShowCommentForm(true);
  };

  const handleViewHistory = async (commentId: number) => {
    try {
      const response = await animalCommentsApi.getHistory(Number(groupId), Number(id), commentId);
      setCommentHistory(response.data);
      setShowHistoryModal(true);
    } catch (error: unknown) {
      console.error('Failed to load comment history:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to load comment history';
      toast.showError(errorMsg);
    }
  };

  const handleEdit = () => {
    navigate(`/groups/${groupId}/animals/${id}`);
  };

  const handleLoadMore = async () => {
    if (!groupId || !id || loadingMore || !hasMore) return;
    
    setLoadingMore(true);
    try {
      const tagFilterString = filterTags.join(',');
      await loadComments(
        Number(groupId), 
        Number(id), 
        tagFilterString, 
        comments.length,
        sortOrder,
        true // append
      );
    } finally {
      setLoadingMore(false);
    }
  };

  const handleSortChange = () => {
    const newOrder = sortOrder === 'desc' ? 'asc' : 'desc';
    setSortOrder(newOrder);
  };

  const getCommentDateRange = () => {
    if (comments.length === 0) return null;
    const dates = comments.map(c => new Date(c.created_at));
    const oldest = new Date(Math.min(...dates.map(d => d.getTime())));
    const newest = new Date(Math.max(...dates.map(d => d.getTime())));
    return { oldest, newest };
  };

  const scrollToComments = (position: 'top' | 'bottom') => {
    if (position === 'top' && commentsTopRef.current) {
      commentsTopRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    } else {
      window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' });
    }
  };

  const handleExportComments = async () => {
    try {
      const tagFilterString = filterTags.length > 0 ? filterTags.join(',') : undefined;
      const response = await animalsApi.exportCommentsCSV(
        Number(groupId),
        Number(id),
        tagFilterString
      );
      
      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      const filterSuffix = filterTags.length > 0 ? `-filtered-${filterTags.join('-')}` : '';
      link.setAttribute('download', `${animal?.name || 'animal'}-comments${filterSuffix}.csv`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Comments exported successfully!');
    } catch (error) {
      console.error('Failed to export comments CSV:', error);
      toast.showError('Failed to export comments. Please try again.');
    }
  };

  const handleOpenProtocolDocument = () => {
    if (!animal?.protocol_document_url) return;
    setShowProtocolModal(true);
  };

  const handleCloseProtocolModal = () => {
    setShowProtocolModal(false);
  };

  if (loading) {
    return (
      <div className="animal-detail-page">
        <div className="animal-detail-container">
          <SkeletonLoader variant="text" width="150px" />
          <div className="animal-header">
            <div className="animal-main-info">
              <SkeletonLoader variant="rectangular" height="400px" />
              <div className="animal-details">
                <SkeletonLoader variant="text" width="60%" height="2rem" />
                <SkeletonLoader variant="text" width="40%" />
                <SkeletonLoader variant="text" width="80%" />
              </div>
            </div>
          </div>
          <div className="comments-section">
            <SkeletonLoader variant="text" width="200px" height="1.5rem" />
            <div className="skeleton-list">
              <SkeletonLoader variant="comment" count={3} />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !animal) {
    return (
      <div className="animal-detail-page">
        <div className="animal-detail-container">
          <ErrorState
            title={error ? 'Unable to Load Animal' : 'Animal Not Found'}
            message={error || 'The animal you\'re looking for doesn\'t exist or you don\'t have permission to view it.'}
            onRetry={error ? () => {
              setLoading(true);
              loadAnimalData(Number(groupId), Number(id));
            } : undefined}
            onGoBack={() => navigate(`/groups/${groupId}`)}
            goBackLabel="Back to Group"
          />
        </div>
      </div>
    );
  }

  const { years: animalAgeYears, months: animalAgeMonths } = calculateAge(animal.estimated_birth_date, animal.age);
  const animalAgeLabel = formatAge(animalAgeYears, animalAgeMonths);

  return (
    <div className="animal-detail-page">
      <div className="animal-detail-container">
        {/* Breadcrumb Navigation */}
        <nav className="breadcrumb" aria-label="Breadcrumb">
          <ol className="breadcrumb-list">
            <li className="breadcrumb-item">
              <Link to="/dashboard" className="breadcrumb-link">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
                  <polyline points="9 22 9 12 15 12 15 22" />
                </svg>
                <span className="sr-only">Home</span>
              </Link>
            </li>
            <li className="breadcrumb-separator" aria-hidden="true">›</li>
            <li className="breadcrumb-item">
              <Link to={`/groups/${groupId}`} className="breadcrumb-link">
                {group?.name || 'Group'}
              </Link>
            </li>
            <li className="breadcrumb-separator" aria-hidden="true">›</li>
            <li className="breadcrumb-item">
              <Link to={`/groups/${groupId}?view=animals`} className="breadcrumb-link">
                Animals
              </Link>
            </li>
            <li className="breadcrumb-separator" aria-hidden="true">›</li>
            <li className="breadcrumb-item">
              <span className="breadcrumb-current" aria-current="page">
                {animal.name}
              </span>
            </li>
          </ol>
        </nav>

        <div className="animal-header">
          <div className="animal-main-info">
            {animal.image_url && (
              <img
                src={animal.image_url}
                alt={animal.name}
                className="animal-main-image"
              />
            )}
            <div className="animal-details">
              <h1>{animal.name}</h1>
              <p className="animal-meta">
                {animal.breed && `${animal.breed} \u2022 `}{animalAgeLabel} {'\u2022'} ID: {animal.id}
              </p>
              <div className="status-badges">
                <span className={`status ${animal.status}`}>{animal.status}</span>
                {animal.return_count > 0 && (
                  <span 
                    className="badge badge-returning" 
                    title={`This animal has returned to the shelter ${animal.return_count} time${animal.return_count > 1 ? 's' : ''} after being previously archived`}
                    aria-label={`Returning animal - ${animal.return_count} return${animal.return_count > 1 ? 's' : ''}`}
                  >
                    ↩ Returned {animal.return_count}×
                  </span>
                )}
              </div>
              {animal.status === 'bite_quarantine' && animal.quarantine_start_date && (
                <div className="quarantine-info">
                  <p className="quarantine-notice">
                    <strong>⚠️ Bite Quarantine</strong>
                  </p>
                  <p className="quarantine-dates">
                    Start: {new Date(animal.quarantine_start_date).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}
                  </p>
                  <p className="quarantine-dates">
                    End: {calculateQuarantineEndDate(animal.quarantine_start_date, 'long')}
                  </p>
                </div>
              )}
              {animal.description && (
                <p className="animal-description">{animal.description}</p>
              )}
              
              {/* Animal Tags Section */}
              {animal.tags && animal.tags.length > 0 && (
                <div className="animal-tags-section">
                  <h3>Tags</h3>
                  <div className="animal-tags-list">
                    {animal.tags.map(tag => (
                      <span
                        key={tag.id}
                        className="animal-tag"
                        style={{ backgroundColor: tag.color }}
                        title={tag.category}
                      >
                        {tag.name}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Name History Section */}
              {animal.name_history && animal.name_history.length > 0 && (
                <div className="name-history-section">
                  <h3>Previous Names</h3>
                  <div className="name-history-list">
                    {animal.name_history.map((history) => (
                      <div key={history.id} className="name-history-item">
                        <span className="name-history-old">{history.old_name}</span>
                        <span className="name-history-arrow">→</span>
                        <span className="name-history-new">{history.new_name}</span>
                        <span className="name-history-date">
                          {new Date(history.created_at).toLocaleDateString('en-US', { 
                            month: 'short', 
                            day: 'numeric', 
                            year: 'numeric' 
                          })}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Protocol Document Section */}
              {animal.protocol_document_url && animal.protocol_document_name && (
                <div className="protocol-document-section">
                  <h3>📋 Protocol Document</h3>
                  <div className="protocol-document-card">
                    <span className="protocol-document-icon">
                      {animal.protocol_document_name.endsWith('.pdf') ? '📄' : '📝'}
                    </span>
                    <div className="protocol-document-details">
                      <span className="protocol-document-name">{animal.protocol_document_name}</span>
                      {animal.protocol_document_size && (
                        <span className="protocol-document-size">
                          {(animal.protocol_document_size / 1024 / 1024).toFixed(2)} MB
                        </span>
                      )}
                    </div>
                    <button
                      onClick={handleOpenProtocolDocument}
                      className="btn-view-document"
                      title="View protocol document"
                      aria-label="View protocol document"
                    >
                      View Protocol →
                    </button>
                  </div>
                </div>
              )}

              {/* Protocol Scripts Section */}
              {group?.has_protocols && (
                <div className="protocol-document-section">
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
                    <h3 style={{ margin: 0 }}>📄 Scripts</h3>
                    {(isAdmin || membership?.is_group_admin || membership?.is_site_admin) && (
                      <button
                        className="btn-view-document"
                        onClick={async () => {
                          try {
                            const res = await scriptsApi.getAll(Number(groupId));
                            setGroupScripts(res.data);
                            setSelectedScriptIds(new Set((animal.scripts || []).map((s) => s.id)));
                            setManageScriptsOpen(true);
                          } catch {
                            toast.showError('Failed to load group scripts');
                          }
                        }}
                        aria-label="Manage scripts for this animal"
                      >
                        Manage Scripts
                      </button>
                    )}
                  </div>
                  {(animal.scripts || []).length === 0 ? (
                    <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: 0 }}>No scripts assigned to this animal yet.</p>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                      {(animal.scripts || []).map((script) => (
                        <div key={script.id} className="protocol-document-card">
                          <span className="protocol-document-icon">📄</span>
                          <div className="protocol-document-details">
                            <span className="protocol-document-name">{script.title}</span>
                            {script.description && (
                              <span className="protocol-document-size">{script.description}</span>
                            )}
                          </div>
                          <button
                            className="btn-view-document"
                            onClick={() => setViewingScript(script)}
                            aria-label={`View ${script.title}`}
                          >
                            View →
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Trainer Notes Section (collapsed by default) */}
              {animal.trainer_notes && (
                <div className="trainer-notes-section">
                  <button
                    className={`trainer-notes-toggle ${trainerNotesExpanded ? 'expanded' : ''}`}
                    onClick={() => setTrainerNotesExpanded(!trainerNotesExpanded)}
                    aria-expanded={trainerNotesExpanded}
                    aria-controls="trainer-notes-content"
                  >
                    <span className="trainer-notes-icon">{trainerNotesExpanded ? '▼' : '▶'}</span>
                    <h3>🐾 Meet With Trainer Notes</h3>
                  </button>
                  {trainerNotesExpanded && (
                    <div id="trainer-notes-content" className="trainer-notes-content">
                      <p>{animal.trainer_notes}</p>
                    </div>
                  )}
                </div>
              )}
              
              <div className="animal-action-buttons">
                <Link 
                  to={`/groups/${groupId}/animals/${id}/photos`} 
                  className="btn-view-gallery-large"
                  title="View and manage animal photos"
                >
                  📷 Photo Gallery
                </Link>
                
                {(isAdmin || membership?.is_group_admin) && (
                  <button onClick={handleEdit} className="btn-edit">
                    Edit Animal Details
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        <div className="comments-section" ref={commentsTopRef}>
          <div className="comments-header">
            <div className="comments-header-title">
              <h2>Comments</h2>
              {totalComments > 20 && getCommentDateRange() && (
                <span className="date-range-info">
                  Showing activity from {getCommentDateRange()?.oldest.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })} to {getCommentDateRange()?.newest.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}
                </span>
              )}
            </div>
            <div className="comments-header-actions">
              {isAdmin && (
                <button
                  onClick={handleExportComments}
                  className="btn-export"
                  title="Export comments to CSV"
                >
                  📥 Export Comments
                </button>
              )}
              <button
                onClick={() => setShowCommentForm(!showCommentForm)}
                className="btn-toggle-form"
              >
                {showCommentForm ? '− Hide Comment Form' : '+ Add Comment'}
              </button>
              {comments.filter(c => c.image_url).length > 0 && (
                <Link to={`/groups/${groupId}/animals/${id}/photos`} className="btn-view-gallery">
                  📷 View All Photos ({comments.filter(c => c.image_url).length})
                </Link>
              )}
            </div>
          </div>

          {showCommentForm && (
            <SessionReportForm
              animalId={Number(id)}
              availableTags={availableTags}
              onSubmit={handleSubmitComment}
              submitting={submitting}
            />
          )}

          {/* Admin: Show Deleted Comments Toggle - HIGH VISIBILITY */}
          {isAdmin && deletedComments && deletedComments.length > 0 && (
            <div style={{
              background: 'linear-gradient(135deg, rgba(239, 68, 68, 0.08) 0%, rgba(239, 68, 68, 0.04) 100%)',
              border: '2px solid #ef4444',
              padding: '1rem',
              borderRadius: '8px',
              marginBottom: '1.5rem',
              marginTop: '1rem'
            }}>
              <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', gap: '0.75rem' }}>
                <input
                  id="show-deleted-comments"
                  type="checkbox"
                  checked={showDeleted}
                  onChange={() => setShowDeleted(!showDeleted)}
                  style={{ width: '20px', height: '20px', cursor: 'pointer', accentColor: '#ef4444' }}
                />
                <span style={{ fontWeight: 700, color: '#ef4444', fontSize: '1.05rem' }}>
                  🗑️ Show Deleted Comments ({deletedComments.length})
                </span>
              </label>
            </div>
          )}

          {/* Deleted Comments Section */}
          {showDeleted && deletedComments && deletedComments.length > 0 && (
            <div className="deleted-comments-section">
              <h3>🗑️ Deleted Comments (Admin Review)</h3>
              {deletedComments.filter(dc => dc.animal_id === Number(id)).map((comment) => (
                <div key={comment.id} className="comment-card deleted-comment">
                  <div className="deleted-badge">🗑️ DELETED</div>
                  <div className="comment-header">
                    {comment.user?.id ? (
                      <Link to={`/users/${comment.user.id}/profile`} className="comment-author comment-author--link">
                        {formatDisplayName(comment.user)}
                      </Link>
                    ) : (
                      <span className="comment-author">{comment.user ? formatDisplayName(comment.user) : ''}</span>
                    )}
                    <span className="comment-date">
                      {new Date(comment.created_at).toLocaleDateString()} at{' '}
                      {new Date(comment.created_at).toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </span>
                  </div>
                  {comment.tags && comment.tags.length > 0 && (
                    <div className="comment-tags">
                      {comment.tags.map((tag) => (
                        <span key={tag.id} className="tag-badge" style={{ backgroundColor: tag.color }}>
                          {tag.name}
                        </span>
                      ))}
                    </div>
                  )}
                  <SessionCommentDisplay comment={comment} />
                  {comment.image_url && (
                    <img
                      src={comment.image_url}
                      alt="Comment"
                      className="comment-image"
                    />
                  )}
                  <div className="comment-meta">
                    Deleted: {new Date(comment.deleted_at!).toLocaleDateString()}
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="comments-controls">
            <div className="comments-count-and-sort">
              <span className="comments-count" aria-live="polite">
                {totalComments > 50 ? (
                  <>
                    <strong>{comments.length}</strong> of <strong>{totalComments}</strong> comments loaded
                    {hasMore && <span className="load-more-hint"> • Scroll down to load more</span>}
                  </>
                ) : (
                  <>Showing {comments.length} of {totalComments} comment{totalComments !== 1 ? 's' : ''}</>
                )}
              </span>
              <button
                onClick={handleSortChange}
                className="btn-sort"
                aria-label={`Sort comments ${sortOrder === 'desc' ? 'oldest first' : 'newest first'}`}
                title={`Currently showing ${sortOrder === 'desc' ? 'newest' : 'oldest'} first. Click to reverse`}
              >
                {sortOrder === 'desc' ? '↓ Newest First' : '↑ Oldest First'}
              </button>
            </div>

            <div className="tag-filters">
              <button
                className={filterTags.length === 0 ? 'tag-filter active' : 'tag-filter'}
                onClick={() => setFilterTags([])}
                aria-label="Show all comments"
              >
                All {filterTags.length === 0 && `(${totalComments})`}
              </button>
              {availableTags.map((tag) => {
                const isActive = filterTags.includes(tag.name);
                return (
                  <button
                    key={tag.id}
                    className={isActive ? 'tag-filter active' : 'tag-filter'}
                    style={{ 
                      borderColor: tag.color,
                      backgroundColor: isActive ? tag.color : 'transparent',
                      color: isActive ? 'white' : 'inherit'
                    }}
                    onClick={() => {
                      if (isActive) {
                        // Remove tag from filter
                        setFilterTags(filterTags.filter(t => t !== tag.name));
                      } else {
                        // Add tag to filter
                        setFilterTags([...filterTags, tag.name]);
                      }
                    }}
                    aria-label={`Filter by ${tag.name}`}
                    aria-pressed={isActive}
                  >
                    {tag.name}
                  </button>
                );
              })}
              {filterTags.length > 0 && (
                <span className="filter-hint">
                  Showing comments with: {filterTags.join(' OR ')}
                </span>
              )}
            </div>
          </div>

          <div className="comments-list">
            {comments.length === 0 ? (
              <EmptyState
                icon={
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                    <line x1="9" y1="10" x2="15" y2="10" />
                    <line x1="9" y1="14" x2="13" y2="14" />
                  </svg>
                }
                title={`No updates yet for ${animal.name}`}
                description={`${animal.name} is waiting for their first comment! Share how your session went, upload a photo, or leave care notes for other volunteers.`}
                primaryAction={{
                  label: 'Add First Comment',
                  onClick: () => setShowCommentForm(true)
                }}
              />
            ) : (
              <>
                {comments.map((comment) => {
                  const isEditing = editingCommentId === comment.id;
                  const isCommentOwner = comment.user?.id === user?.id;
                  
                  return (
                    <div key={comment.id} className={`comment-card ${isEditing ? 'editing' : ''}`}>
                      {isEditing ? (
                        // Inline editing form
                        <SessionReportForm
                          animalId={Number(id)}
                          availableTags={availableTags}
                          onSubmit={handleUpdateComment}
                          submitting={submitting}
                          editingComment={comment}
                          onCancelEdit={handleCancelEdit}
                        />
                      ) : (
                        <>
                          <div className="comment-header">
                            {comment.user?.id ? (
                              <Link to={`/users/${comment.user.id}/profile`} className="comment-author comment-author--link">
                                {formatDisplayName(comment.user)}
                              </Link>
                            ) : (
                              <span className="comment-author">{comment.user ? formatDisplayName(comment.user) : ''}</span>
                            )}
                            <span className="comment-date">
                              {new Date(comment.created_at).toLocaleDateString()} at{' '}
                              {new Date(comment.created_at).toLocaleTimeString([], {
                                hour: '2-digit',
                                minute: '2-digit',
                              })}
                              {comment.is_edited && (
                                <span className="edited-badge" title={`Last edited: ${new Date(comment.updated_at).toLocaleString()}`}>
                                  (edited)
                                </span>
                              )}
                            </span>
                            <div className="comment-actions">
                              {(isAdmin || membership?.is_group_admin) && new Date(comment.created_at).getTime() !== new Date(comment.updated_at).getTime() && (
                                <button
                                  className="btn-view-history"
                                  onClick={() => handleViewHistory(comment.id)}
                                  title="View edit history"
                                  aria-label="View edit history"
                                >
                                  📜
                                </button>
                              )}
                              {isCommentOwner && (
                                <button
                                  className="btn-edit-comment"
                                  onClick={() => handleEditComment(comment.id)}
                                  title="Edit comment"
                                  aria-label="Edit comment"
                                >
                                  ✏️
                                </button>
                              )}
                              {(isAdmin || isCommentOwner) && (
                                <button
                                  className="btn-delete-comment"
                                  onClick={() => handleDeleteComment(comment.id)}
                                  title="Delete comment"
                                  aria-label="Delete comment"
                                >
                                  🗑️
                                </button>
                              )}
                            </div>
                          </div>
                          {comment.tags && comment.tags.length > 0 && (
                            <div className="comment-tags">
                              {comment.tags.map((tag) => (
                                <span key={tag.id} className="tag-badge" style={{ backgroundColor: tag.color }}>
                                  {tag.name}
                                </span>
                              ))}
                            </div>
                          )}
                          <SessionCommentDisplay comment={comment} />
                          {comment.image_url && (
                            <img
                              src={comment.image_url}
                              alt="Comment"
                              className="comment-image"
                            />
                          )}
                        </>
                      )}
                    </div>
                  );
                })}

                {hasMore && (
                  <div className="load-more-container">
                    {comments.length > 20 && (
                      <button
                        onClick={() => scrollToComments('top')}
                        className="btn-scroll-to-top-inline"
                        aria-label="Jump to top of comments"
                      >
                        ↑ Back to Top
                      </button>
                    )}
                    <button
                      onClick={handleLoadMore}
                      disabled={loadingMore}
                      className="btn-load-more"
                      aria-label="Load more comments"
                    >
                      {loadingMore ? (
                        <>
                          <span className="loading-spinner" aria-hidden="true"></span>
                          Loading...
                        </>
                      ) : (
                        <>
                          Load More Comments
                          <span className="remaining-count">
                            ({totalComments - comments.length} more • {Math.ceil((totalComments - comments.length) / COMMENTS_PER_PAGE)} pages)
                          </span>
                        </>
                      )}
                    </button>
                  </div>
                )}

                {!hasMore && comments.length > 0 && comments.length < totalComments && (
                  <div className="end-of-comments">
                    <span>🎉 You've reached the end of all comments</span>
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        {/* Floating scroll to top button */}
        {showScrollToTop && totalComments > 20 && (
          <button
            onClick={() => scrollToComments('top')}
            className="btn-scroll-to-top-floating"
            aria-label="Scroll to top of comments"
            title="Jump to top of comments"
          >
            ↑
          </button>
        )}

        {/* Protocol Document Modal */}
        {showProtocolModal && animal?.protocol_document_url && (
          <ProtocolViewerErrorBoundary onError={(error) => {
            console.error('Protocol viewer error:', error);
            toast.showError('Failed to display protocol document. Please try downloading it.');
          }}>
            <Suspense fallback={
              <div style={{
                position: 'fixed',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                backgroundColor: 'rgba(0, 0, 0, 0.75)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: 1000,
              }}>
                <div style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: '1rem',
                  color: 'white',
                }}>
                  <span className="loading-spinner" aria-label="Loading protocol viewer"></span>
                  <p>Loading protocol viewer...</p>
                </div>
              </div>
            }>
              <ProtocolViewer
                documentUrl={animal.protocol_document_url}
                documentName={animal.protocol_document_name}
                onClose={handleCloseProtocolModal}
              />
            </Suspense>
          </ProtocolViewerErrorBoundary>
        )}

        {/* Script Viewer Modal */}
        {viewingScript && (
          <div
            role="dialog"
            aria-modal="true"
            aria-label={viewingScript.title}
            style={{
              position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
              backgroundColor: 'rgba(0,0,0,0.75)', display: 'flex',
              alignItems: 'center', justifyContent: 'center', zIndex: 1000,
              padding: '1rem',
            }}
            onClick={(e) => { if (e.target === e.currentTarget) setViewingScript(null); }}
          >
            <div style={{
              background: 'var(--surface)', borderRadius: '12px',
              width: '100%', maxWidth: '900px', maxHeight: '90vh',
              display: 'flex', flexDirection: 'column', overflow: 'hidden',
            }}>
              <div style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                padding: '1rem 1.5rem', borderBottom: '1px solid var(--border-color)',
              }}>
                <h2 style={{ margin: 0, fontSize: '1.25rem' }}>{viewingScript.title}</h2>
                <button
                  onClick={() => setViewingScript(null)}
                  aria-label="Close script viewer"
                  style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.5rem', color: 'var(--text-secondary)' }}
                >
                  ×
                </button>
              </div>
              <div style={{ flex: 1, overflow: 'auto' }}>
                <AuthenticatedScriptViewer script={viewingScript} />
              </div>
            </div>
          </div>
        )}

        {/* Manage Scripts Modal */}
        {manageScriptsOpen && (
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Manage Scripts"
            style={{
              position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
              backgroundColor: 'rgba(0,0,0,0.6)', display: 'flex',
              alignItems: 'center', justifyContent: 'center', zIndex: 1000,
              padding: '1rem',
            }}
          >
            <div style={{
              background: 'var(--surface)', borderRadius: '12px',
              width: '100%', maxWidth: '500px', maxHeight: '80vh',
              display: 'flex', flexDirection: 'column', overflow: 'hidden',
            }}>
              <div style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                padding: '1rem 1.5rem', borderBottom: '1px solid var(--border-color)',
              }}>
                <h2 style={{ margin: 0, fontSize: '1.25rem' }}>Manage Scripts</h2>
                <button
                  onClick={() => setManageScriptsOpen(false)}
                  aria-label="Close"
                  style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.5rem', color: 'var(--text-secondary)' }}
                >
                  ×
                </button>
              </div>
              <div style={{ flex: 1, overflow: 'auto', padding: '1.25rem 1.5rem' }}>
                {groupScripts.length === 0 ? (
                  <p style={{ color: 'var(--text-secondary)' }}>No scripts available in this group. Upload scripts from the Scripts tab first.</p>
                ) : (
                  <fieldset style={{ border: 'none', padding: 0, margin: 0 }}>
                    <legend style={{ fontWeight: 600, marginBottom: '0.75rem', fontSize: '0.9375rem' }}>Select scripts to assign:</legend>
                    {groupScripts.map((script) => (
                      <label
                        key={script.id}
                        style={{
                          display: 'flex', alignItems: 'flex-start', gap: '0.75rem',
                          padding: '0.625rem 0', cursor: 'pointer',
                          borderBottom: '1px solid var(--border-color)',
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={selectedScriptIds.has(script.id)}
                          onChange={(e) => {
                            setSelectedScriptIds((prev) => {
                              const next = new Set(prev);
                              if (e.target.checked) next.add(script.id); else next.delete(script.id);
                              return next;
                            });
                          }}
                          aria-label={script.title}
                          style={{ marginTop: '2px', flexShrink: 0 }}
                        />
                        <div>
                          <div style={{ fontWeight: 500 }}>{script.title}</div>
                          {script.description && (
                            <div style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>{script.description}</div>
                          )}
                        </div>
                      </label>
                    ))}
                  </fieldset>
                )}
              </div>
              <div style={{
                display: 'flex', gap: '0.75rem', justifyContent: 'flex-end',
                padding: '1rem 1.5rem', borderTop: '1px solid var(--border-color)',
              }}>
                <button
                  className="btn-secondary"
                  onClick={() => setManageScriptsOpen(false)}
                  disabled={savingScripts}
                  style={{ padding: '0.5rem 1.25rem', borderRadius: '8px', border: '1px solid var(--border-color)', background: 'var(--surface)', cursor: 'pointer', fontFamily: 'inherit', fontWeight: 500 }}
                >
                  Cancel
                </button>
                <button
                  className="btn-primary"
                  disabled={savingScripts || groupScripts.length === 0}
                  style={{ padding: '0.5rem 1.5rem', borderRadius: '8px', background: 'var(--brand)', color: 'white', border: 'none', cursor: 'pointer', fontFamily: 'inherit', fontWeight: 600 }}
                  onClick={async () => {
                    try {
                      setSavingScripts(true);
                      await scriptsApi.setAnimalScripts(Number(groupId), Number(id), Array.from(selectedScriptIds));
                      // Reload animal to get updated scripts
                      const res = await animalsApi.getById(Number(groupId), Number(id));
                      setAnimal(res.data);
                      setManageScriptsOpen(false);
                      toast.showSuccess('Protocol scripts updated');
                    } catch {
                      toast.showError('Failed to update scripts. Please try again.');
                    } finally {
                      setSavingScripts(false);
                    }
                  }}
                >
                  {savingScripts ? 'Saving…' : 'Save Changes'}
                </button>
              </div>
            </div>
          </div>
        )}
        {showHistoryModal && (
          <CommentHistoryModal history={commentHistory} onClose={() => setShowHistoryModal(false)} />
        )}
      </div>
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        title={confirmDialog.title}
        message={confirmDialog.message}
        variant="danger"
        confirmLabel="Delete"
        onConfirm={confirmDialog.onConfirm}
        onCancel={closeConfirmDialog}
      />
    </div>
  );
};

// ─────────────────────────────────────────────
// AuthenticatedScriptViewer — fetches the file with the stored JWT then
// renders it inline: DOCX → HTML (via mammoth), PDF → iframe, other → download
// ─────────────────────────────────────────────
interface AuthenticatedScriptViewerProps {
  script: Script;
}

const AuthenticatedScriptViewer: React.FC<AuthenticatedScriptViewerProps> = ({ script }) => {
  const [viewState, setViewState] = useState<{
    status: 'loading' | 'pdf' | 'docx' | 'other' | 'error';
    blobUrl: string | null;
    htmlContent: string | null;
  }>({ status: 'loading', blobUrl: null, htmlContent: null });

  useEffect(() => {
    let activeBlobUrl: string | null = null;
    let cancelled = false;

    const load = async () => {
      try {
        const token = localStorage.getItem('token');
        const res = await fetch(script.file_url, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok || cancelled) return;

        const name = script.file_name.toLowerCase();
        const type = (script.file_type || '').toLowerCase();
        const isDocx =
          name.endsWith('.docx') ||
          name.endsWith('.doc') ||
          type.includes('officedocument') ||
          type.includes('msword');

        if (isDocx) {
          const arrayBuffer = await res.arrayBuffer();
          if (cancelled) return;
          const mammoth = (await import('mammoth')).default;
          const result = await mammoth.convertToHtml({ arrayBuffer });
          if (cancelled) return;
          const DOMPurify = (await import('dompurify')).default;
          const safe = DOMPurify.sanitize(result.value);
          setViewState({ status: 'docx', blobUrl: null, htmlContent: safe });
        } else {
          const blob = await res.blob();
          if (cancelled) return;
          activeBlobUrl = URL.createObjectURL(blob);
          const isPdf =
            type === 'application/pdf' || name.endsWith('.pdf');
          setViewState({ status: isPdf ? 'pdf' : 'other', blobUrl: activeBlobUrl, htmlContent: null });
        }
      } catch {
        if (!cancelled) setViewState({ status: 'error', blobUrl: null, htmlContent: null });
      }
    };

    load();
    return () => {
      cancelled = true;
      if (activeBlobUrl) URL.revokeObjectURL(activeBlobUrl);
    };
  }, [script.file_url, script.file_name, script.file_type]);

  if (viewState.status === 'loading') {
    return (
      <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-secondary)' }}>
        Loading…
      </div>
    );
  }

  if (viewState.status === 'error') {
    return (
      <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-secondary)' }}>
        Unable to load file.
      </div>
    );
  }

  if (viewState.status === 'docx' && viewState.htmlContent) {
    return (
      <div
        style={{
          padding: '1.5rem',
          lineHeight: '1.7',
          overflowY: 'auto',
          maxHeight: '65vh',
          color: 'var(--text-primary)',
        }}
        // eslint-disable-next-line react/no-danger
        dangerouslySetInnerHTML={{ __html: viewState.htmlContent }}
      />
    );
  }

  if (viewState.status === 'pdf' && viewState.blobUrl) {
    return (
      <iframe
        src={viewState.blobUrl}
        title={script.title}
        style={{ width: '100%', height: '65vh', border: 'none', display: 'block' }}
        aria-label={`${script.title} PDF viewer`}
      />
    );
  }

  // Other file types — offer blob-URL download (no forced download prompt)
  return (
    <div style={{ textAlign: 'center', padding: '2rem' }}>
      <p style={{ marginBottom: '1rem' }}>{script.file_name}</p>
      {viewState.blobUrl && (
        <a
          href={viewState.blobUrl}
          download={script.file_name}
          className="btn-view-document"
          aria-label={`Download ${script.file_name}`}
        >
          Download File
        </a>
      )}
    </div>
  );
};

export default AnimalDetailPage;
