import React from 'react';
import { Link } from 'react-router-dom';
import './GroupsPage.css';
import type { Group, Animal, GroupStatistics } from '../api/client';
import { groupsApi, animalsApi, statisticsApi } from '../api/client';
import { useToast } from '../hooks/useToast';

const GroupsPage: React.FC = () => {
  const toast = useToast();
  const [groups, setGroups] = React.useState<Group[]>([]);
  const [filteredGroups, setFilteredGroups] = React.useState<Group[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [statistics, setStatistics] = React.useState<Record<number, GroupStatistics>>({});

  // Filter and search state
  const [searchQuery, setSearchQuery] = React.useState('');
  const [sortBy, setSortBy] = React.useState<'name' | 'members' | 'animals' | 'activity'>('name');
  const [sortOrder, setSortOrder] = React.useState<'asc' | 'desc'>('asc');
  const [filterActivity, setFilterActivity] = React.useState<'all' | 'active' | 'inactive'>('all');

  // Create/Edit modal state
  const [showModal, setShowModal] = React.useState(false);
  const [editingGroup, setEditingGroup] = React.useState<Group | null>(null);
  const [modalData, setModalData] = React.useState({ 
    name: '', 
    description: '', 
    image_url: '', 
    hero_image_url: '',
    has_protocols: false,
    groupme_bot_id: '',
    groupme_enabled: false
  });
  const [modalLoading, setModalLoading] = React.useState(false);
  const [modalError, setModalError] = React.useState<string | null>(null);
  const [availableAnimals, setAvailableAnimals] = React.useState<Animal[]>([]);
  const [showAnimalSelector, setShowAnimalSelector] = React.useState(false);

  // Fetch groups and statistics
  const fetchGroups = React.useCallback(() => {
    setLoading(true);
    setError(null);
    
    Promise.all([
      groupsApi.getAll(),
      statisticsApi.getGroupStatistics()
    ])
      .then(([groupsRes, statsRes]) => {
        setGroups(groupsRes.data);
        
        // Create a map of group_id to statistics
        const statsMap: Record<number, GroupStatistics> = {};
        statsRes.data.data.forEach(stat => {
          statsMap[stat.group_id] = stat;
        });
        setStatistics(statsMap);
        
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

  // Filter and sort groups
  React.useEffect(() => {
    let filtered = [...groups];

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(group =>
        group.name.toLowerCase().includes(query) ||
        group.description.toLowerCase().includes(query)
      );
    }

    // Apply activity filter
    if (filterActivity !== 'all') {
      const now = new Date();
      const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
      
      filtered = filtered.filter(group => {
        const stats = statistics[group.id];
        const hasRecentActivity = stats?.last_activity && new Date(stats.last_activity) > dayAgo;
        
        return filterActivity === 'active' ? hasRecentActivity : !hasRecentActivity;
      });
    }

    // Apply sorting
    filtered.sort((a, b) => {
      let comparison = 0;
      const statsA = statistics[a.id];
      const statsB = statistics[b.id];

      switch (sortBy) {
        case 'name':
          comparison = a.name.localeCompare(b.name);
          break;
        case 'members': {
          const membersA = statsA?.user_count || 0;
          const membersB = statsB?.user_count || 0;
          comparison = membersB - membersA; // Most members first
          break;
        }
        case 'animals': {
          const animalsA = statsA?.animal_count || 0;
          const animalsB = statsB?.animal_count || 0;
          comparison = animalsB - animalsA; // Most animals first
          break;
        }
        case 'activity': {
          const activityA = statsA?.last_activity ? new Date(statsA.last_activity).getTime() : 0;
          const activityB = statsB?.last_activity ? new Date(statsB.last_activity).getTime() : 0;
          comparison = activityB - activityA; // Most recent first
          break;
        }
      }

      return sortOrder === 'asc' ? comparison : -comparison;
    });

    setFilteredGroups(filtered);
  }, [groups, searchQuery, sortBy, sortOrder, filterActivity, statistics]);

  // Open modal for creating a new group
  const openCreateModal = () => {
    setEditingGroup(null);
    setModalData({ name: '', description: '', image_url: '', hero_image_url: '', has_protocols: false, groupme_bot_id: '', groupme_enabled: false });
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
      hero_image_url: group.hero_image_url || '',
      has_protocols: group.has_protocols || false,
      groupme_bot_id: group.groupme_bot_id || '',
      groupme_enabled: group.groupme_enabled || false
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
    setModalData({ name: '', description: '', image_url: '', hero_image_url: '', has_protocols: false, groupme_bot_id: '', groupme_enabled: false });
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
          modalData.hero_image_url,
          modalData.has_protocols,
          modalData.groupme_bot_id,
          modalData.groupme_enabled
        );
      } else {
        // Create new group
        await groupsApi.create(
          modalData.name, 
          modalData.description, 
          modalData.image_url, 
          modalData.hero_image_url,
          modalData.has_protocols,
          modalData.groupme_bot_id,
          modalData.groupme_enabled
        );
      }
      fetchGroups();
      closeModal();
    } catch (err: unknown) {
      setModalError((err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to save group');
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
    } catch (err: unknown) {
      setError((err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to delete group');
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

      {/* Search and Filter Section */}
      <div className="groups-filters">
        <div className="filter-row">
          <div className="search-box">
            <svg className="search-icon" width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M9 17A8 8 0 1 0 9 1a8 8 0 0 0 0 16zM18 18l-4.35-4.35" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            <input
              type="text"
              className="search-input"
              placeholder="Search groups by name or description..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              aria-label="Search groups"
            />
            {searchQuery && (
              <button
                className="clear-search"
                onClick={() => setSearchQuery('')}
                aria-label="Clear search"
              >
                ×
              </button>
            )}
          </div>

          <select
            className="filter-select"
            value={filterActivity}
            onChange={(e) => setFilterActivity(e.target.value as 'all' | 'active' | 'inactive')}
            aria-label="Filter by activity"
          >
            <option value="all">All Groups</option>
            <option value="active">Active (24h)</option>
            <option value="inactive">Inactive</option>
          </select>

          <select
            className="filter-select"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as 'name' | 'members' | 'animals' | 'activity')}
            aria-label="Sort by"
          >
            <option value="name">Sort by Name</option>
            <option value="members">Sort by Members</option>
            <option value="animals">Sort by Animals</option>
            <option value="activity">Sort by Activity</option>
          </select>

          <button
            className="sort-order-btn"
            onClick={() => setSortOrder(order => order === 'asc' ? 'desc' : 'asc')}
            aria-label={`Sort order: ${sortOrder === 'asc' ? 'ascending' : 'descending'}`}
            title={sortOrder === 'asc' ? 'Ascending' : 'Descending'}
          >
            {sortOrder === 'asc' ? (
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M10 5v10M5 10l5-5 5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            ) : (
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M10 15V5M15 10l-5 5-5-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            )}
          </button>
        </div>
        
        <div className="filter-summary">
          Showing {filteredGroups.length} of {groups.length} groups
          {searchQuery && <span> matching "{searchQuery}"</span>}
        </div>
      </div>

      {loading ? (
        <div className="groups-loading">Loading groups…</div>
      ) : error ? (
        <div className="groups-error">{error}</div>
      ) : filteredGroups.length === 0 ? (
        <div className="groups-empty">
          {searchQuery || filterActivity !== 'all' 
            ? 'No groups match your filters.' 
            : 'No groups yet. Create one to get started!'}
        </div>
      ) : (
        <table className="groups-table">
          <thead>
            <tr>
              <th>Image</th>
              <th>Name</th>
              <th>Description</th>
              <th>Members</th>
              <th>Animals</th>
              <th>Last Activity</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredGroups.map(group => {
              const stats = statistics[group.id];
              return (
                <tr key={group.id}>
                  <td className="group-image">
                    {group.image_url ? (
                      <img src={group.image_url} alt={group.name} className="group-thumbnail" />
                    ) : (
                      <div className="no-image">No image</div>
                    )}
                  </td>
                  <td className="group-name">
                    <Link 
                      to={`/groups/${group.id}`}
                      className="group-name-link"
                      title={`View ${group.name} page`}
                    >
                      {group.name}
                    </Link>
                  </td>
                  <td className="group-description">{group.description || '—'}</td>
                  <td className="group-stat">
                    {stats ? (
                      <span className="stat-badge" title={`${stats.user_count} member${stats.user_count !== 1 ? 's' : ''}`}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                          <circle cx="9" cy="7" r="4"></circle>
                          <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                        </svg>
                        {stats.user_count}
                      </span>
                    ) : '—'}
                  </td>
                  <td className="group-stat">
                    {stats ? (
                      <span className="stat-badge" title={`${stats.animal_count} animal${stats.animal_count !== 1 ? 's' : ''}`}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <circle cx="11" cy="4" r="2"></circle>
                          <circle cx="18" cy="8" r="2"></circle>
                          <circle cx="20" cy="16" r="2"></circle>
                          <circle cx="4" cy="16" r="2"></circle>
                          <circle cx="4" cy="8" r="2"></circle>
                          <path d="M10.5 6.5 A 6 6 0 0 1 13.5 13.5 L 10.5 6.5 Z"></path>
                        </svg>
                        {stats.animal_count}
                      </span>
                    ) : '—'}
                  </td>
                  <td className="group-stat">
                    {stats?.last_activity ? (
                      <span className="last-activity" title={new Date(stats.last_activity).toLocaleString()}>
                        {formatRelativeTime(stats.last_activity)}
                      </span>
                    ) : (
                      <span className="no-activity">No activity</span>
                    )}
                  </td>
                  <td>
                    <div className="group-actions">
                      <Link
                        to={`/groups/${group.id}`}
                        className="group-action-btn view"
                        title="View group page"
                      >
                        View Group
                      </Link>
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
              );
            })}
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
                <label htmlFor="has_protocols" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer' }}>
                  <input
                    id="has_protocols"
                    name="has_protocols"
                    type="checkbox"
                    checked={modalData.has_protocols}
                    onChange={(e) => setModalData(d => ({ ...d, has_protocols: e.target.checked }))}
                    style={{ width: 'auto', cursor: 'pointer' }}
                  />
                  <span>Enable Protocols for this group</span>
                </label>
                <small style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginTop: '0.25rem', display: 'block', marginLeft: '1.75rem' }}>
                  Protocols allow you to document standardized procedures and workflows for this group.
                </small>
              </div>

              {/* GroupMe Integration Section */}
              <div className="form-group" style={{ borderTop: '1px solid var(--border-color)', paddingTop: '1.5rem', marginTop: '1.5rem' }}>
                <h4 style={{ marginBottom: '1rem', fontSize: '1.1rem', fontWeight: 600 }}>GroupMe Integration</h4>
                <label htmlFor="groupme_enabled" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', marginBottom: '1rem' }}>
                  <input
                    id="groupme_enabled"
                    name="groupme_enabled"
                    type="checkbox"
                    checked={modalData.groupme_enabled}
                    onChange={(e) => setModalData(d => ({ ...d, groupme_enabled: e.target.checked }))}
                    style={{ width: 'auto', cursor: 'pointer' }}
                  />
                  <span>Enable GroupMe integration for this group</span>
                </label>
                <small style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem', display: 'block', marginLeft: '1.75rem' }}>
                  Allow announcements to be automatically sent to this group's GroupMe chat.
                </small>
                
                {modalData.groupme_enabled && (
                  <div style={{ marginLeft: '1.75rem', marginTop: '1rem' }}>
                    <label htmlFor="groupme_bot_id">GroupMe Bot ID</label>
                    <input
                      id="groupme_bot_id"
                      name="groupme_bot_id"
                      type="text"
                      value={modalData.groupme_bot_id}
                      onChange={handleModalChange}
                      placeholder="Enter your GroupMe Bot ID"
                      required={modalData.groupme_enabled}
                      style={{ marginTop: '0.5rem' }}
                    />
                    <small style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginTop: '0.5rem', display: 'block' }}>
                      Create a bot at <a href="https://dev.groupme.com/bots" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--primary)' }}>dev.groupme.com/bots</a> and paste the Bot ID here.
                    </small>
                  </div>
                )}
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
                    } catch (err: unknown) {
                      console.error('Upload error:', err);
                      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to upload image. Please try again.';
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
                    } catch (err: unknown) {
                      console.error('Upload error:', err);
                      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to upload hero image. Please try again.';
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

export default GroupsPage;
