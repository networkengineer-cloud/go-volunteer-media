import React, { useEffect, useState } from 'react';
import { commentTagsApi, statisticsApi, type CommentTag, type CommentTagStatistics } from '../../api/client';
import './CommentTagsTab.css';

const CommentTagsTab: React.FC = () => {
  const [tags, setTags] = useState<CommentTag[]>([]);
  const [statistics, setStatistics] = useState<Record<number, CommentTagStatistics>>({});
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
      const [tagsRes, statsRes] = await Promise.all([
        commentTagsApi.getAll(),
        statisticsApi.getCommentTagStatistics()
      ]);
      
      setTags(tagsRes.data);
      
      // Create a map of tag_id to statistics
      const statsMap: Record<number, CommentTagStatistics> = {};
      statsRes.data.forEach(stat => {
        statsMap[stat.tag_id] = stat;
      });
      setStatistics(statsMap);
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
            {tags.map((tag) => {
              const stats = statistics[tag.id];
              return (
                <div key={tag.id} className="tag-card">
                  <div className="tag-header">
                    <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                      {tag.name}
                    </div>
                    <button
                      onClick={() => handleDeleteTag(tag.id)}
                      className="btn-delete"
                      aria-label="Delete tag"
                      title="Delete tag"
                    >
                      ✕
                    </button>
                  </div>
                  
                  {stats && (
                    <div className="tag-stats">
                      <div className="tag-stat-item">
                        <span className="tag-stat-label">Uses:</span>
                        <span className="tag-stat-value">{stats.usage_count}</span>
                      </div>
                      
                      {stats.last_used && (
                        <div className="tag-stat-item">
                          <span className="tag-stat-label">Last used:</span>
                          <span className="tag-stat-value" title={new Date(stats.last_used).toLocaleString()}>
                            {formatRelativeTime(stats.last_used)}
                          </span>
                        </div>
                      )}
                      
                      {stats.most_tagged_animal_name && (
                        <div className="tag-stat-item">
                          <span className="tag-stat-label">Most tagged:</span>
                          <span className="tag-stat-value">{stats.most_tagged_animal_name}</span>
                        </div>
                      )}
                    </div>
                  )}
                  
                  {!stats || stats.usage_count === 0 && (
                    <div className="tag-stats">
                      <span className="no-usage">Not used yet</span>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

// Helper function to format relative time
function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export default CommentTagsTab;
