import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { commentTagsApi, animalTagsApi, groupsApi, type CommentTag, type Group, type AnimalTag as ApiAnimalTag, type UserSkillTag } from '../api/client';
import Modal from '../components/Modal';
import ConfirmDialog from '../components/ConfirmDialog';
import { useConfirmDialog } from '../hooks/useConfirmDialog';
import SkeletonLoader from '../components/SkeletonLoader';
import './AdminAnimalTagsPage.css';

type AnimalTag = ApiAnimalTag;

// Color presets used across all tag forms
const colorPresets = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308',
  '#84cc16', '#22c55e', '#10b981', '#14b8a6',
  '#06b6d4', '#0ea5e9', '#3b82f6', '#6366f1',
  '#8b5cf6', '#a855f7', '#d946ef', '#ec4899',
  '#f43f5e', '#6b7280', '#374151', '#1f2937',
];

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
  // Groups state
  const [groups, setGroups] = useState<Group[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
  const [loadingGroups, setLoadingGroups] = useState(true);

  // Animal Tags state
  const [animalTags, setAnimalTags] = useState<AnimalTag[]>([]);
  const [loadingAnimalTags, setLoadingAnimalTags] = useState(false);
  const [animalTagError, setAnimalTagError] = useState<string | null>(null);
  const [isAnimalTagModalOpen, setIsAnimalTagModalOpen] = useState(false);
  const [editingAnimalTag, setEditingAnimalTag] = useState<AnimalTag | null>(null);

  // Comment Tags state
  const [commentTags, setCommentTags] = useState<CommentTag[]>([]);
  const [loadingCommentTags, setLoadingCommentTags] = useState(false);
  const [commentTagError, setCommentTagError] = useState<string | null>(null);
  const [isCommentTagModalOpen, setIsCommentTagModalOpen] = useState(false);

  // Member (Skill) Tags state
  const [skillTags, setSkillTags] = useState<UserSkillTag[]>([]);
  const [loadingSkillTags, setLoadingSkillTags] = useState(false);
  const [skillTagError, setSkillTagError] = useState<string | null>(null);
  const [newSkillTagName, setNewSkillTagName] = useState('');
  const [newSkillTagColor, setNewSkillTagColor] = useState('#6b7280');
  const [editingSkillTag, setEditingSkillTag] = useState<UserSkillTag | null>(null);
  const [editSkillTagName, setEditSkillTagName] = useState('');
  const [editSkillTagColor, setEditSkillTagColor] = useState('#6b7280');

  // Active section for mobile
  const [activeSection, setActiveSection] = useState<'animal' | 'comment' | 'member'>('animal');
  const { confirmDialog, openConfirmDialog, closeConfirmDialog } = useConfirmDialog();

  // Fetch Groups
  const fetchGroups = useCallback(async () => {
    try {
      setLoadingGroups(true);
      const response = await groupsApi.getAll();
      setGroups(response.data);
      // Auto-select first group if none selected
      if (response.data.length > 0 && !selectedGroupId) {
        setSelectedGroupId(response.data[0].id);
      }
    } catch (err) {
      console.error('Error fetching groups:', err);
    } finally {
      setLoadingGroups(false);
    }
  }, [selectedGroupId]);

  // Fetch Animal Tags for selected group
  const fetchAnimalTags = useCallback(async () => {
    if (!selectedGroupId) return;
    try {
      setLoadingAnimalTags(true);
      const response = await animalTagsApi.getAll(selectedGroupId);
      setAnimalTags(response.data);
      setAnimalTagError(null);
    } catch (err) {
      setAnimalTagError('Failed to load animal tags');
      console.error('Error fetching animal tags:', err);
    } finally {
      setLoadingAnimalTags(false);
    }
  }, [selectedGroupId]);

  // Fetch Comment Tags for selected group
  const fetchCommentTags = useCallback(async () => {
    if (!selectedGroupId) return;
    try {
      setLoadingCommentTags(true);
      const tagsRes = await commentTagsApi.getAll(selectedGroupId);
      setCommentTags(tagsRes.data);
      setCommentTagError(null);
    } catch (err) {
      setCommentTagError('Failed to load comment tags');
      console.error('Error fetching comment tags:', err);
    } finally {
      setLoadingCommentTags(false);
    }
  }, [selectedGroupId]);

  // Fetch Member Skill Tags for selected group
  const fetchSkillTags = useCallback(async () => {
    if (!selectedGroupId) return;
    try {
      setLoadingSkillTags(true);
      const res = await groupsApi.getUserSkillTags(selectedGroupId);
      setSkillTags(res.data);
      setSkillTagError(null);
    } catch (err) {
      setSkillTagError('Failed to load member tags');
      console.error('Error fetching skill tags:', err);
    } finally {
      setLoadingSkillTags(false);
    }
  }, [selectedGroupId]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  useEffect(() => {
    if (selectedGroupId) {
      fetchAnimalTags();
      fetchCommentTags();
      fetchSkillTags();
    }
  }, [selectedGroupId, fetchAnimalTags, fetchCommentTags, fetchSkillTags]);

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
    if (!selectedGroupId) return;
    try {
      if (editingAnimalTag) {
        await animalTagsApi.update(selectedGroupId, editingAnimalTag.id, data);
      } else {
        await animalTagsApi.create(selectedGroupId, data);
      }
      await fetchAnimalTags();
      handleCloseAnimalTagModal();
    } catch (err) {
      console.error('Error saving animal tag:', err);
      throw err;
    }
  }, [selectedGroupId, editingAnimalTag, fetchAnimalTags, handleCloseAnimalTagModal]);

  const handleDeleteAnimalTag = useCallback((tagId: number) => {
    if (!selectedGroupId) return;
    openConfirmDialog(
      'Delete Animal Tag',
      'Are you sure you want to delete this animal tag?',
      async () => {
        try {
          await animalTagsApi.delete(selectedGroupId, tagId);
          await fetchAnimalTags();
        } catch (err) {
          console.error('Error deleting animal tag:', err);
          setAnimalTagError('Failed to delete tag');
        }
      },
    );
  }, [selectedGroupId, fetchAnimalTags, openConfirmDialog]);

  // Comment Tag handlers
  const handleCreateCommentTag = useCallback(async (name: string, color: string) => {
    if (!selectedGroupId) return;
    try {
      await commentTagsApi.create(selectedGroupId, name, color);
      await fetchCommentTags();
    } catch (err) {
      console.error('Error creating comment tag:', err);
      throw err;
    }
  }, [selectedGroupId, fetchCommentTags]);

  const handleDeleteCommentTag = useCallback((tagId: number) => {
    if (!selectedGroupId) return;
    openConfirmDialog(
      'Delete Comment Tag',
      'Are you sure you want to delete this comment tag?',
      async () => {
        try {
          await commentTagsApi.delete(selectedGroupId, tagId);
          await fetchCommentTags();
        } catch (err) {
          console.error('Error deleting comment tag:', err);
          setCommentTagError('Failed to delete tag');
        }
      },
    );
  }, [selectedGroupId, fetchCommentTags, openConfirmDialog]);

  // Skill Tag handlers
  const handleCreateSkillTag = useCallback(async () => {
    if (!selectedGroupId || !newSkillTagName.trim()) return;
    try {
      await groupsApi.createUserSkillTag(selectedGroupId, newSkillTagName.trim(), newSkillTagColor);
      setNewSkillTagName('');
      setNewSkillTagColor('#6b7280');
      await fetchSkillTags();
    } catch (err) {
      console.error('Error creating skill tag:', err);
      setSkillTagError('Failed to create member tag');
    }
  }, [selectedGroupId, newSkillTagName, newSkillTagColor, fetchSkillTags]);

  const handleUpdateSkillTag = useCallback(async () => {
    if (!selectedGroupId || !editingSkillTag || !editSkillTagName.trim()) return;
    try {
      await groupsApi.updateUserSkillTag(selectedGroupId, editingSkillTag.id, editSkillTagName.trim(), editSkillTagColor);
      setEditingSkillTag(null);
      setEditSkillTagName('');
      setEditSkillTagColor('#6b7280');
      await fetchSkillTags();
    } catch (err) {
      console.error('Error updating skill tag:', err);
      setSkillTagError('Failed to update member tag');
    }
  }, [selectedGroupId, editingSkillTag, editSkillTagName, editSkillTagColor, fetchSkillTags]);

  const handleDeleteSkillTag = useCallback((tagId: number, tagName: string) => {
    if (!selectedGroupId) return;
    openConfirmDialog(
      'Delete Member Tag',
      `Are you sure you want to delete the "${tagName}" tag? It will be removed from all members.`,
      async () => {
        try {
          await groupsApi.deleteUserSkillTag(selectedGroupId, tagId);
          await fetchSkillTags();
        } catch (err) {
          console.error('Error deleting skill tag:', err);
          setSkillTagError('Failed to delete member tag');
        }
      },
    );
  }, [selectedGroupId, fetchSkillTags, openConfirmDialog]);

  const behaviorTags = animalTags.filter(tag => tag.category === 'behavior');
  const walkerStatusTags = animalTags.filter(tag => tag.category === 'walker_status');

  const isLoading = loadingGroups || loadingAnimalTags || loadingCommentTags || loadingSkillTags;

  if (loadingGroups) {
    return (
      <div className="admin-tags-page">
        <SkeletonLoader variant="rectangular" height="40px" count={3} />
      </div>
    );
  }

  if (groups.length === 0) {
    return (
      <div className="admin-tags-page">
        <div className="page-header">
          <div className="page-header-content">
            <h1>🏷️ Tag Management</h1>
            <p className="page-subtitle">No groups available. Tags are group-specific.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="admin-tags-page">
      {/* Breadcrumb Navigation */}
      {selectedGroupId && (
        <nav className="breadcrumb" aria-label="Breadcrumb">
          <Link to="/" className="breadcrumb-link">Home</Link>
          <span className="breadcrumb-separator" aria-hidden="true">/</span>
          <Link to={`/groups/${selectedGroupId}`} className="breadcrumb-link">
            {groups.find(g => g.id === selectedGroupId)?.name || 'Group'}
          </Link>
          <span className="breadcrumb-separator" aria-hidden="true">/</span>
          <span className="breadcrumb-current" aria-current="page">Tag Management</span>
        </nav>
      )}

      <div className="page-header">
        <div className="page-header-content">
          <h1>🏷️ Tag Management</h1>
          <p className="page-subtitle">
            Manage tags used to organize animals and comments. Tags are group-specific and help volunteers quickly understand important information.
          </p>
        </div>
      </div>

      {/* Group Selector */}
      <div className="group-selector">
        <label htmlFor="group-select">Select Group:</label>
        <select
          id="group-select"
          value={selectedGroupId || ''}
          onChange={(e) => setSelectedGroupId(Number(e.target.value))}
          className="group-select"
        >
          {groups.map((group) => (
            <option key={group.id} value={group.id}>
              {group.name}
            </option>
          ))}
        </select>
      </div>

      {isLoading && <div className="loading-indicator">Loading tags for selected group...</div>}

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
        <button
          role="tab"
          aria-selected={activeSection === 'member'}
          className={`section-tab ${activeSection === 'member' ? 'active' : ''}`}
          onClick={() => setActiveSection('member')}
        >
          👥 Member Tags
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
              commentTags.map((tag) => (
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
                </div>
              ))
            )}
          </div>
        </section>

        {/* Divider */}
        <div className="section-divider" aria-hidden="true">
          <div className="divider-line"></div>
        </div>

        {/* Member Tags Section */}
        <section
          className={`tag-section member-tags-section ${activeSection === 'member' ? 'active' : ''}`}
          aria-label="Member Tags"
        >
          <div className="section-header">
            <div className="section-title">
              <h2>👥 Member Tags</h2>
              <p className="section-description">
                Tags assigned to volunteers to indicate experience level or role. Create tags here, then assign them to members on the group page.
              </p>
            </div>
          </div>

          {skillTagError && <div className="error-message">{skillTagError}</div>}

          <div className="comment-tags-grid">
            {skillTags.length === 0 ? (
              <div className="empty-state">
                <p className="no-tags">No member tags yet.</p>
                <p className="empty-hint">Create tags like "Beginner", "Experienced", "Dog Walker" to categorize volunteers.</p>
              </div>
            ) : (
              skillTags.map((tag) => (
                <div key={tag.id} className="comment-tag-card">
                  <div className="tag-header">
                    {editingSkillTag?.id === tag.id ? (
                      <form
                        className="skill-tag-inline-edit"
                        onSubmit={(e) => { e.preventDefault(); handleUpdateSkillTag(); }}
                      >
                        <input
                          type="text"
                          value={editSkillTagName}
                          onChange={(e) => setEditSkillTagName(e.target.value)}
                          className="skill-tag-edit-input"
                          autoFocus
                          required
                          maxLength={50}
                        />
                        <input
                          type="color"
                          value={editSkillTagColor}
                          onChange={(e) => setEditSkillTagColor(e.target.value)}
                          className="color-input color-input--sm"
                          title="Tag color"
                        />
                        <button type="submit" className="btn-primary btn-sm">Save</button>
                        <button type="button" className="btn-secondary btn-sm" onClick={() => { setEditingSkillTag(null); setEditSkillTagName(''); setEditSkillTagColor('#6b7280'); }}>Cancel</button>
                      </form>
                    ) : (
                      <>
                        <div className="tag-preview" style={{ backgroundColor: tag.color }}>
                          {tag.name}
                        </div>
                        <button
                          onClick={() => { setEditingSkillTag(tag); setEditSkillTagName(tag.name); setEditSkillTagColor(tag.color); }}
                          className="btn-edit"
                          aria-label={`Edit ${tag.name} tag`}
                          title="Edit tag"
                        >
                          ✎
                        </button>
                        <button
                          onClick={() => handleDeleteSkillTag(tag.id, tag.name)}
                          className="btn-delete"
                          aria-label={`Delete ${tag.name} tag`}
                          title="Delete tag"
                        >
                          ✕
                        </button>
                      </>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Add new skill tag form */}
          {!editingSkillTag && (
            <form
              className="comment-tag-form"
              onSubmit={(e) => { e.preventDefault(); handleCreateSkillTag(); }}
            >
              <div className="form-group">
                <label htmlFor="skillTagName">New Member Tag</label>
                <div className="inline-form">
                  <input
                    id="skillTagName"
                    type="text"
                    value={newSkillTagName}
                    onChange={(e) => setNewSkillTagName(e.target.value)}
                    placeholder="e.g. Beginner, Experienced"
                    maxLength={50}
                    required
                    autoComplete="off"
                  />
                  <input
                    type="color"
                    value={newSkillTagColor}
                    onChange={(e) => setNewSkillTagColor(e.target.value)}
                    className="color-input"
                    title="Tag color"
                  />
                  <div className="color-presets">
                    {colorPresets.map((presetColor) => (
                      <button
                        key={presetColor}
                        type="button"
                        className={`color-preset ${newSkillTagColor === presetColor ? 'selected' : ''}`}
                        style={{ backgroundColor: presetColor }}
                        onClick={() => setNewSkillTagColor(presetColor)}
                        aria-label={`Select color ${presetColor}`}
                      />
                    ))}
                  </div>
                  <div className="tag-preview-inline">
                    <span className="tag-badge" style={{ backgroundColor: newSkillTagColor }}>
                      {newSkillTagName || 'preview'}
                    </span>
                  </div>
                  <button type="submit" className="btn-primary btn-sm" disabled={!newSkillTagName.trim()}>
                    + Add Tag
                  </button>
                </div>
              </div>
            </form>
          )}
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
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        title={confirmDialog.title}
        message={confirmDialog.message}
        variant="danger"
        confirmLabel="Delete"
        onConfirm={confirmDialog.onConfirm}
        onCancel={closeConfirmDialog}
      />
    </div>
  );
};

export default AdminAnimalTagsPage;
