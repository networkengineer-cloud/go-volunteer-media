import React, { useEffect, useState, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { animalsApi, animalCommentsApi, animalsApi as imageUploadApi } from '../api/client';
import type { Animal, AnimalComment } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import './AnimalDetailPage.css';

const AnimalDetailPage: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [comments, setComments] = useState<AnimalComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [commentText, setCommentText] = useState('');
  const [commentImage, setCommentImage] = useState('');
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (groupId && id) {
      loadAnimalData(Number(groupId), Number(id));
    }
  }, [groupId, id]);

  const loadAnimalData = async (gId: number, animalId: number) => {
    try {
      const [animalRes, commentsRes] = await Promise.all([
        animalsApi.getById(gId, animalId),
        animalCommentsApi.getAll(gId, animalId),
      ]);
      setAnimal(animalRes.data);
      setComments(commentsRes.data);
    } catch (error) {
      console.error('Failed to load animal data:', error);
    } finally {
      setLoading(false);
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
        commentImage
      );
      setComments([newComment.data, ...comments]);
      setCommentText('');
      setCommentImage('');
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
          <h2>Comments & Photos</h2>

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
