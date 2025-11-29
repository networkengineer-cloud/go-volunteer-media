import React, { useState, useEffect, useCallback } from 'react';
import apiClient from '../api/client';
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

// Isolated form component to prevent focus loss
interface TagFormModalProps {
  isOpen: boolean;
  editingTag: AnimalTag | null;
  onClose: () => void;
  onSubmit: (data: { name: string; category: string; color: string }) => Promise<void>;
}

const TagFormModal: React.FC<TagFormModalProps> = ({ isOpen, editingTag, onClose, onSubmit }) => {
  const [name, setName] = useState('');
  const [category, setCategory] = useState<'behavior' | 'walker_status'>('behavior');
  const [color, setColor] = useState('#6b7280');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Reset form when modal opens/closes or editing tag changes
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

  const colorPresets = [
    '#ef4444', '#f97316', '#f59e0b', '#eab308',
    '#84cc16', '#22c55e', '#10b981', '#14b8a6',
    '#06b6d4', '#0ea5e9', '#3b82f6', '#6366f1',
    '#8b5cf6', '#a855f7', '#d946ef', '#ec4899',
    '#f43f5e', '#6b7280', '#374151', '#1f2937',
  ];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={editingTag ? 'Edit Tag' : 'Create New Tag'}
    >
      <form onSubmit={handleSubmit} className="tag-form">
        <div className="form-group">
          <label htmlFor="tagName">Tag Name</label>
          <input
            type="text"
            id="tagName"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Enter tag name"
            required
            autoComplete="off"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label htmlFor="tagCategory">Category</label>
          <select
            id="tagCategory"
            value={category}
            onChange={(e) => setCategory(e.target.value as 'behavior' | 'walker_status')}
          >
            <option value="behavior">Behavior</option>
            <option value="walker_status">Walker Status</option>
          </select>
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
          <span
            className="tag-badge"
            style={{ backgroundColor: color }}
          >
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

const AdminAnimalTagsPage: React.FC = () => {
  const [tags, setTags] = useState<AnimalTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingTag, setEditingTag] = useState<AnimalTag | null>(null);

  const fetchTags = useCallback(async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/animal-tags');
      setTags(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load tags');
      console.error('Error fetching tags:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTags();
  }, [fetchTags]);

  const handleCreateClick = useCallback(() => {
    setEditingTag(null);
    setIsModalOpen(true);
  }, []);

  const handleEditClick = useCallback((tag: AnimalTag) => {
    setEditingTag(tag);
    setIsModalOpen(true);
  }, []);

  const handleCloseModal = useCallback(() => {
    setIsModalOpen(false);
    setEditingTag(null);
  }, []);

  const handleSubmit = useCallback(async (data: { name: string; category: string; color: string }) => {
    try {
      if (editingTag) {
        await apiClient.put(`/animal-tags/${editingTag.id}`, data);
      } else {
        await apiClient.post('/animal-tags', data);
      }
      await fetchTags();
      handleCloseModal();
    } catch (err) {
      console.error('Error saving tag:', err);
      throw err;
    }
  }, [editingTag, fetchTags, handleCloseModal]);

  const handleDeleteTag = useCallback(async (tagId: number) => {
    if (!window.confirm('Are you sure you want to delete this tag?')) return;
    
    try {
      await apiClient.delete(`/animal-tags/${tagId}`);
      await fetchTags();
    } catch (err) {
      console.error('Error deleting tag:', err);
      setError('Failed to delete tag');
    }
  }, [fetchTags]);

  const behaviorTags = tags.filter(tag => tag.category === 'behavior');
  const walkerStatusTags = tags.filter(tag => tag.category === 'walker_status');

  if (loading) {
    return <div className="admin-tags-page loading">Loading tags...</div>;
  }

  return (
    <div className="admin-tags-page">
      <div className="page-header">
        <h1>Animal Tags</h1>
        <button className="btn-primary" onClick={handleCreateClick}>
          + Create New Tag
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="tags-sections">
        <section className="tags-section">
          <h2>Behavior Tags</h2>
          <div className="tags-grid">
            {behaviorTags.length === 0 ? (
              <p className="no-tags">No behavior tags yet</p>
            ) : (
              behaviorTags.map(tag => (
                <div key={tag.id} className="tag-card">
                  <div className="tag-display">
                    <span
                      className="tag-badge"
                      style={{ backgroundColor: tag.color }}
                    >
                      {tag.name}
                    </span>
                  </div>
                  <div className="tag-actions">
                    <button
                      className="btn-text"
                      onClick={() => handleEditClick(tag)}
                    >
                      Edit
                    </button>
                    <button
                      className="btn-text btn-danger"
                      onClick={() => handleDeleteTag(tag.id)}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        <section className="tags-section">
          <h2>Walker Status Tags</h2>
          <div className="tags-grid">
            {walkerStatusTags.length === 0 ? (
              <p className="no-tags">No walker status tags yet</p>
            ) : (
              walkerStatusTags.map(tag => (
                <div key={tag.id} className="tag-card">
                  <div className="tag-display">
                    <span
                      className="tag-badge"
                      style={{ backgroundColor: tag.color }}
                    >
                      {tag.name}
                    </span>
                  </div>
                  <div className="tag-actions">
                    <button
                      className="btn-text"
                      onClick={() => handleEditClick(tag)}
                    >
                      Edit
                    </button>
                    <button
                      className="btn-text btn-danger"
                      onClick={() => handleDeleteTag(tag.id)}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>
      </div>

      <TagFormModal
        isOpen={isModalOpen}
        editingTag={editingTag}
        onClose={handleCloseModal}
        onSubmit={handleSubmit}
      />
    </div>
  );
};

export default AdminAnimalTagsPage;
