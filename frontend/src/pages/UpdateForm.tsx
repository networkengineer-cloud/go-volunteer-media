import React, { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { updatesApi, animalsApi } from '../api/client';
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
            <label htmlFor="image_upload">Update Image</label>
            <input
              id="image_upload"
              type="file"
              accept=".jpg,.jpeg,.png,.gif"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                setLoading(true);
                try {
                  const res = await animalsApi.uploadImage(file);
                  setFormData({ ...formData, image_url: res.data.url });
                  alert('Image uploaded successfully!');
                } catch (err: any) {
                  console.error('Upload error:', err);
                  const errorMsg = err.response?.data?.error || err.message || 'Failed to upload image';
                  alert('Failed to upload image: ' + errorMsg);
                } finally {
                  setLoading(false);
                  // Clear the file input
                  if (e.target) e.target.value = '';
                }
              }}
            />
            {formData.image_url && (
              <div className="image-preview">
                <label>Preview:</label>
                <img src={formData.image_url} alt="Update Preview" />
              </div>
            )}
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
