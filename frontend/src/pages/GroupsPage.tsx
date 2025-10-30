import React from 'react';
import './GroupsPage.css';
import type { Group } from '../api/client';
import { groupsApi } from '../api/client';

const GroupsPage: React.FC = () => {
  const [groups, setGroups] = React.useState<Group[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Create/Edit modal state
  const [showModal, setShowModal] = React.useState(false);
  const [editingGroup, setEditingGroup] = React.useState<Group | null>(null);
  const [modalData, setModalData] = React.useState({ name: '', description: '', image_url: '' });
  const [modalLoading, setModalLoading] = React.useState(false);
  const [modalError, setModalError] = React.useState<string | null>(null);

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
    setModalData({ name: '', description: '', image_url: '' });
    setModalError(null);
    setShowModal(true);
  };

  // Open modal for editing an existing group
  const openEditModal = (group: Group) => {
    setEditingGroup(group);
    setModalData({ name: group.name, description: group.description, image_url: group.image_url || '' });
    setModalError(null);
    setShowModal(true);
  };

  // Close modal
  const closeModal = () => {
    setShowModal(false);
    setEditingGroup(null);
    setModalData({ name: '', description: '', image_url: '' });
    setModalError(null);
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
        await groupsApi.update(editingGroup.id, modalData.name, modalData.description, modalData.image_url);
      } else {
        // Create new group
        await groupsApi.create(modalData.name, modalData.description, modalData.image_url);
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
                <label htmlFor="image_url">Image URL</label>
                <input
                  id="image_url"
                  name="image_url"
                  type="text"
                  value={modalData.image_url}
                  onChange={handleModalChange}
                  placeholder="Paste an image URL or upload a file"
                />
                <label htmlFor="image_upload" className="upload-label">Or upload image</label>
                <input
                  id="image_upload"
                  type="file"
                  accept=".jpg,.jpeg,.png,.gif"
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    setModalLoading(true);
                    try {
                      const res = await groupsApi.uploadImage(file);
                      setModalData({ ...modalData, image_url: res.data.url });
                      alert('Image uploaded successfully!');
                    } catch (err: any) {
                      console.error('Upload error:', err);
                      const errorMsg = err.response?.data?.error || err.message || 'Failed to upload image';
                      alert('Failed to upload image: ' + errorMsg);
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
                    <img src={modalData.image_url} alt="Group Preview" />
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
