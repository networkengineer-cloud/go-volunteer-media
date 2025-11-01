import React, { useEffect, useState, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, animalsApi as imageUploadApi, commentTagsApi } from '../api/client';
import type { Animal, AnimalComment, CommentTag } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import './AnimalDetailPage.css';

const AnimalDetailPage: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const [animal, setAnimal] = useState<Animal | null>(null);
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

  useEffect(() => {
    if (groupId && id) {
      loadAnimalData(Number(groupId), Number(id));
      loadTags();
    }
  }, [groupId, id]);

  useEffect(() => {
    if (groupId && id) {
      const tagFilterString = filterTags.join(',');
      loadComments(Number(groupId), Number(id), tagFilterString);
    }
  }, [filterTags, groupId, id]);

  const loadTags = async () => {
    try {
      const res = await commentTagsApi.getAll();
      setAvailableTags(res.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
    }
  };

  const loadAnimalData = async (gId: number, animalId: number) => {
    try {
      setError('');
      const animalRes = await animalsApi.getById(gId, animalId);
      setAnimal(animalRes.data);
      await loadComments(gId, animalId, '');
    } catch (error: any) {
      console.error('Failed to load animal data:', error);
      setError(error.response?.data?.error || 'Failed to load animal information. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const loadComments = async (gId: number, animalId: number, filter: string) => {
    try {
      const commentsRes = await animalCommentsApi.getAll(gId, animalId, filter);
      setComments(commentsRes.data);
    } catch (error) {
      console.error('Failed to load comments:', error);
    }
  };

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    try {
      const res = await imageUploadApi.uploadImage(file);
      setCommentImage(res.data.url);
    } catch (error: any) {
      console.error('Upload error:', error);
      alert('Failed to upload image: ' + (error.response?.data?.error || error.message));
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
    } catch (error: any) {
      console.error('Failed to post comment:', error);
      alert('Failed to post comment: ' + (error.response?.data?.error || error.message));
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
    } catch (error) {
      console.error('Failed to export comments CSV:', error);
      alert('Failed to export comments CSV');
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
        <Link to={`/groups/${groupId}`} className="back-link">
          ← Back to Group
        </Link>

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
              {animal.description && (
                <p className="animal-description">{animal.description}</p>
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
