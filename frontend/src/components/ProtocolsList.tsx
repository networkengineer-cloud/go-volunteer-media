import React, { useEffect, useState, useCallback } from 'react';
import { protocolsApi } from '../api/client';
import type { Protocol } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import EmptyState from './EmptyState';
import SkeletonLoader from './SkeletonLoader';
import ErrorState from './ErrorState';
import Modal from './Modal';
import ConfirmDialog from './ConfirmDialog';
import './ProtocolsList.css';

interface ProtocolsListProps {
  groupId: number;
  isGroupAdmin?: boolean; // True if user is a group admin for this group
}

const ProtocolsList: React.FC<ProtocolsListProps> = ({ groupId, isGroupAdmin = false }) => {
  const { isAdmin } = useAuth();
  // Allow editing if user is site admin OR group admin
  const canEdit = isAdmin || isGroupAdmin;
  const [protocols, setProtocols] = useState<Protocol[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [showForm, setShowForm] = useState(false);
  const [editingProtocol, setEditingProtocol] = useState<Protocol | null>(null);
  const [viewingProtocol, setViewingProtocol] = useState<Protocol | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; protocol: Protocol | null }>({ 
    show: false, 
    protocol: null 
  });

  const loadProtocols = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const response = await protocolsApi.getAll(groupId);
      setProtocols(response.data);
    } catch (error) {
      console.error('Failed to load protocols:', error);
      const err = error as { response?: { data?: { error?: string } } };
      setError(err.response?.data?.error || 'Failed to load protocols. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [groupId]);

  useEffect(() => {
    loadProtocols();
  }, [loadProtocols]);

  useEffect(() => {
    console.log('showForm changed to:', showForm);
  }, [showForm]);

  const handleDelete = async (protocolId: number) => {
    try {
      await protocolsApi.delete(groupId, protocolId);
      await loadProtocols();
    } catch (error) {
      console.error('Failed to delete protocol:', error);
      alert('Failed to delete protocol. Please try again.');
    }
  };

  const handleDeleteClick = (protocol: Protocol) => {
    setDeleteConfirm({ show: true, protocol });
  };

  const handleEdit = (protocol: Protocol) => {
    setEditingProtocol(protocol);
    setShowForm(true);
  };

  const handleFormSuccess = () => {
    setShowForm(false);
    setEditingProtocol(null);
    loadProtocols();
  };

  const handleViewProtocol = (protocol: Protocol) => {
    setViewingProtocol(protocol);
  };

  if (loading) {
    return (
      <div className="protocols-list">
        <SkeletonLoader variant="card" count={3} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="protocols-list">
        <ErrorState
          title="Unable to Load Protocols"
          message={error}
          onRetry={loadProtocols}
        />
      </div>
    );
  }

  return (
    <div className="protocols-list">
      {protocols.length === 0 ? (
        <EmptyState
          icon={
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <path d="M14 2v6h6M16 13H8M16 17H8M10 9H8" />
            </svg>
          }
          title="No Protocols Yet"
          description={
            canEdit
              ? 'Get started by adding your first protocol. Protocols help volunteers follow standardized procedures.'
              : 'No protocols have been added to this group yet. Protocols will appear here once an admin adds them.'
          }
          primaryAction={
            canEdit
              ? {
                  label: 'Add First Protocol',
                  onClick: () => setShowForm(true),
                }
              : undefined
          }
        />
      ) : (
        <>
      <div className="protocols-header">
        <h2>Protocols</h2>
        {canEdit && (
          <button
            className="btn-primary"
            onClick={() => {
              console.log('Add Protocol button clicked');
              setEditingProtocol(null);
              setShowForm(true);
              console.log('showForm should now be true');
            }}
            aria-label="Add new protocol"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
              <path d="M12 5v14M5 12h14" />
            </svg>
            <span>Add Protocol</span>
          </button>
        )}
      </div>

      <div className="protocols-grid">
        {protocols.map((protocol) => (
          <div key={protocol.id} className="protocol-card">
            <div 
              className="protocol-card-clickable"
              onClick={() => handleViewProtocol(protocol)}
              role="button"
              tabIndex={0}
              onKeyPress={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleViewProtocol(protocol);
                }
              }}
              aria-label={`View ${protocol.title} details`}
            >
              {protocol.image_url && (
                <div className="protocol-image">
                  <img src={protocol.image_url} alt={protocol.title} />
                </div>
              )}
              <div className="protocol-content">
                <h3>{protocol.title}</h3>
                <div className="protocol-text protocol-text-preview">
                  {protocol.content.length > 150 
                    ? `${protocol.content.substring(0, 150)}...` 
                    : protocol.content}
                </div>
                <div className="protocol-view-more">
                  Click to view full protocol →
                </div>
              </div>
            </div>
            {canEdit && (
              <div className="protocol-actions">
                <button
                  className="btn-secondary"
                  onClick={() => handleEdit(protocol)}
                  aria-label={`Edit ${protocol.title}`}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  <span>Edit</span>
                </button>
                <button
                  className="btn-danger"
                  onClick={() => handleDeleteClick(protocol)}
                  aria-label={`Delete ${protocol.title}`}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  <span>Delete</span>
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
      </>
      )}

      {/* Protocol Form Modal */}
      {showForm && (
        <Modal
          isOpen={showForm}
          onClose={() => {
            console.log('Modal onClose called');
            setShowForm(false);
            setEditingProtocol(null);
          }}
          title={editingProtocol ? 'Edit Protocol' : 'Add Protocol'}
        >
          <ProtocolForm
            groupId={groupId}
            protocol={editingProtocol}
            onSuccess={handleFormSuccess}
            onCancel={() => {
              console.log('Form cancel called');
              setShowForm(false);
              setEditingProtocol(null);
            }}
          />
        </Modal>
      )}

      {/* Protocol View Modal */}
      {viewingProtocol && (
        <Modal
          isOpen={!!viewingProtocol}
          onClose={() => setViewingProtocol(null)}
          title={viewingProtocol.title}
          size="large"
        >
          <div className="protocol-view">
            {viewingProtocol.image_url && (
              <div className="protocol-view-image">
                <img src={viewingProtocol.image_url} alt={viewingProtocol.title} />
              </div>
            )}
            <div className="protocol-view-content">
              <p>{viewingProtocol.content}</p>
            </div>
            {canEdit && (
              <div className="protocol-view-actions">
                <button
                  className="btn-secondary"
                  onClick={() => {
                    setViewingProtocol(null);
                    handleEdit(viewingProtocol);
                  }}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  <span>Edit</span>
                </button>
                <button
                  className="btn-danger"
                  onClick={() => {
                    setViewingProtocol(null);
                    handleDeleteClick(viewingProtocol);
                  }}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  <span>Delete</span>
                </button>
              </div>
            )}
          </div>
        </Modal>
      )}

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={deleteConfirm.show}
        title="Delete Protocol?"
        message={`Are you sure you want to delete "${deleteConfirm.protocol?.title}"? This action cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        onConfirm={() => {
          if (deleteConfirm.protocol) {
            handleDelete(deleteConfirm.protocol.id);
          }
        }}
        onCancel={() => setDeleteConfirm({ show: false, protocol: null })}
      />
    </div>
  );
};

interface ProtocolFormProps {
  groupId: number;
  protocol: Protocol | null;
  onSuccess: () => void;
  onCancel: () => void;
}

const ProtocolForm: React.FC<ProtocolFormProps> = ({ groupId, protocol, onSuccess, onCancel }) => {
  const [title, setTitle] = useState(protocol?.title || '');
  const [content, setContent] = useState(protocol?.content || '');
  const [imageUrl, setImageUrl] = useState(protocol?.image_url || '');
  const [orderIndex, setOrderIndex] = useState(protocol?.order_index || 0);
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      setUploading(true);
      const response = await protocolsApi.uploadImage(groupId, file);
      setImageUrl(response.data.url);
    } catch (error) {
      console.error('Failed to upload image:', error);
      alert('Failed to upload image. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim() || !content.trim()) {
      alert('Please fill in all required fields.');
      return;
    }

    try {
      setSubmitting(true);
      const data = {
        title: title.trim(),
        content: content.trim(),
        image_url: imageUrl,
        order_index: orderIndex,
      };

      if (protocol) {
        await protocolsApi.update(groupId, protocol.id, data);
      } else {
        await protocolsApi.create(groupId, data);
      }

      onSuccess();
    } catch (error) {
      console.error('Failed to save protocol:', error);
      const err = error as { response?: { data?: { error?: string } } };
      alert(err.response?.data?.error || 'Failed to save protocol. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="protocol-form" onSubmit={handleSubmit}>
      <div className="form-group">
        <label htmlFor="protocol-title">
          Title <span style={{ color: 'var(--danger)' }}>*</span>
        </label>
        <input
          id="protocol-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Enter protocol title"
          required
          maxLength={200}
          aria-label="Protocol title"
          aria-required="true"
          aria-describedby="title-hint"
        />
        <small id="title-hint" style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', display: 'block', marginTop: '0.25rem' }}>
          {title.length}/200 characters
        </small>
      </div>

      <div className="form-group">
        <label htmlFor="protocol-content">
          Content <span style={{ color: 'var(--danger)' }}>*</span>
        </label>
        <textarea
          id="protocol-content"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Enter protocol details and instructions..."
          required
          rows={8}
          maxLength={1000}
          aria-label="Protocol content"
          aria-required="true"
          aria-describedby="content-hint"
        />
        <small id="content-hint" style={{ color: content.length > 900 ? 'var(--danger)' : 'var(--text-secondary)', fontSize: '0.875rem', display: 'block', marginTop: '0.25rem' }}>
          {content.length}/1000 characters{content.length > 900 && ` (${1000 - content.length} remaining)`}
        </small>
      </div>

      <div className="form-group">
        <label htmlFor="protocol-image">Image (optional)</label>
        <input
          id="protocol-image"
          type="file"
          accept="image/*"
          onChange={handleImageUpload}
          disabled={uploading}
          aria-label="Upload protocol image"
          aria-describedby="image-hint"
        />
        <small id="image-hint" style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', display: 'block', marginTop: '0.25rem' }}>
          Accepted formats: JPG, PNG, GIF. Maximum size: 10MB
        </small>
        {uploading && <p className="upload-status" role="status" aria-live="polite">Uploading image...</p>}
        {imageUrl && !uploading && (
          <div className="image-preview">
            <img src={imageUrl} alt="Protocol preview" />
            <button
              type="button"
              className="remove-image"
              onClick={() => setImageUrl('')}
              aria-label="Remove image"
            >
              ×
            </button>
          </div>
        )}
      </div>

      <div className="form-group">
        <label htmlFor="protocol-order">Display Order</label>
        <input
          id="protocol-order"
          type="number"
          value={orderIndex}
          onChange={(e) => setOrderIndex(Number(e.target.value))}
          placeholder="0"
          aria-label="Display order for protocol"
          aria-describedby="order-hint"
        />
        <small id="order-hint" style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', display: 'block', marginTop: '0.25rem' }}>
          Lower numbers appear first. Leave as 0 to append to the end.
        </small>
      </div>

      <div className="form-actions">
        <button type="button" className="btn-secondary" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? 'Saving...' : protocol ? 'Update Protocol' : 'Add Protocol'}
        </button>
      </div>
    </form>
  );
};

export default ProtocolsList;
