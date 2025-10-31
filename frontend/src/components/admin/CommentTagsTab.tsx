import React, { useEffect, useState } from 'react';
import { commentTagsApi, type CommentTag } from '../../api/client';
import './CommentTagsTab.css';

const CommentTagsTab: React.FC = () => {
  const [tags, setTags] = useState<CommentTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [newTagName, setNewTagName] = useState('');
  const [newTagColor, setNewTagColor] = useState('#006b54');
  const [message, setMessage] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadTags();
  }, []);

  const loadTags = async () => {
    try {
      const response = await commentTagsApi.getAll();
      setTags(response.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
      setMessage('Failed to load tags');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTag = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!newTagName.trim()) {
      setMessage('Tag name is required');
      return;
    }

    setSaving(true);
    setMessage('');

    try {
      const response = await commentTagsApi.create(newTagName.trim(), newTagColor);
      setTags([...tags, response.data]);
      setNewTagName('');
      setNewTagColor('#006b54');
      setMessage('Tag created successfully!');
      setTimeout(() => setMessage(''), 3000);
    } catch (error) {
      console.error('Failed to create tag:', error);
      setMessage('Failed to create tag');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteTag = async (tagId: number) => {
    if (!confirm('Are you sure you want to delete this tag?')) {
      return;
    }

    try {
      await commentTagsApi.delete(tagId);
      setTags(tags.filter(t => t.id !== tagId));
      setMessage('Tag deleted successfully!');
      setTimeout(() => setMessage(''), 3000);
    } catch (error) {
      console.error('Failed to delete tag:', error);
      setMessage('Failed to delete tag');
    }
  };

  if (loading) {
    return <div className="loading">Loading tags...</div>;
  }

  return (
    <div className="tab-panel">
      <h2>Comment Tags</h2>
      <p className="section-description">
        Create custom tags that users can apply to animal comments for better organization.
      </p>

      <form onSubmit={handleCreateTag} className="create-tag-form">
        <div className="form-row">
          <div className="form-group">
            <label htmlFor="tagName">Tag Name</label>
            <input
              type="text"
              id="tagName"
              value={newTagName}
              onChange={(e) => setNewTagName(e.target.value)}
              placeholder="e.g., Needs Attention, Good News"
              maxLength={50}
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="tagColor">Color</label>
            <input
              type="color"
              id="tagColor"
              value={newTagColor}
              onChange={(e) => setNewTagColor(e.target.value)}
            />
          </div>
          <button type="submit" className="btn-create" disabled={saving}>
            {saving ? 'Creating...' : 'Create Tag'}
          </button>
        </div>
      </form>

      {message && (
        <div className={`message ${message.includes('success') ? 'success' : 'error'}`}>
          {message}
        </div>
      )}

      <div className="tags-list">
        <h3>Existing Tags ({tags.length})</h3>
        {tags.length === 0 ? (
          <p className="no-tags">No tags created yet. Create your first tag above.</p>
        ) : (
          <div className="tags-grid">
            {tags.map((tag) => (
              <div key={tag.id} className="tag-card">
                <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                  {tag.name}
                </div>
                <button
                  onClick={() => handleDeleteTag(tag.id)}
                  className="btn-delete"
                  aria-label="Delete tag"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default CommentTagsTab;
