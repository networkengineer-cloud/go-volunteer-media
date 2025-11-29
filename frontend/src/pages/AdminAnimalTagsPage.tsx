import React, { useEffect, useState, useCallback } from 'react';
import { animalTagsApi } from '../api/client';
import type { AnimalTag } from '../api/client';
import Modal from '../components/Modal';
import SkeletonLoader from '../components/SkeletonLoader';
import './AdminAnimalTagsPage.css';

interface TagFormData {
  name: string;
  category: 'behavior' | 'walker_status';
  color: string;
  icon: string;
}

const EMOJI_PRESETS = [
  { icon: '✏️', label: 'Pencil' },
  { icon: '😊', label: 'Friendly' },
  { icon: '😳', label: 'Shy' },
  { icon: '⚠️', label: 'Warning' },
  { icon: '🛡️', label: 'Shield' },
  { icon: '🚶', label: 'Walking' },
  { icon: '👥', label: 'People' },
  { icon: '🎓', label: 'Education' },
  { icon: '🏥', label: 'Hospital' },
  { icon: '🏋️', label: 'Exercise' },
  { icon: '⚡', label: 'Energy' },
  { icon: '❤️', label: 'Heart' },
  { icon: '🐕', label: 'Dog' },
  { icon: '🦴', label: 'Bone' },
  { icon: '🎾', label: 'Ball' },
  { icon: '🚗', label: 'Car' },
];

const AdminAnimalTagsPage: React.FC = () => {
  const [tags, setTags] = useState<AnimalTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [showForm, setShowForm] = useState(false);
  const [editingTag, setEditingTag] = useState<AnimalTag | null>(null);
  const [formData, setFormData] = useState<TagFormData>({
    name: '',
    category: 'behavior',
    color: '#6b7280',
    icon: '✏️',
  });
  const [submitting, setSubmitting] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);

  useEffect(() => {
    loadTags();
  }, []);

  const loadTags = async () => {
    try {
      setLoading(true);
      setError('');
      const res = await animalTagsApi.getAll();
      setTags(res.data);
    } catch (err) {
      console.error('Failed to load tags:', err);
      setError('Failed to load tags. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  // Memoized form change handlers to prevent focus loss
  const handleNameChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({ ...prev, name: e.target.value }));
  }, []);

  const handleCategoryChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    setFormData(prev => ({ ...prev, category: e.target.value as 'behavior' | 'walker_status' }));
  }, []);

  const handleIconChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({ ...prev, icon: e.target.value }));
  }, []);

  const handleIconSelect = useCallback((icon: string) => {
    setFormData(prev => ({ ...prev, icon }));
  }, []);

  const handleColorChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData(prev => ({ ...prev, color: e.target.value }));
  }, []);

  const handleOpenForm = (tag?: AnimalTag) => {
    if (tag) {
      setEditingTag(tag);
      setFormData({
        name: tag.name,
        category: tag.category as 'behavior' | 'walker_status',
        color: tag.color,
        icon: tag.icon,
      });
    } else {
      setEditingTag(null);
      setFormData({
        name: '',
        category: 'behavior',
        color: '#6b7280',
        icon: '✏️',
      });
    }
    setShowForm(true);
  };

  const handleCloseForm = () => {
    setShowForm(false);
    setEditingTag(null);
  };

  const handleSubmitForm = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.name.trim()) {
      setError('Tag name is required');
      return;
    }

    try {
      setSubmitting(true);
      setError('');

      if (editingTag) {
        await animalTagsApi.update(editingTag.id, formData);
        setSuccessMessage(`Tag "${formData.name}" updated successfully`);
      } else {
        await animalTagsApi.create(formData);
        setSuccessMessage(`Tag "${formData.name}" created successfully`);
      }

      handleCloseForm();
      await loadTags();
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      console.error('Failed to save tag:', err);
      setError('Failed to save tag. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteTag = async (tagId: number) => {
    try {
      setSubmitting(true);
      setError('');
      const tagName = tags.find(t => t.id === tagId)?.name;
      await animalTagsApi.delete(tagId);
      setSuccessMessage(`Tag "${tagName}" deleted successfully`);
      setDeleteConfirm(null);
      await loadTags();
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      console.error('Failed to delete tag:', err);
      setError('Failed to delete tag. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const behaviorTags = tags.filter(t => t.category === 'behavior');
  const walkerStatusTags = tags.filter(t => t.category === 'walker_status');

  if (loading) {
    return (
      <div className="admin-page">
        <div className="admin-header">
          <h1>Tag Management</h1>
        </div>
        <div className="skeleton-section">
          <SkeletonLoader variant="card" count={4} />
        </div>
      </div>
    );
  }

  return (
    <div className="admin-page">
      <div className="admin-header">
        <h1>Animal Tag Management</h1>
        <p className="subtitle">Create and manage tags for animal behavior and walker status</p>
      </div>

      {successMessage && (
        <div className="success-message">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22 4 12 14.01 9 11.01" />
          </svg>
          {successMessage}
        </div>
      )}

      {error && (
        <div className="error-message">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          {error}
        </div>
      )}

      <div className="action-bar">
        <button
          className="btn-primary"
          onClick={() => handleOpenForm()}
          disabled={submitting}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          Create New Tag
        </button>
      </div>

      <section className="tags-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 11H7c-1.1 0-2 .9-2 2v7c0 1.1.9 2 2 2h7c1.1 0 2-.9 2-2v-2m-7-4h10c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2h-10c-1.1 0-2 .9-2 2v7c0 1.1.9 2 2 2z" />
          </svg>
          Behavior Tags
        </h2>
        {behaviorTags.length === 0 ? (
          <p className="empty-section">No behavior tags yet</p>
        ) : (
          <div className="tags-grid">
            {behaviorTags.map(tag => (
              <div key={tag.id} className="tag-card">
                <div className="tag-header">
                  <div className="tag-display">
                    <span className="tag-icon">{tag.icon}</span>
                    <span className="tag-name">{tag.name}</span>
                  </div>
                  <div className="tag-actions">
                    <button
                      className="btn-icon edit"
                      onClick={() => handleOpenForm(tag)}
                      disabled={submitting}
                    >
                      ✏️
                    </button>
                    <button
                      className="btn-icon delete"
                      onClick={() => setDeleteConfirm(tag.id)}
                      disabled={submitting}
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                <div className="tag-color" style={{ backgroundColor: tag.color }} />
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="tags-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          Walker Status Tags
        </h2>
        {walkerStatusTags.length === 0 ? (
          <p className="empty-section">No walker status tags yet</p>
        ) : (
          <div className="tags-grid">
            {walkerStatusTags.map(tag => (
              <div key={tag.id} className="tag-card">
                <div className="tag-header">
                  <div className="tag-display">
                    <span className="tag-icon">{tag.icon}</span>
                    <span className="tag-name">{tag.name}</span>
                  </div>
                  <div className="tag-actions">
                    <button
                      className="btn-icon edit"
                      onClick={() => handleOpenForm(tag)}
                      disabled={submitting}
                    >
                      ✏️
                    </button>
                    <button
                      className="btn-icon delete"
                      onClick={() => setDeleteConfirm(tag.id)}
                      disabled={submitting}
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                <div className="tag-color" style={{ backgroundColor: tag.color }} />
              </div>
            ))}
          </div>
        )}
      </section>

      <Modal
        isOpen={showForm}
        onClose={handleCloseForm}
        title={editingTag ? 'Edit Tag' : 'Create New Tag'}
        size="medium"
      >
        <form onSubmit={handleSubmitForm} className="tag-form">
          <div className="form-group">
            <label htmlFor="tag-name">Tag Name *</label>
            <input
              id="tag-name"
              type="text"
              value={formData.name}
              onChange={handleNameChange}
              placeholder="e.g., Friendly, Needs Walker"
              required
              disabled={submitting}
            />
          </div>

          <div className="form-group">
            <label htmlFor="tag-category">Category *</label>
            <select
              id="tag-category"
              value={formData.category}
              onChange={handleCategoryChange}
              disabled={submitting}
            >
              <option value="behavior">Behavior</option>
              <option value="walker_status">Walker Status</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="tag-icon">Icon *</label>
            <div className="icon-selector">
              <input
                id="tag-icon"
                type="text"
                value={formData.icon}
                onChange={handleIconChange}
                placeholder="Enter emoji"
                maxLength={10}
                disabled={submitting}
              />
              <div className="emoji-presets">
                {EMOJI_PRESETS.map((preset) => (
                  <button
                    key={preset.icon}
                    type="button"
                    className={`emoji-button ${formData.icon === preset.icon ? 'selected' : ''}`}
                    onClick={() => handleIconSelect(preset.icon)}
                    disabled={submitting}
                  >
                    {preset.icon}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="tag-color">Color *</label>
            <div className="color-picker">
              <input
                id="tag-color"
                type="color"
                value={formData.color}
                onChange={handleColorChange}
                disabled={submitting}
              />
              <span className="color-value">{formData.color}</span>
            </div>
          </div>

          <div className="form-preview">
            <div className="preview-label">Preview:</div>
            <span className="preview-tag" style={{ backgroundColor: formData.color }}>
              {formData.icon} {formData.name || 'Your Tag Name'}
            </span>
          </div>

          <div className="form-actions">
            <button type="button" className="btn-secondary" onClick={handleCloseForm} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={submitting}>
              {submitting ? 'Saving...' : editingTag ? 'Update Tag' : 'Create Tag'}
            </button>
          </div>
        </form>
      </Modal>

      {deleteConfirm !== null && (
        <Modal
          isOpen={deleteConfirm !== null}
          onClose={() => setDeleteConfirm(null)}
          title="Delete Tag"
          size="small"
        >
          <div className="confirmation-content">
            <p>Are you sure you want to delete this tag? This action cannot be undone.</p>
            <div className="confirmation-actions">
              <button className="btn-secondary" onClick={() => setDeleteConfirm(null)} disabled={submitting}>
                Cancel
              </button>
              <button className="btn-danger" onClick={() => handleDeleteTag(deleteConfirm)} disabled={submitting}>
                {submitting ? 'Deleting...' : 'Delete Tag'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default AdminAnimalTagsPage;
