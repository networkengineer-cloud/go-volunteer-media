import React, { useEffect, useState, useCallback } from 'react';
import { animalTagsApi } from '../api/client';
import type { AnimalTag } from '../api/client';
import { useToast } from '../hooks/useToast';
import Button from '../components/Button';
import Modal from '../components/Modal';
import FormField from '../components/FormField';
import './AdminAnimalTagsPage.css';

const AdminAnimalTagsPage: React.FC = () => {
  const toast = useToast();
  const [tags, setTags] = useState<AnimalTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingTag, setEditingTag] = useState<AnimalTag | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    category: 'behavior',
    color: '#6b7280',
  });

  const loadTags = useCallback(async () => {
    try {
      const response = await animalTagsApi.getAll();
      setTags(response.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
      toast.showError('Failed to load tags');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  const handleOpenCreate = () => {
    setEditingTag(null);
    setFormData({ name: '', category: 'behavior', color: '#6b7280' });
    setShowModal(true);
  };

  const handleOpenEdit = (tag: AnimalTag) => {
    setEditingTag(tag);
    setFormData({ name: tag.name, category: tag.category, color: tag.color });
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.name.trim()) {
      toast.showError('Tag name is required');
      return;
    }

    try {
      if (editingTag) {
        await animalTagsApi.update(editingTag.id, formData);
        toast.showSuccess('Tag updated successfully');
      } else {
        await animalTagsApi.create(formData);
        toast.showSuccess('Tag created successfully');
      }
      setShowModal(false);
      loadTags();
    } catch (error: unknown) {
      const errorMsg = error.response?.data?.error || 'Failed to save tag';
      toast.showError(errorMsg);
    }
  };

  const handleDelete = async (tag: AnimalTag) => {
    if (!confirm(`Are you sure you want to delete the tag "${tag.name}"? This will remove it from all animals.`)) {
      return;
    }

    try {
      await animalTagsApi.delete(tag.id);
      toast.showSuccess('Tag deleted successfully');
      loadTags();
    } catch (error: unknown) {
      const errorMsg = error.response?.data?.error || 'Failed to delete tag';
      toast.showError(errorMsg);
    }
  };

  const behaviorTags = tags.filter(tag => tag.category === 'behavior');
  const walkerStatusTags = tags.filter(tag => tag.category === 'walker_status');

  if (loading) {
    return (
      <div className="admin-tags-page">
        <div className="page-header">
          <h1>Animal Tags Management</h1>
        </div>
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div className="admin-tags-page">
      <div className="page-header">
        <h1>Animal Tags Management</h1>
        <Button variant="primary" onClick={handleOpenCreate}>
          + Create New Tag
        </Button>
      </div>

      <div className="tags-sections">
        {/* Behavior Tags */}
        <section className="tags-section">
          <h2>Behavior Traits</h2>
          <div className="tags-grid">
            {behaviorTags.length === 0 ? (
              <p className="empty-message">No behavior tags yet</p>
            ) : (
              behaviorTags.map(tag => (
                <div key={tag.id} className="tag-card">
                  <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                    {tag.name}
                  </div>
                  <div className="tag-actions">
                    <button onClick={() => handleOpenEdit(tag)} className="btn-edit">
                      Edit
                    </button>
                    <button onClick={() => handleDelete(tag)} className="btn-delete">
                      Delete
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        {/* Walker Status Tags */}
        <section className="tags-section">
          <h2>Walker Status</h2>
          <div className="tags-grid">
            {walkerStatusTags.length === 0 ? (
              <p className="empty-message">No walker status tags yet</p>
            ) : (
              walkerStatusTags.map(tag => (
                <div key={tag.id} className="tag-card">
                  <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                    {tag.name}
                  </div>
                  <div className="tag-actions">
                    <button onClick={() => handleOpenEdit(tag)} className="btn-edit">
                      Edit
                    </button>
                    <button onClick={() => handleDelete(tag)} className="btn-delete">
                      Delete
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <Modal
          isOpen={showModal}
          onClose={() => setShowModal(false)}
          title={editingTag ? 'Edit Tag' : 'Create New Tag'}
        >
          <form onSubmit={handleSubmit} className="tag-form">
            <FormField
              label="Tag Name"
              id="name"
              type="text"
              value={formData.name}
              onChange={(value) => setFormData({ ...formData, name: value })}
              required
              helperText="The display name for this tag"
            />

            <div className="form-field">
              <label htmlFor="category" className="form-field__label">
                Category
              </label>
              <select
                id="category"
                value={formData.category}
                onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                className="form-field__input"
                required
              >
                <option value="behavior">Behavior Traits</option>
                <option value="walker_status">Walker Status</option>
              </select>
              <p className="form-field__helper">The category this tag belongs to</p>
            </div>

            <div className="form-field">
              <label htmlFor="color" className="form-field__label">
                Color
              </label>
              <div className="color-picker-container">
                <input
                  type="color"
                  id="color"
                  value={formData.color}
                  onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                  className="color-picker"
                />
                <input
                  type="text"
                  value={formData.color}
                  onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                  className="color-input"
                  placeholder="#6b7280"
                  pattern="^#[0-9A-Fa-f]{6}$"
                />
                <div className="color-preview" style={{ backgroundColor: formData.color }}>
                  Preview
                </div>
              </div>
              <p className="form-field__helper">The color used to display this tag</p>
            </div>

            <div className="form-actions">
              <Button type="submit" variant="primary">
                {editingTag ? 'Update Tag' : 'Create Tag'}
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => setShowModal(false)}
              >
                Cancel
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
};

export default AdminAnimalTagsPage;
