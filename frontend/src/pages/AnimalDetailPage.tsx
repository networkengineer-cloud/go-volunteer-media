import React, { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, animalsApi as imageUploadApi, commentTagsApi, groupsApi } from '../api/client';
import type { Animal, AnimalComment, CommentTag, Group } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
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
  const { isAdmin } = useAuth();
  const toast = useToast();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [group, setGroup] = useState<Group | null>(null);
  const [comments, setComments] = useState<AnimalComment[]>([]);
  const [availableTags, setAvailableTags] = useState<CommentTag[]>([]);
  const [selectedTags, setSelectedTags] = useState<number[]>([]);
  const [filterTags, setFilterTags] = useState<string[]>([]); // Tags selected for filtering
  const [loading, setLoading] = useState(true);
  const [commentText, setCommentText] = useState('');
  const [commentImage, setCommentImage] = useState('');
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [showCommentForm, setShowCommentForm] = useState(true);
  const [error, setError] = useState<string>('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadComments = useCallback(async (gId: number, animalId: number, filter: string) => {
    try {
      const commentsRes = await animalCommentsApi.getAll(gId, animalId, filter);
      setComments(commentsRes.data);
    } catch (error) {
      console.error('Failed to load comments:', error);
    }
  }, []);

  const loadTags = useCallback(async () => {
    try {
      const res = await commentTagsApi.getAll();
      setAvailableTags(res.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
    }
  }, []);

  const loadGroupData = useCallback(async (gId: number) => {
    try {
      const groupRes = await groupsApi.getById(gId);
      setGroup(groupRes.data);
    } catch (error) {
      console.error('Failed to load group data:', error);
    }
  }, []);

  const loadAnimalData = useCallback(async (gId: number, animalId: number) => {
    try {
      setError('');
      const animalRes = await animalsApi.getById(gId, animalId);
      setAnimal(animalRes.data);
      await loadComments(gId, animalId, '');
    } catch (error: unknown) {
      console.error('Failed to load animal data:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to load animal information. Please try again.';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [loadComments]);

  useEffect(() => {
    if (groupId && id) {
      loadAnimalData(Number(groupId), Number(id));
      loadGroupData(Number(groupId));
      loadTags();
    }
  }, [groupId, id, loadAnimalData, loadGroupData, loadTags]);

  useEffect(() => {
    if (groupId && id) {
      const tagFilterString = filterTags.join(',');
      loadComments(Number(groupId), Number(id), tagFilterString);
    }
  }, [filterTags, groupId, id, loadComments]);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      toast.showError('Please select an image file (JPG, PNG, or GIF)');
      return;
    }

    // Validate file size (max 10MB)
    if (file.size > 10 * 1024 * 1024) {
      toast.showError(`Image size must be under 10MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
      return;
    }

    setUploading(true);
    try {
      const res = await imageUploadApi.uploadImage(file);
      setCommentImage(res.data.url);
      toast.showSuccess('Image uploaded successfully!');
    } catch (error: unknown) {
      console.error('Upload error:', error);
      const errorMsg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleSubmitComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commentText.trim() && !commentImage) {
      toast.showWarning('Please enter a comment or upload an image');
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
      setComments([newComment.data, ...comments]);
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
                {animal.breed && `${animal.breed} • `}{animal.age} years old
              </p>
              <span className={`status ${animal.status}`}>{animal.status}</span>
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
              
              {isAdmin && (
                <button onClick={handleEdit} className="btn-edit">
                  Edit Animal Details
                </button>
              )}
            </div>
          </div>
        </div>

        <div className="comments-section">
          <div className="comments-header">
            <h2>Comments & Photos</h2>
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
              <textarea
                placeholder="Share an update or comment about this animal..."
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
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".jpg,.jpeg,.png,.gif"
                  onChange={handleImageUpload}
                  style={{ display: 'none' }}
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="btn-upload"
                  disabled={uploading || submitting}
                >
                  {uploading ? 'Uploading...' : '📷 Add Photo'}
                </button>
                <button type="submit" className="btn-post" disabled={submitting || (!commentText.trim() && !commentImage)}>
                  {submitting ? 'Posting...' : 'Post'}
                </button>
              </div>
            </form>
          )}

          <div className="tag-filters">
            <button
              className={filterTags.length === 0 ? 'tag-filter active' : 'tag-filter'}
              onClick={() => setFilterTags([])}
            >
              All {filterTags.length === 0 && `(${comments.length})`}
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
              comments.map((comment) => (
                <div key={comment.id} className="comment-card">
                  <div className="comment-header">
                    <span className="comment-author">{comment.user?.username}</span>
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
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default AnimalDetailPage;
