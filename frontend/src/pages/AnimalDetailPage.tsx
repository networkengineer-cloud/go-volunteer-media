import React, { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, commentTagsApi, groupsApi } from '../api/client';
import type { Animal, AnimalComment, CommentTag, Group, GroupMembership } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import { renderAsync as renderDocxAsync } from 'docx-preview';
import './AnimalDetailPage.css';

// Helper function to calculate quarantine end date (10 days, cannot end on weekend)
const calculateQuarantineEndDate = (startDateString?: string): string => {
  if (!startDateString) return '-';
  
  const startDate = new Date(startDateString);
  // Add 10 days
  const endDate = new Date(startDate);
  endDate.setDate(endDate.getDate() + 10);
  
  // If end date is Saturday (6) or Sunday (0), move to Monday
  while (endDate.getDay() === 0 || endDate.getDay() === 6) {
    endDate.setDate(endDate.getDate() + 1);
  }
  
  return endDate.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
};

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
  const [selectedTags, setSelectedTags] = useState<number[]>([]);
  const [filterTags, setFilterTags] = useState<string[]>([]); // Tags selected for filtering
  const [loading, setLoading] = useState(true);
  const [commentText, setCommentText] = useState('');
  const [commentImage, setCommentImage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [showCommentForm, setShowCommentForm] = useState(true);
  const [error, setError] = useState<string>('');
  
  // Pagination state
  const [totalComments, setTotalComments] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc'); // Default: newest first
  const [showScrollToTop, setShowScrollToTop] = useState(false);
  const [showProtocolModal, setShowProtocolModal] = useState(false);
  const [protocolDocumentUrl, setProtocolDocumentUrl] = useState<string | null>(null);
  const [protocolDocumentBlob, setProtocolDocumentBlob] = useState<Blob | null>(null);
  const [protocolDocumentKind, setProtocolDocumentKind] = useState<'pdf' | 'docx' | 'unknown'>('unknown');
  const [protocolRenderError, setProtocolRenderError] = useState<string | null>(null);
  const [loadingProtocol, setLoadingProtocol] = useState(false);
  const protocolDocxBodyRef = useRef<HTMLDivElement>(null);
  const protocolDocxStyleRef = useRef<HTMLDivElement>(null);
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

  const handleDeleteComment = async (commentId: number) => {
    if (!groupId || !id) return;
    
    if (!window.confirm('Are you sure you want to delete this comment?')) {
      return;
    }

    try {
      await animalCommentsApi.delete(Number(groupId), Number(id), commentId);
      toast.showSuccess('Comment deleted successfully');
      
      // Reload comments
      await loadComments(Number(groupId), Number(id), filterTags.join(','), 0, sortOrder, false);
      
      // Reload deleted comments if admin
      if (isAdmin) {
        await loadDeletedComments(Number(groupId));
      }
    } catch (error) {
      console.error('Failed to delete comment:', error);
      toast.showError('Failed to delete comment. Please try again.');
    }
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

  const handleSubmitComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commentText.trim() && !commentImage) {
      toast.showWarning('Please write a comment');
      return;
    }

    setSubmitting(true);
    try {
      const newComment = await animalCommentsApi.create(
        Number(groupId),
        Number(id),
        commentText,
        commentImage,
        selectedTags.length > 0 ? selectedTags : undefined
      );
      
      // Add new comment to the beginning or end based on sort order
      if (sortOrder === 'desc') {
        setComments([newComment.data, ...comments]);
      } else {
        setComments([...comments, newComment.data]);
      }
      
      setTotalComments(prev => prev + 1);
      setCommentText('');
      setCommentImage('');
      setSelectedTags([]);
      toast.showSuccess('Comment posted successfully!');
    } catch (error: unknown) {
      console.error('Failed to post comment:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to post comment. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setSubmitting(false);
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

  const handleOpenProtocolDocument = async () => {
    if (!animal?.protocol_document_url) return;
    
    setLoadingProtocol(true);
    setShowProtocolModal(true);
    setProtocolRenderError(null);
    
    try {
      // Extract UUID from the document URL (format: /api/documents/UUID)
      const uuid = animal.protocol_document_url.split('/').pop();
      if (!uuid) {
        toast.showError('Invalid document URL');
        setShowProtocolModal(false);
        return;
      }

      // Fetch document with authorization header via API client
      const response = await animalsApi.getProtocolDocument(uuid);

      // Axios responseType=blob -> response.data is a Blob
      const responseBlob = response.data as Blob;
      const headerContentType = (response.headers?.['content-type'] || response.headers?.['Content-Type']) as string | undefined;
      const filename = (animal.protocol_document_name || '').toLowerCase();

      const inferredContentType =
        responseBlob.type ||
        headerContentType ||
        (filename.endsWith('.docx')
          ? 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
          : 'application/pdf');

      const isDocx =
        inferredContentType.includes('officedocument.wordprocessingml.document') ||
        filename.endsWith('.docx');

      // Clear any previous preview state
      if (protocolDocumentUrl) {
        window.URL.revokeObjectURL(protocolDocumentUrl);
        setProtocolDocumentUrl(null);
      }
      setProtocolDocumentBlob(null);

      if (isDocx) {
        setProtocolDocumentKind('docx');
        setProtocolDocumentBlob(new Blob([responseBlob], { type: inferredContentType }));
      } else {
        setProtocolDocumentKind('pdf');
        const pdfBlob = new Blob([responseBlob], { type: inferredContentType });
        const blobUrl = window.URL.createObjectURL(pdfBlob);
        setProtocolDocumentUrl(blobUrl);
      }
    } catch (error) {
      console.error('Failed to open protocol document:', error);
      toast.showError('Failed to open protocol document. Please try again.');
      setShowProtocolModal(false);
    } finally {
      setLoadingProtocol(false);
    }
  };

  const handleCloseProtocolModal = () => {
    setShowProtocolModal(false);
    setProtocolRenderError(null);
    // Clean up blob URL
    if (protocolDocumentUrl) {
      window.URL.revokeObjectURL(protocolDocumentUrl);
      setProtocolDocumentUrl(null);
    }
    setProtocolDocumentBlob(null);
    setProtocolDocumentKind('unknown');
  };

  useEffect(() => {
    const shouldRender = showProtocolModal && !loadingProtocol && protocolDocumentKind === 'docx' && protocolDocumentBlob;
    if (!shouldRender) return;

    const bodyContainer = protocolDocxBodyRef.current;
    const styleContainer = protocolDocxStyleRef.current;
    if (!bodyContainer || !styleContainer) return;

    bodyContainer.innerHTML = '';
    styleContainer.innerHTML = '';

    let cancelled = false;
    (async () => {
      try {
        await renderDocxAsync(protocolDocumentBlob, bodyContainer, styleContainer, {
          inWrapper: true,
          breakPages: true,
          useBase64URL: true,
        });
      } catch (err) {
        if (cancelled) return;
        console.error('Failed to render protocol DOCX:', err);
        setProtocolRenderError('Unable to render this DOCX document.');
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [showProtocolModal, loadingProtocol, protocolDocumentKind, protocolDocumentBlob]);

  // Close modal on Escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && showProtocolModal) {
        handleCloseProtocolModal();
      }
    };
    
    if (showProtocolModal) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [showProtocolModal]);

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
                {animal.breed && `${animal.breed} • `}{animal.age} years old • ID: {animal.id}
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
                    End: {calculateQuarantineEndDate(animal.quarantine_start_date)}
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
              
              <div className="animal-action-buttons">
                <Link 
                  to={`/groups/${groupId}/animals/${id}/photos`} 
                  className="btn-view-gallery-large"
                  title="View and manage animal photos"
                >
                  📷 Photo Gallery
                </Link>
                
                {isAdmin && (
                  <button onClick={handleEdit} className="btn-edit">
                    Edit Animal Details
                  </button>
                )}
                {!isAdmin && (membership?.is_group_admin) && (
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
            <form onSubmit={handleSubmitComment} className="comment-form">
              <div className="form-section-header">
                <span className="form-hint">💬 Add a comment about this animal</span>
              </div>
              <textarea
                placeholder="Add a comment..."
                value={commentText}
                onChange={(e) => setCommentText(e.target.value)}
                rows={3}
                disabled={submitting}
              />
              {commentImage && (
                <div className="comment-image-preview">
                  <img src={commentImage} alt="Preview" />
                  <button
                    type="button"
                    onClick={() => setCommentImage('')}
                    className="remove-image"
                  >
                    ✕
                  </button>
                </div>
              )}
              <div className="comment-tags-selection">
                <label htmlFor="tag-select" className="tag-select-label">Tags (optional):</label>
                <select
                  id="tag-select"
                  multiple
                  size={3}
                  value={selectedTags.map(String)}
                  onChange={(e) => {
                    const options = Array.from(e.target.selectedOptions);
                    setSelectedTags(options.map(opt => Number(opt.value)));
                  }}
                  className="tag-multi-select"
                >
                  {availableTags.map((tag) => (
                    <option key={tag.id} value={tag.id}>
                      {tag.name}
                    </option>
                  ))}
                </select>
                <span className="tag-select-hint">Hold Ctrl/Cmd to select multiple tags</span>
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
                            aria-label={`Remove ${tag.name}`}
                          >
                            ×
                          </button>
                        </span>
                      ) : null;
                    })}
                  </div>
                )}
              </div>
              <div className="comment-form-actions">
                <button 
                  type="submit" 
                  className="btn-post" 
                  disabled={submitting || (!commentText.trim() && !commentImage)}
                  title={!commentText.trim() && !commentImage ? 'Add a comment or photo first' : 'Post comment'}
                >
                  {submitting ? 'Posting...' : 'Post Comment'}
                </button>
              </div>
            </form>
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
                        {comment.user.username}
                      </Link>
                    ) : (
                      <span className="comment-author">{comment.user?.username}</span>
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
                  <p className="comment-content">{comment.content}</p>
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
                {comments.map((comment) => (
                  <div key={comment.id} className="comment-card">
                    <div className="comment-header">
                      {comment.user?.id ? (
                        <Link to={`/users/${comment.user.id}/profile`} className="comment-author comment-author--link">
                          {comment.user.username}
                        </Link>
                      ) : (
                        <span className="comment-author">{comment.user?.username}</span>
                      )}
                      <span className="comment-date">
                        {new Date(comment.created_at).toLocaleDateString()} at{' '}
                        {new Date(comment.created_at).toLocaleTimeString([], {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                      {(isAdmin || comment.user?.id === user?.id) && (
                        <button
                          className="btn-delete-comment"
                          onClick={() => handleDeleteComment(comment.id)}
                          title="Delete comment"
                        >
                          🗑️
                        </button>
                      )}
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
                    <p className="comment-content">{comment.content}</p>
                    {comment.image_url && (
                      <img
                        src={comment.image_url}
                        alt="Comment"
                        className="comment-image"
                      />
                    )}
                  </div>
                ))}

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
        {showProtocolModal && (
          <div 
            className="protocol-modal-overlay"
            onClick={handleCloseProtocolModal}
            role="dialog"
            aria-modal="true"
            aria-labelledby="protocol-modal-title"
          >
            <div 
              className="protocol-modal-content"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="protocol-modal-header">
                <div>
                  <h2 id="protocol-modal-title">📋 Protocol Document</h2>
                  <p className="protocol-modal-filename">{animal?.protocol_document_name}</p>
                </div>
                <button
                  onClick={handleCloseProtocolModal}
                  className="protocol-modal-close"
                  aria-label="Close protocol document"
                  title="Close (Esc)"
                >
                  ✕
                </button>
              </div>
              <div className="protocol-modal-body">
                {loadingProtocol ? (
                  <div className="protocol-loading">
                    <span className="loading-spinner" aria-label="Loading protocol document"></span>
                    <p>Loading protocol document...</p>
                  </div>
                ) : protocolRenderError ? (
                  <div className="protocol-render-error" role="alert">
                    <p>{protocolRenderError}</p>
                  </div>
                ) : protocolDocumentKind === 'docx' ? (
                  <div className="protocol-docx-root">
                    <div ref={protocolDocxStyleRef} />
                    <div ref={protocolDocxBodyRef} className="protocol-docx-container" />
                  </div>
                ) : protocolDocumentUrl ? (
                  <iframe
                    src={protocolDocumentUrl}
                    title={animal?.protocol_document_name || 'Protocol Document'}
                    className="protocol-iframe"
                  />
                ) : null}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AnimalDetailPage;
