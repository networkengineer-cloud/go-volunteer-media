import React, { useState, useEffect, useCallback } from 'react';
import apiClient, { commentTagsApi, statisticsApi, type CommentTag, type CommentTagStatistics } from '../api/client';
import Modal from '../components/Modal';
import './AdminAnimalTagsPage.css';

interface AnimalTag {
  id: number;
  name: string;
  category: string;
  color: string;
  created_at: string;
  updated_at: string;
}

// Color presets used across all tag forms
const colorPresets = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308',
  '#84cc16', '#22c55e', '#10b981', '#14b8a6',
  '#06b6d4', '#0ea5e9', '#3b82f6', '#6366f1',
  '#8b5cf6', '#a855f7', '#d946ef', '#ec4899',
  '#f43f5e', '#6b7280', '#374151', '#1f2937',
];

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

// Isolated form component for Animal Tags
interface AnimalTagFormModalProps {
  isOpen: boolean;
  editingTag: AnimalTag | null;
  onClose: () => void;
  onSubmit: (data: { name: string; category: string; color: string }) => Promise<void>;
}

const AnimalTagFormModal: React.FC<AnimalTagFormModalProps> = ({ isOpen, editingTag, onClose, onSubmit }) => {
  const [name, setName] = useState('');
  const [category, setCategory] = useState<'behavior' | 'walker_status'>('behavior');
  const [color, setColor] = useState('#6b7280');
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      if (editingTag) {
        setName(editingTag.name);
        setCategory(editingTag.category as 'behavior' | 'walker_status');
        setColor(editingTag.color);
      } else {
        setName('');
        setCategory('behavior');
        setColor('#6b7280');
      }
    }
  }, [isOpen, editingTag]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    
    setIsSubmitting(true);
    try {
      await onSubmit({ name: name.trim().toLowerCase(), category, color });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={editingTag ? 'Edit Animal Tag' : 'Create Animal Tag'}
    >
      <form onSubmit={handleSubmit} className="tag-form">
        <div className="form-group">
          <label htmlFor="animalTagName">Tag Name</label>
          <input
            type="text"
            id="animalTagName"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., jumpy, leash-reactive, staff-only"
            required
            autoComplete="off"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label htmlFor="animalTagCategory">Category</label>
          <select
            id="animalTagCategory"
            value={category}
            onChange={(e) => setCategory(e.target.value as 'behavior' | 'walker_status')}
          >
            <option value="behavior">Behavior Trait</option>
            <option value="walker_status">Walker Level</option>
          </select>
          <p className="form-hint">
            {category === 'behavior' 
              ? 'Behavior traits describe how an animal acts (e.g., jumpy, mouthy, reactive)'
              : 'Walker levels indicate required volunteer experience (e.g., green, yellow, red)'}
          </p>
        </div>

        <div className="form-group">
          <label>Color</label>
          <div className="color-picker">
            <input
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              className="color-input"
            />
            <div className="color-presets">
              {colorPresets.map((presetColor) => (
                <button
                  key={presetColor}
                  type="button"
                  className={`color-preset ${color === presetColor ? 'selected' : ''}`}
                  style={{ backgroundColor: presetColor }}
                  onClick={() => setColor(presetColor)}
                  aria-label={`Select color ${presetColor}`}
                />
              ))}
            </div>
          </div>
        </div>

        <div className="tag-preview">
          <span>Preview:</span>
          <span className="tag-badge" style={{ backgroundColor: color }}>
            {name || 'tag name'}
          </span>
        </div>

        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={isSubmitting || !name.trim()}>
            {isSubmitting ? 'Saving...' : editingTag ? 'Update Tag' : 'Create Tag'}
          </button>
        </div>
      </form>
    </Modal>
  );
};

// Comment Tag Form Modal
interface CommentTagFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (name: string, color: string) => Promise<void>;
}

const CommentTagFormModal: React.FC<CommentTagFormModalProps> = ({ isOpen, onClose, onSubmit }) => {
  const [name, setName] = useState('');
  const [color, setColor] = useState('#006b54');
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setName('');
      setColor('#006b54');
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    
    setIsSubmitting(true);
    try {
      await onSubmit(name.trim(), color);
      onClose();
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Create Comment Tag"
    >
      <form onSubmit={handleSubmit} className="tag-form">
        <div className="form-group">
          <label htmlFor="commentTagName">Tag Name</label>
          <input
            type="text"
            id="commentTagName"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Medical Update, Behavior Note"
            required
            autoComplete="off"
            autoFocus
            maxLength={50}
          />
        </div>

        <div className="form-group">
          <label>Color</label>
          <div className="color-picker">
            <input
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              className="color-input"
            />
            <div className="color-presets">
              {colorPresets.map((presetColor) => (
                <button
                  key={presetColor}
                  type="button"
                  className={`color-preset ${color === presetColor ? 'selected' : ''}`}
                  style={{ backgroundColor: presetColor }}
                  onClick={() => setColor(presetColor)}
                  aria-label={`Select color ${presetColor}`}
                />
              ))}
            </div>
          </div>
        </div>

        <div className="tag-preview">
          <span>Preview:</span>
          <span className="tag-badge" style={{ backgroundColor: color }}>
            {name || 'tag name'}
          </span>
        </div>

        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={isSubmitting || !name.trim()}>
            {isSubmitting ? 'Creating...' : 'Create Tag'}
          </button>
        </div>
      </form>
    </Modal>
  );
};

const AdminAnimalTagsPage: React.FC = () => {
  // Animal Tags state
  const [animalTags, setAnimalTags] = useState<AnimalTag[]>([]);
  const [loadingAnimalTags, setLoadingAnimalTags] = useState(true);
  const [animalTagError, setAnimalTagError] = useState<string | null>(null);
  const [isAnimalTagModalOpen, setIsAnimalTagModalOpen] = useState(false);
  const [editingAnimalTag, setEditingAnimalTag] = useState<AnimalTag | null>(null);

  // Comment Tags state
  const [commentTags, setCommentTags] = useState<CommentTag[]>([]);
  const [commentTagStats, setCommentTagStats] = useState<Record<number, CommentTagStatistics>>({});
  const [loadingCommentTags, setLoadingCommentTags] = useState(true);
  const [commentTagError, setCommentTagError] = useState<string | null>(null);
  const [isCommentTagModalOpen, setIsCommentTagModalOpen] = useState(false);

  // Active section for mobile
  const [activeSection, setActiveSection] = useState<'animal' | 'comment'>('animal');

  // Fetch Animal Tags
  const fetchAnimalTags = useCallback(async () => {
    try {
      setLoadingAnimalTags(true);
      const response = await apiClient.get('/animal-tags');
      setAnimalTags(response.data);
      setAnimalTagError(null);
    } catch (err) {
      setAnimalTagError('Failed to load animal tags');
      console.error('Error fetching animal tags:', err);
    } finally {
      setLoadingAnimalTags(false);
    }
  }, []);

  // Fetch Comment Tags
  const fetchCommentTags = useCallback(async () => {
    try {
      setLoadingCommentTags(true);
      const [tagsRes, statsRes] = await Promise.all([
        commentTagsApi.getAll(),
        statisticsApi.getCommentTagStatistics()
      ]);
      
      setCommentTags(tagsRes.data);
      
      const statsMap: Record<number, CommentTagStatistics> = {};
      statsRes.data.forEach(stat => {
        statsMap[stat.tag_id] = stat;
      });
      setCommentTagStats(statsMap);
      setCommentTagError(null);
    } catch (err) {
      setCommentTagError('Failed to load comment tags');
      console.error('Error fetching comment tags:', err);
    } finally {
      setLoadingCommentTags(false);
    }
  }, []);

  useEffect(() => {
    fetchAnimalTags();
    fetchCommentTags();
  }, [fetchAnimalTags, fetchCommentTags]);

  // Animal Tag handlers
  const handleCreateAnimalTag = useCallback(() => {
    setEditingAnimalTag(null);
    setIsAnimalTagModalOpen(true);
  }, []);

  const handleEditAnimalTag = useCallback((tag: AnimalTag) => {
    setEditingAnimalTag(tag);
    setIsAnimalTagModalOpen(true);
  }, []);

  const handleCloseAnimalTagModal = useCallback(() => {
    setIsAnimalTagModalOpen(false);
    setEditingAnimalTag(null);
  }, []);

  const handleSubmitAnimalTag = useCallback(async (data: { name: string; category: string; color: string }) => {
    try {
      if (editingAnimalTag) {
        await apiClient.put(`/animal-tags/${editingAnimalTag.id}`, data);
      } else {
        await apiClient.post('/animal-tags', data);
      }
      await fetchAnimalTags();
      handleCloseAnimalTagModal();
    } catch (err) {
      console.error('Error saving animal tag:', err);
      throw err;
    }
  }, [editingAnimalTag, fetchAnimalTags, handleCloseAnimalTagModal]);

  const handleDeleteAnimalTag = useCallback(async (tagId: number) => {
    if (!window.confirm('Are you sure you want to delete this animal tag?')) return;
    
    try {
      await apiClient.delete(`/animal-tags/${tagId}`);
      await fetchAnimalTags();
    } catch (err) {
      console.error('Error deleting animal tag:', err);
      setAnimalTagError('Failed to delete tag');
    }
  }, [fetchAnimalTags]);

  // Comment Tag handlers
  const handleCreateCommentTag = useCallback(async (name: string, color: string) => {
    try {
      await commentTagsApi.create(name, color);
      await fetchCommentTags();
    } catch (err) {
      console.error('Error creating comment tag:', err);
      throw err;
    }
  }, [fetchCommentTags]);

  const handleDeleteCommentTag = useCallback(async (tagId: number) => {
    if (!window.confirm('Are you sure you want to delete this comment tag?')) return;
    
    try {
      await commentTagsApi.delete(tagId);
      await fetchCommentTags();
    } catch (err) {
      console.error('Error deleting comment tag:', err);
      setCommentTagError('Failed to delete tag');
    }
  }, [fetchCommentTags]);

  const behaviorTags = animalTags.filter(tag => tag.category === 'behavior');
  const walkerStatusTags = animalTags.filter(tag => tag.category === 'walker_status');

  const isLoading = loadingAnimalTags || loadingCommentTags;

  if (isLoading) {
    return <div className="admin-tags-page loading">Loading tags...</div>;
  }

  return (
    <div className="admin-tags-page">
      <div className="page-header">
        <div className="page-header-content">
          <h1>🏷️ Tag Management</h1>
          <p className="page-subtitle">
            Manage tags used to organize animals and comments. Tags help volunteers quickly understand important information.
          </p>
        </div>
      </div>

      {/* Section Tabs for mobile */}
      <div className="section-tabs" role="tablist">
        <button
          role="tab"
          aria-selected={activeSection === 'animal'}
          className={`section-tab ${activeSection === 'animal' ? 'active' : ''}`}
          onClick={() => setActiveSection('animal')}
        >
          🐾 Animal Tags
        </button>
        <button
          role="tab"
          aria-selected={activeSection === 'comment'}
          className={`section-tab ${activeSection === 'comment' ? 'active' : ''}`}
          onClick={() => setActiveSection('comment')}
        >
          💬 Comment Tags
        </button>
      </div>

      <div className="tags-layout">
        {/* Animal Tags Section */}
        <section 
          className={`tag-section animal-tags-section ${activeSection === 'animal' ? 'active' : ''}`}
          aria-label="Animal Tags"
        >
          <div className="section-header">
            <div className="section-title">
              <h2>🐾 Animal Tags</h2>
              <p className="section-description">
                Tags applied directly to animals. Displayed on animal cards and detail pages to help volunteers understand each animal's traits and required handling.
              </p>
            </div>
            <button className="btn-primary btn-sm" onClick={handleCreateAnimalTag}>
              + Add Animal Tag
            </button>
          </div>

          {animalTagError && <div className="error-message">{animalTagError}</div>}

          <div className="tag-categories">
            {/* Behavior Tags */}
            <div className="tag-category">
              <div className="category-header">
                <h3>🎭 Behavior Traits</h3>
                <span className="category-count">{behaviorTags.length} tags</span>
              </div>
              <p className="category-description">
                Describe how an animal behaves or acts. Examples: jumpy, mouthy, dog-reactive, leash-puller
              </p>
              <div className="tags-grid">
                {behaviorTags.length === 0 ? (
                  <p className="no-tags">No behavior tags yet. Add one to get started.</p>
                ) : (
                  behaviorTags.map(tag => (
                    <div key={tag.id} className="tag-card">
                      <div className="tag-display">
                        <span className="tag-badge" style={{ backgroundColor: tag.color }}>
                          {tag.name}
                        </span>
                      </div>
                      <div className="tag-actions">
                        <button className="btn-text" onClick={() => handleEditAnimalTag(tag)}>
                          Edit
                        </button>
                        <button className="btn-text btn-danger" onClick={() => handleDeleteAnimalTag(tag.id)}>
                          Delete
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Walker Status Tags */}
            <div className="tag-category">
              <div className="category-header">
                <h3>👟 Walker Levels</h3>
                <span className="category-count">{walkerStatusTags.length} tags</span>
              </div>
              <p className="category-description">
                Indicate required volunteer experience level. Examples: green (all volunteers), yellow (experienced), red (staff only)
              </p>
              <div className="tags-grid">
                {walkerStatusTags.length === 0 ? (
                  <p className="no-tags">No walker level tags yet. Add one to get started.</p>
                ) : (
                  walkerStatusTags.map(tag => (
                    <div key={tag.id} className="tag-card">
                      <div className="tag-display">
                        <span className="tag-badge" style={{ backgroundColor: tag.color }}>
                          {tag.name}
                        </span>
                      </div>
                      <div className="tag-actions">
                        <button className="btn-text" onClick={() => handleEditAnimalTag(tag)}>
                          Edit
                        </button>
                        <button className="btn-text btn-danger" onClick={() => handleDeleteAnimalTag(tag.id)}>
                          Delete
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </section>

        {/* Divider */}
        <div className="section-divider" aria-hidden="true">
          <div className="divider-line"></div>
        </div>

        {/* Comment Tags Section */}
        <section 
          className={`tag-section comment-tags-section ${activeSection === 'comment' ? 'active' : ''}`}
          aria-label="Comment Tags"
        >
          <div className="section-header">
            <div className="section-title">
              <h2>💬 Comment Tags</h2>
              <p className="section-description">
                Tags applied to volunteer comments on animals. Used to categorize and filter updates about medical info, behavior observations, and other notes.
              </p>
            </div>
            <button className="btn-primary btn-sm" onClick={() => setIsCommentTagModalOpen(true)}>
              + Add Comment Tag
            </button>
          </div>

          {commentTagError && <div className="error-message">{commentTagError}</div>}

          <div className="comment-tags-grid">
            {commentTags.length === 0 ? (
              <div className="empty-state">
                <p className="no-tags">No comment tags yet.</p>
                <p className="empty-hint">Create tags like "Medical Update", "Behavior Note", "Needs Attention" to help organize comments.</p>
              </div>
            ) : (
              commentTags.map((tag) => {
                const stats = commentTagStats[tag.id];
                return (
                  <div key={tag.id} className="comment-tag-card">
                    <div className="tag-header">
                      <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                        {tag.name}
                      </div>
                      <button
                        onClick={() => handleDeleteCommentTag(tag.id)}
                        className="btn-delete"
                        aria-label={`Delete ${tag.name} tag`}
                        title="Delete tag"
                      >
                        ✕
                      </button>
                    </div>
                    
                    {stats && stats.usage_count > 0 ? (
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
                    ) : (
                      <div className="tag-stats">
                        <span className="no-usage">Not used yet</span>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </section>
      </div>

      {/* Modals */}
      <AnimalTagFormModal
        isOpen={isAnimalTagModalOpen}
        editingTag={editingAnimalTag}
        onClose={handleCloseAnimalTagModal}
        onSubmit={handleSubmitAnimalTag}
      />

      <CommentTagFormModal
        isOpen={isCommentTagModalOpen}
        onClose={() => setIsCommentTagModalOpen(false)}
        onSubmit={handleCreateCommentTag}
      />
    </div>
  );
};

export default AdminAnimalTagsPage;
