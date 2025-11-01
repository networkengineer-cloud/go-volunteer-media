import React from 'react';
import './GroupsPage.css';
import type { Group, Animal } from '../api/client';
import { groupsApi, animalsApi } from '../api/client';
import { useToast } from '../contexts/ToastContext';

const GroupsPage: React.FC = () => {
  const toast = useToast();
  const [groups, setGroups] = React.useState<Group[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Create/Edit modal state
  const [showModal, setShowModal] = React.useState(false);
  const [editingGroup, setEditingGroup] = React.useState<Group | null>(null);
  const [modalData, setModalData] = React.useState({ 
    name: '', 
    description: '', 
    image_url: '', 
    hero_image_url: '' 
  });
  const [modalLoading, setModalLoading] = React.useState(false);
  const [modalError, setModalError] = React.useState<string | null>(null);
  const [availableAnimals, setAvailableAnimals] = React.useState<Animal[]>([]);
  const [showAnimalSelector, setShowAnimalSelector] = React.useState(false);

  // Fetch groups
  const fetchGroups = React.useCallback(() => {
    setLoading(true);
    setError(null);
    groupsApi.getAll()
      .then(res => {
        setGroups(res.data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.response?.data?.error || 'Failed to fetch groups');
        setLoading(false);
      });
  }, []);

  React.useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  // Open modal for creating a new group
  const openCreateModal = () => {
    setEditingGroup(null);
    setModalData({ name: '', description: '', image_url: '', hero_image_url: '' });
    setModalError(null);
    setShowModal(true);
  };

  // Open modal for editing an existing group
  const openEditModal = async (group: Group) => {
    setEditingGroup(group);
    setModalData({ 
      name: group.name, 
      description: group.description, 
      image_url: group.image_url || '', 
      hero_image_url: group.hero_image_url || '' 
    });
    setModalError(null);
    setShowModal(true);
    
    // Fetch animals for this group to allow hero image selection
    try {
      const animalsRes = await animalsApi.getAll(group.id, 'all');
      setAvailableAnimals(animalsRes.data.filter(a => a.image_url));
    } catch (err) {
      console.error('Failed to fetch animals:', err);
      setAvailableAnimals([]);
    }
  };

  // Close modal
  const closeModal = () => {
    setShowModal(false);
    setEditingGroup(null);
    setModalData({ name: '', description: '', image_url: '', hero_image_url: '' });
    setModalError(null);
    setAvailableAnimals([]);
    setShowAnimalSelector(false);
  };

  // Handle form input changes
  const handleModalChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setModalData(d => ({ ...d, [name]: value }));
  };

  // Submit create or update
  const handleModalSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setModalLoading(true);
    setModalError(null);

    try {
      if (editingGroup) {
        // Update existing group
        await groupsApi.update(
          editingGroup.id, 
          modalData.name, 
          modalData.description, 
          modalData.image_url, 
          modalData.hero_image_url
        );
      } else {
        // Create new group
        await groupsApi.create(
          modalData.name, 
          modalData.description, 
          modalData.image_url, 
          modalData.hero_image_url
        );
      }
      fetchGroups();
      closeModal();
    } catch (err: any) {
      setModalError(err.response?.data?.error || 'Failed to save group');
    } finally {
      setModalLoading(false);
    }
  };

  // Delete group
  const handleDelete = async (group: Group) => {
    if (!window.confirm(`Delete group "${group.name}"? This cannot be undone and will affect all associated data.`)) {
      return;
    }

    try {
      await groupsApi.delete(group.id);
      fetchGroups();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete group');
    }
  };

  return (
    <div className="groups-page">
      <h1>Manage Groups</h1>
      <div className="groups-header">
        <button className="group-action-btn" onClick={openCreateModal}>
          Add Group
        </button>
      </div>

      {loading ? (
        <div className="groups-loading">Loading groups…</div>
      ) : error ? (
        <div className="groups-error">{error}</div>
      ) : groups.length === 0 ? (
        <div className="groups-empty">No groups yet. Create one to get started!</div>
      ) : (
        <table className="groups-table">
          <thead>
            <tr>
              <th>Image</th>
              <th>Name</th>
              <th>Description</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {groups.map(group => (
              <tr key={group.id}>
                <td className="group-image">
                  {group.image_url ? (
                    <img src={group.image_url} alt={group.name} className="group-thumbnail" />
                  ) : (
                    <div className="no-image">No image</div>
                  )}
                </td>
                <td className="group-name">{group.name}</td>
                <td className="group-description">{group.description || '—'}</td>
                <td>
                  <div className="group-actions">
                    <button
                      className="group-action-btn"
                      title="Edit group"
                      onClick={() => openEditModal(group)}
                    >
                      Edit
                    </button>
                    <button
                      className="group-action-btn danger"
                      title="Delete group"
                      onClick={() => handleDelete(group)}
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="group-modal-backdrop" onClick={closeModal}>
          <div className="group-modal" onClick={e => e.stopPropagation()}>
            <h2>{editingGroup ? 'Edit Group' : 'Create New Group'}</h2>
            <form onSubmit={handleModalSubmit}>
              <div className="form-group">
                <label htmlFor="name">
                  Name <span className="required">*</span>
                </label>
                <input
                  id="name"
                  name="name"
                  type="text"
                  value={modalData.name}
                  onChange={handleModalChange}
                  required
                  minLength={2}
                  maxLength={100}
                  placeholder="Enter group name"
                  autoFocus
                />
              </div>
              <div className="form-group">
                <label htmlFor="description">Description</label>
                <textarea
                  id="description"
                  name="description"
                  value={modalData.description}
                  onChange={handleModalChange}
                  maxLength={500}
                  placeholder="Enter group description (optional)"
                  rows={4}
                />
              </div>
              <div className="form-group">
                <label htmlFor="image_upload">Group Card Image</label>
                <input
                  id="image_upload"
                  type="file"
                  accept=".jpg,.jpeg,.png,.gif"
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    
                    // Validate file type
                    if (!file.type.startsWith('image/')) {
                      toast.showError('Please select an image file (JPG, PNG, or GIF)');
                      return;
                    }
                    
                    // Validate file size (max 10MB)
                    if (file.size > 10 * 1024 * 1024) {
                      toast.showError(`Image size must be under 10MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
                      return;
                    }
                    
                    setModalLoading(true);
                    try {
                      const res = await groupsApi.uploadImage(file);
                      setModalData({ ...modalData, image_url: res.data.url });
                      toast.showSuccess('Image uploaded successfully!');
                    } catch (err: any) {
                      console.error('Upload error:', err);
                      const errorMsg = err.response?.data?.error || 'Failed to upload image. Please try again.';
                      toast.showError(errorMsg);
                    } finally {
                      setModalLoading(false);
                      // Clear the file input
                      e.target.value = '';
                    }
                  }}
                />
                {modalData.image_url && (
                  <div className="image-preview">
                    <label>Preview:</label>
                    <img src={modalData.image_url} alt="Group Card Preview" />
                  </div>
                )}
              </div>
              <div className="form-group">
                <label htmlFor="hero_image_upload">Hero Image (Group Page Banner)</label>
                <input
                  id="hero_image_upload"
                  type="file"
                  accept=".jpg,.jpeg,.png,.gif"
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    
                    // Validate file type
                    if (!file.type.startsWith('image/')) {
                      toast.showError('Please select an image file (JPG, PNG, or GIF)');
                      return;
                    }
                    
                    // Validate file size (max 10MB)
                    if (file.size > 10 * 1024 * 1024) {
                      toast.showError(`Image size must be under 10MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
                      return;
                    }
                    
                    setModalLoading(true);
                    try {
                      const res = await groupsApi.uploadImage(file);
                      setModalData({ ...modalData, hero_image_url: res.data.url });
                      toast.showSuccess('Hero image uploaded successfully!');
                    } catch (err: any) {
                      console.error('Upload error:', err);
                      const errorMsg = err.response?.data?.error || 'Failed to upload hero image. Please try again.';
                      toast.showError(errorMsg);
                    } finally {
                      setModalLoading(false);
                      // Clear the file input
                      e.target.value = '';
                    }
                  }}
                />
                {editingGroup && availableAnimals.length > 0 && (
                  <div style={{ marginTop: '0.5rem' }}>
                    <button
                      type="button"
                      className="group-action-btn secondary"
                      onClick={() => setShowAnimalSelector(!showAnimalSelector)}
                    >
                      {showAnimalSelector ? 'Hide' : 'Select from Animal Images'}
                    </button>
                  </div>
                )}
                {showAnimalSelector && availableAnimals.length > 0 && (
                  <div className="animal-selector">
                    <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', marginBottom: '0.75rem' }}>
                      Click an animal image to use it as the hero image:
                    </p>
                    <div className="animal-images-grid">
                      {availableAnimals.map((animal) => (
                        <div
                          key={animal.id}
                          className="animal-image-option"
                          onClick={() => {
                            setModalData({ ...modalData, hero_image_url: animal.image_url });
                            setShowAnimalSelector(false);
                            toast.showSuccess(`Selected ${animal.name}'s image as hero image`);
                          }}
                        >
                          <img src={animal.image_url} alt={animal.name} />
                          <span>{animal.name}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {modalData.hero_image_url && (
                  <div className="image-preview">
                    <label>Preview:</label>
                    <img src={modalData.hero_image_url} alt="Hero Image Preview" />
                  </div>
                )}
              </div>
              {modalError && <div className="groups-error">{modalError}</div>}
              <div className="modal-actions">
                <button
                  type="button"
                  className="group-action-btn secondary"
                  onClick={closeModal}
                  disabled={modalLoading}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="group-action-btn"
                  disabled={modalLoading}
                >
                  {modalLoading ? 'Saving…' : editingGroup ? 'Update' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default GroupsPage;
