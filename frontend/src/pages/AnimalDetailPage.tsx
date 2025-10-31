import React, { useEffect, useState, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, animalsApi as imageUploadApi, commentTagsApi } from '../api/client';
import type { Animal, AnimalComment, CommentTag } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import './AnimalDetailPage.css';

const AnimalDetailPage: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [comments, setComments] = useState<AnimalComment[]>([]);
  const [availableTags, setAvailableTags] = useState<CommentTag[]>([]);
  const [selectedTags, setSelectedTags] = useState<number[]>([]);
  const [tagFilter, setTagFilter] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [commentText, setCommentText] = useState('');
  const [commentImage, setCommentImage] = useState('');
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [showCommentForm, setShowCommentForm] = useState(true);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (groupId && id) {
      loadAnimalData(Number(groupId), Number(id));
      loadTags();
    }
  }, [groupId, id]);

  useEffect(() => {
    if (groupId && id) {
      loadComments(Number(groupId), Number(id), tagFilter);
    }
  }, [tagFilter, groupId, id]);

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
      const animalRes = await animalsApi.getById(gId, animalId);
      setAnimal(animalRes.data);
      await loadComments(gId, animalId, '');
    } catch (error) {
      console.error('Failed to load animal data:', error);
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
      const response = await animalsApi.exportCommentsCSV(
        Number(groupId),
        Number(id)
      );
      
      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${animal?.name || 'animal'}-comments.csv`);
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
    return <div className="loading">Loading...</div>;
  }

  if (!animal) {
    return <div className="error">Animal not found</div>;
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
                {animal.species} {animal.breed && `• ${animal.breed}`} • {animal.age} years old
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
                <span>Tags:</span>
                {availableTags.map((tag) => (
                  <label key={tag.id} className="tag-checkbox">
                    <input
                      type="checkbox"
                      checked={selectedTags.includes(tag.id)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedTags([...selectedTags, tag.id]);
                        } else {
                          setSelectedTags(selectedTags.filter((t) => t !== tag.id));
                        }
                      }}
                    />
                    <span className="tag-label" style={{ backgroundColor: tag.color }}>
                      {tag.name}
                    </span>
                  </label>
                ))}
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
              className={tagFilter === '' ? 'tag-filter active' : 'tag-filter'}
              onClick={() => setTagFilter('')}
            >
              All
            </button>
            {availableTags.map((tag) => (
              <button
                key={tag.id}
                className={tagFilter === tag.name ? 'tag-filter active' : 'tag-filter'}
                style={{ borderColor: tag.color }}
                onClick={() => setTagFilter(tagFilter === tag.name ? '' : tag.name)}
              >
                {tag.name}
              </button>
            ))}
          </div>

          <div className="comments-list">
            {comments.length === 0 ? (
              <p className="no-comments">No comments yet. Be the first to share!</p>
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
