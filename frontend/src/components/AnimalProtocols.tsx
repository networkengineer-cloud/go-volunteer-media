import React, { useEffect, useState, useRef } from 'react';
import { animalProtocolsApi, animalsApi as imageUploadApi } from '../api/client';
import type { AnimalProtocol } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import Modal from './Modal';
import ConfirmDialog from './ConfirmDialog';
import EmptyState from './EmptyState';
import './AnimalProtocols.css';

interface AnimalProtocolsProps {
  groupId: number;
  animalId: number;
}

const AnimalProtocols: React.FC<AnimalProtocolsProps> = ({ groupId, animalId }) => {
  const { isAdmin } = useAuth();
  const toast = useToast();
  const [protocols, setProtocols] = useState<AnimalProtocol[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; protocol: AnimalProtocol | null }>({
    show: false,
    protocol: null,
  });

  useEffect(() => {
    loadProtocols();
  }, [groupId, animalId]);

  const loadProtocols = async () => {
    try {
      setLoading(true);
      const response = await animalProtocolsApi.getAll(groupId, animalId);
      setProtocols(response.data);
    } catch (error) {
      console.error('Failed to load protocols:', error);
      toast.showError('Failed to load protocols. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (protocolId: number) => {
    try {
      await animalProtocolsApi.delete(groupId, animalId, protocolId);
      await loadProtocols();
      toast.showSuccess('Protocol deleted successfully');
    } catch (error) {
      console.error('Failed to delete protocol:', error);
      toast.showError('Failed to delete protocol. Please try again.');
    }
  };

  const handleDeleteClick = (protocol: AnimalProtocol) => {
    setDeleteConfirm({ show: true, protocol });
  };

  const handleFormSuccess = () => {
    setShowForm(false);
    loadProtocols();
    toast.showSuccess('Protocol saved successfully!');
  };

  if (loading) {
    return (
      <div className="animal-protocols">
        <p>Loading protocols...</p>
      </div>
    );
  }

  return (
    <div className="animal-protocols">
      <div className="protocols-header">
        <h3>Protocol Entries</h3>
        <button
          className="btn-add-protocol"
          onClick={() => setShowForm(true)}
          aria-label="Add protocol entry"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <path d="M12 5v14M5 12h14" />
          </svg>
          <span>Add Protocol</span>
        </button>
      </div>

      {protocols.length === 0 ? (
        <EmptyState
          icon={
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <path d="M14 2v6h6M16 13H8M16 17H8M10 9H8" />
            </svg>
          }
          title="No Protocol Entries"
          description="Start documenting procedures and protocols for this animal. Protocol entries help maintain consistent care practices."
          primaryAction={{
            label: 'Add First Protocol',
            onClick: () => setShowForm(true),
          }}
        />
      ) : (
        <div className="protocols-list">
          {protocols.map((protocol) => (
            <div key={protocol.id} className="protocol-card">
              <div className="protocol-header">
                <span className="protocol-author">{protocol.user?.username}</span>
                <span className="protocol-date">
                  {new Date(protocol.created_at).toLocaleDateString()} at{' '}
                  {new Date(protocol.created_at).toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              </div>
              <p className="protocol-content">{protocol.content}</p>
              {protocol.image_url && (
                <img
                  src={protocol.image_url}
                  alt="Protocol"
                  className="protocol-image"
                />
              )}
              {isAdmin && (
                <div className="protocol-actions">
                  <button
                    className="btn-delete-protocol"
                    onClick={() => handleDeleteClick(protocol)}
                    aria-label="Delete protocol"
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
      )}

      {/* Protocol Form Modal */}
      {showForm && (
        <Modal
          isOpen={showForm}
          onClose={() => setShowForm(false)}
          title="Add Protocol Entry"
        >
          <ProtocolForm
            groupId={groupId}
            animalId={animalId}
            onSuccess={handleFormSuccess}
            onCancel={() => setShowForm(false)}
          />
        </Modal>
      )}

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={deleteConfirm.show}
        title="Delete Protocol Entry?"
        message="Are you sure you want to delete this protocol entry? This action cannot be undone."
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        onConfirm={() => {
          if (deleteConfirm.protocol) {
            handleDelete(deleteConfirm.protocol.id);
          }
          setDeleteConfirm({ show: false, protocol: null });
        }}
        onCancel={() => setDeleteConfirm({ show: false, protocol: null })}
      />
    </div>
  );
};

interface ProtocolFormProps {
  groupId: number;
  animalId: number;
  onSuccess: () => void;
  onCancel: () => void;
}

const ProtocolForm: React.FC<ProtocolFormProps> = ({ groupId, animalId, onSuccess, onCancel }) => {
  const toast = useToast();
  const [content, setContent] = useState('');
  const [imageUrl, setImageUrl] = useState('');
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
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

    setUploading(true);
    try {
      const res = await imageUploadApi.uploadImage(file);
      setImageUrl(res.data.url);
      toast.showSuccess('Image uploaded successfully!');
    } catch (error: any) {
      console.error('Upload error:', error);
      const errorMsg = error.response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!content.trim()) {
      toast.showWarning('Please enter protocol content');
      return;
    }

    if (content.trim().length > 1000) {
      toast.showError('Protocol content must be 1000 characters or less');
      return;
    }

    setSubmitting(true);
    try {
      await animalProtocolsApi.create(groupId, animalId, content.trim(), imageUrl || undefined);
      onSuccess();
    } catch (error: any) {
      console.error('Failed to save protocol:', error);
      const errorMsg = error.response?.data?.error || 'Failed to save protocol. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setSubmitting(false);
    }
  };

  const remainingChars = 1000 - content.length;

  return (
    <form className="protocol-form" onSubmit={handleSubmit}>
      <div className="form-group">
        <label htmlFor="protocol-content">
          Protocol Content <span style={{ color: 'var(--danger)' }}>*</span>
        </label>
        <textarea
          id="protocol-content"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Enter protocol instructions, procedures, or notes..."
          required
          rows={8}
          maxLength={1000}
          aria-label="Protocol content"
          aria-required="true"
          aria-describedby="content-hint"
          disabled={submitting}
        />
        <small
          id="content-hint"
          style={{
            color: remainingChars < 100 ? 'var(--danger)' : 'var(--text-secondary)',
            fontSize: '0.875rem',
            display: 'block',
            marginTop: '0.25rem',
          }}
        >
          {content.length}/1000 characters{remainingChars < 100 && ` (${remainingChars} remaining)`}
        </small>
      </div>

      {imageUrl && (
        <div className="image-preview">
          <img src={imageUrl} alt="Protocol preview" />
          <button
            type="button"
            className="remove-image"
            onClick={() => setImageUrl('')}
            aria-label="Remove image"
            disabled={submitting}
          >
            ✕
          </button>
        </div>
      )}

      <div className="form-actions">
        <input
          ref={fileInputRef}
          type="file"
          accept=".jpg,.jpeg,.png,.gif"
          onChange={handleImageUpload}
          style={{ display: 'none' }}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          className="btn-upload"
          disabled={uploading || submitting}
        >
          {uploading ? 'Uploading...' : '📷 Add Image'}
        </button>
        <div className="form-actions-right">
          <button type="button" className="btn-secondary" onClick={onCancel} disabled={submitting}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={submitting || !content.trim()}>
            {submitting ? 'Saving...' : 'Save Protocol'}
          </button>
        </div>
      </div>
    </form>
  );
};

export default AnimalProtocols;
