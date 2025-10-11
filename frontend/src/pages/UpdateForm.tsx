import React, { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { updatesApi } from '../api/client';
import './Form.css';

const UpdateForm: React.FC = () => {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    title: '',
    content: '',
    image_url: '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      if (groupId) {
        await updatesApi.create(
          parseInt(groupId),
          formData.title,
          formData.content,
          formData.image_url
        );
        navigate(`/groups/${groupId}`);
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Failed to create update');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="form-page">
      <div className="form-container">
        <Link to={`/groups/${groupId}`} className="back-link">
          ← Back to Group
        </Link>
        <h1>New Update</h1>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="title">Title *</label>
            <input
              id="title"
              type="text"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="content">Content *</label>
            <textarea
              id="content"
              value={formData.content}
              onChange={(e) => setFormData({ ...formData, content: e.target.value })}
              rows={8}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="image_url">Image URL</label>
            <input
              id="image_url"
              type="url"
              value={formData.image_url}
              onChange={(e) => setFormData({ ...formData, image_url: e.target.value })}
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? 'Posting...' : 'Post Update'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default UpdateForm;
