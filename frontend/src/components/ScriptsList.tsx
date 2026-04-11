import React, { useEffect, useState, useCallback } from 'react';
import { scriptsApi } from '../api/client';
import type { Script } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import EmptyState from './EmptyState';
import SkeletonLoader from './SkeletonLoader';
import ErrorState from './ErrorState';
import Modal from './Modal';
import ConfirmDialog from './ConfirmDialog';
import './ScriptsList.css';

interface ScriptsListProps {
  groupId: number;
  isGroupAdmin?: boolean;
  showFormExternal?: boolean;
  onShowFormChange?: (show: boolean) => void;
  hideAddButton?: boolean;
}

const FILE_ICON = (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <polyline points="14 2 14 8 20 8" />
    <line x1="16" y1="13" x2="8" y2="13" />
    <line x1="16" y1="17" x2="8" y2="17" />
    <polyline points="10 9 9 9 8 9" />
  </svg>
);

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const ScriptsList: React.FC<ScriptsListProps> = ({
  groupId,
  isGroupAdmin = false,
  showFormExternal = false,
  onShowFormChange,
  hideAddButton = false,
}) => {
  const { isAdmin } = useAuth();
  const canEdit = isAdmin || isGroupAdmin;
  const [scripts, setScripts] = useState<Script[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [searchText, setSearchText] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [editingScript, setEditingScript] = useState<Script | null>(null);
  const [viewingScript, setViewingScript] = useState<Script | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; script: Script | null }>({
    show: false,
    script: null,
  });

  useEffect(() => {
    if (showFormExternal && !showForm) {
      setEditingScript(null);
      setShowForm(true);
    }
  }, [showFormExternal, showForm]);

  const handleSetShowForm = (show: boolean) => {
    setShowForm(show);
    onShowFormChange?.(show);
  };

  const loadScripts = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const response = await scriptsApi.getAll(groupId);
      setScripts(response.data);
    } catch (err) {
      const e = err as { response?: { data?: { error?: string } } };
      setError(e.response?.data?.error || 'Failed to load scripts. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [groupId]);

  useEffect(() => {
    loadScripts();
  }, [loadScripts]);

  const handleDelete = async (scriptId: number) => {
    try {
      await scriptsApi.delete(groupId, scriptId);
      await loadScripts();
    } catch {
      alert('Failed to delete script. Please try again.');
    }
  };

  const handleEdit = (script: Script) => {
    setEditingScript(script);
    handleSetShowForm(true);
  };

  const handleFormSuccess = () => {
    handleSetShowForm(false);
    setEditingScript(null);
    loadScripts();
  };

  const filteredScripts = searchText.trim()
    ? scripts.filter(
        (s) =>
          s.title.toLowerCase().includes(searchText.toLowerCase()) ||
          (s.description || '').toLowerCase().includes(searchText.toLowerCase()) ||
          s.file_name.toLowerCase().includes(searchText.toLowerCase())
      )
    : scripts;

  if (loading) {
    return (
      <div className="scripts-list">
        <SkeletonLoader variant="card" count={3} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="scripts-list">
        <ErrorState title="Unable to Load Scripts" message={error} onRetry={loadScripts} />
      </div>
    );
  }

  return (
    <div className="scripts-list">
      {scripts.length === 0 ? (
        <EmptyState
          icon={
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
              <line x1="16" y1="13" x2="8" y2="13" />
              <line x1="16" y1="17" x2="8" y2="17" />
            </svg>
          }
          title="No Scripts Yet"
          description={
            canEdit
              ? 'Get started by uploading your first script. Scripts are shared procedures volunteers can follow.'
              : 'No scripts have been added to this group yet.'
          }
          primaryAction={
            canEdit && !hideAddButton
              ? { label: 'Upload First Script', onClick: () => handleSetShowForm(true) }
              : undefined
          }
        />
      ) : (
        <>
          <div className="scripts-header">
            <h2>Scripts</h2>
            {canEdit && !hideAddButton && (
              <button
                className="btn-primary"
                onClick={() => {
                  setEditingScript(null);
                  handleSetShowForm(true);
                }}
                aria-label="Upload new script"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <path d="M12 5v14M5 12h14" />
                </svg>
                <span>Upload Script</span>
              </button>
            )}
          </div>

          {/* Search bar — only show when there are scripts to filter */}
          {scripts.length > 0 && (
            <div className="scripts-search-bar">
              <svg
                className="scripts-search-icon"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                aria-hidden="true"
              >
                <circle cx="11" cy="11" r="8" />
                <line x1="21" y1="21" x2="16.65" y2="16.65" />
              </svg>
              <input
                type="search"
                className="scripts-search-input"
                placeholder="Search scripts…"
                value={searchText}
                onChange={(e) => setSearchText(e.target.value)}
                aria-label="Search scripts"
              />
              {searchText && (
                <span className="scripts-search-count" aria-live="polite">
                  {filteredScripts.length} of {scripts.length}
                </span>
              )}
            </div>
          )}

          <div className="scripts-grid">
            {filteredScripts.map((script) => (
              <div key={script.id} className="script-card">
                <div
                  className="script-card-clickable"
                  onClick={() => setViewingScript(script)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      setViewingScript(script);
                    }
                  }}
                  aria-label={`View ${script.title}`}
                >
                  <div className="script-file-badge">{FILE_ICON}</div>
                  <div className="script-content">
                    <h3>{script.title}</h3>
                    {script.description && (
                      <p className="script-description">
                        {script.description.length > 150
                          ? `${script.description.substring(0, 150)}…`
                          : script.description}
                      </p>
                    )}
                    <div className="script-meta">
                      <span className="script-filename">{script.file_name}</span>
                      {script.file_size > 0 && (
                        <span className="script-filesize">{formatFileSize(script.file_size)}</span>
                      )}
                    </div>
                    <div className="script-view-more">Click to view →</div>
                  </div>
                </div>
                {canEdit && (
                  <div className="script-actions">
                    <button
                      className="btn-secondary"
                      onClick={() => handleEdit(script)}
                      aria-label={`Edit ${script.title}`}
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                      </svg>
                      <span>Edit</span>
                    </button>
                    <button
                      className="btn-danger"
                      onClick={() => setDeleteConfirm({ show: true, script })}
                      aria-label={`Delete ${script.title}`}
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

          {searchText && filteredScripts.length === 0 && (
            <p className="scripts-no-results">
              No scripts match &ldquo;{searchText}&rdquo;.
            </p>
          )}
        </>
      )}

      {/* Script Form Modal */}
      {showForm && (
        <Modal
          isOpen={showForm}
          onClose={() => {
            handleSetShowForm(false);
            setEditingScript(null);
          }}
          title={editingScript ? 'Edit Script' : 'Upload Script'}
        >
          <ScriptForm
            groupId={groupId}
            script={editingScript}
            onSuccess={handleFormSuccess}
            onCancel={() => {
              handleSetShowForm(false);
              setEditingScript(null);
            }}
          />
        </Modal>
      )}

      {/* Script View Modal */}
      {viewingScript && (
        <Modal
          isOpen={!!viewingScript}
          onClose={() => setViewingScript(null)}
          title={viewingScript.title}
          size="large"
        >
          <ScriptViewer
            script={viewingScript}
            canEdit={canEdit}
            onEdit={() => {
              setViewingScript(null);
              handleEdit(viewingScript);
            }}
            onDelete={() => {
              setViewingScript(null);
              setDeleteConfirm({ show: true, script: viewingScript });
            }}
          />
        </Modal>
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={deleteConfirm.show}
        title="Delete Script?"
        message={`Are you sure you want to delete "${deleteConfirm.script?.title}"? This action cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        onConfirm={() => {
          if (deleteConfirm.script) {
            handleDelete(deleteConfirm.script.id);
          }
          setDeleteConfirm({ show: false, script: null });
        }}
        onCancel={() => setDeleteConfirm({ show: false, script: null })}
      />
    </div>
  );
};

// ─────────────────────────────────────────────
// ScriptViewer: fetches the file with the stored JWT, then renders inline.
// DOCX/DOC → HTML via mammoth.js | PDF → iframe | other → download link
// ─────────────────────────────────────────────

export interface ScriptViewerProps {
  script: Script;
  canEdit?: boolean;
  onEdit?: () => void;
  onDelete?: () => void;
}

export const ScriptViewer: React.FC<ScriptViewerProps> = ({ script, canEdit, onEdit, onDelete }) => {
  const [viewState, setViewState] = useState<{
    status: 'loading' | 'pdf' | 'docx' | 'other' | 'error';
    blobUrl: string | null;
    htmlContent: string | null;
  }>({ status: 'loading', blobUrl: null, htmlContent: null });

  useEffect(() => {
    let activeBlobUrl: string | null = null;
    let cancelled = false;

    const load = async () => {
      try {
        const token = localStorage.getItem('token');
        const res = await fetch(script.file_url, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok || cancelled) return;

        const name = script.file_name.toLowerCase();
        const type = (script.file_type || '').toLowerCase();
        const isDocx =
          name.endsWith('.docx') ||
          type.includes('officedocument') ||
          type.includes('msword');

        if (isDocx) {
          const arrayBuffer = await res.arrayBuffer();
          if (cancelled) return;
          const mammoth = (await import('mammoth')).default;
          const result = await mammoth.convertToHtml({ arrayBuffer });
          if (cancelled) return;
          const DOMPurify = (await import('dompurify')).default;
          const safe = DOMPurify.sanitize(result.value);
          setViewState({ status: 'docx', blobUrl: null, htmlContent: safe });
        } else {
          const blob = await res.blob();
          if (cancelled) return;
          activeBlobUrl = URL.createObjectURL(blob);
          const isPdf = type === 'application/pdf' || name.endsWith('.pdf');
          setViewState({ status: isPdf ? 'pdf' : 'other', blobUrl: activeBlobUrl, htmlContent: null });
        }
      } catch {
        if (!cancelled) setViewState({ status: 'error', blobUrl: null, htmlContent: null });
      }
    };

    load();
    return () => {
      cancelled = true;
      if (activeBlobUrl) URL.revokeObjectURL(activeBlobUrl);
    };
  }, [script.file_url, script.file_name, script.file_type]);

  if (viewState.status === 'loading') {
    return (
      <div className="script-viewer">
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-secondary)' }}>Loading…</div>
      </div>
    );
  }

  if (viewState.status === 'error') {
    return (
      <div className="script-viewer script-viewer-download">
        <div className="script-viewer-icon">{FILE_ICON}</div>
        <p className="script-viewer-name">{script.file_name}</p>
        <p style={{ color: 'var(--error, #e53e3e)', marginBottom: '1rem' }}>Unable to load file.</p>
        {canEdit && (
          <div className="script-viewer-actions">
            <button className="btn-secondary" onClick={onEdit}>Edit</button>
            <button className="btn-danger" onClick={onDelete}>Delete</button>
          </div>
        )}
      </div>
    );
  }

  if (viewState.status === 'docx' && viewState.htmlContent) {
    return (
      <div className="script-viewer">
        <div
          className="script-docx-body"
          // eslint-disable-next-line react/no-danger
          dangerouslySetInnerHTML={{ __html: viewState.htmlContent }}
        />
        {canEdit && (
          <div className="script-viewer-actions">
            <button className="btn-secondary" onClick={onEdit}>Edit</button>
            <button className="btn-danger" onClick={onDelete}>Delete</button>
          </div>
        )}
      </div>
    );
  }

  if (viewState.status === 'pdf') {
    return (
      <div className="script-viewer">
        <iframe
          src={viewState.blobUrl!}
          title={script.title}
          className="script-iframe"
          aria-label={`${script.title} PDF viewer`}
        />
        {canEdit && (
          <div className="script-viewer-actions">
            <button className="btn-secondary" onClick={onEdit}>Edit</button>
            <button className="btn-danger" onClick={onDelete}>Delete</button>
          </div>
        )}
      </div>
    );
  }

  // Other file types — offer blob-URL download (no forced browser download prompt)
  return (
    <div className="script-viewer script-viewer-download">
      <div className="script-viewer-icon">{FILE_ICON}</div>
      <p className="script-viewer-name">{script.file_name}</p>
      {script.description && <p className="script-viewer-desc">{script.description}</p>}
      {viewState.blobUrl && (
        <a
          href={viewState.blobUrl}
          download={script.file_name}
          className="btn-primary"
          aria-label={`Download ${script.file_name}`}
        >
          Download File
        </a>
      )}
      {canEdit && (
        <div className="script-viewer-actions">
          <button className="btn-secondary" onClick={onEdit}>Edit</button>
          <button className="btn-danger" onClick={onDelete}>Delete</button>
        </div>
      )}
    </div>
  );
};

// ─────────────────────────────────────────────
// ScriptForm: upload / edit form
// ─────────────────────────────────────────────

interface ScriptFormProps {
  groupId: number;
  script: Script | null;
  onSuccess: () => void;
  onCancel: () => void;
}

const ScriptForm: React.FC<ScriptFormProps> = ({ groupId, script, onSuccess, onCancel }) => {
  const [title, setTitle] = useState(script?.title || '');
  const [description, setDescription] = useState(script?.description || '');
  const [orderIndex, setOrderIndex] = useState(script?.order_index ?? 0);
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      alert('Title is required.');
      return;
    }
    if (!script && !file) {
      alert('A file is required when uploading a new script.');
      return;
    }

    const formData = new FormData();
    formData.append('title', title.trim());
    formData.append('description', description.trim());
    formData.append('order_index', String(orderIndex));
    if (file) {
      formData.append('file', file);
    }

    try {
      setSubmitting(true);
      if (script) {
        await scriptsApi.update(groupId, script.id, formData);
      } else {
        await scriptsApi.create(groupId, formData);
      }
      onSuccess();
    } catch (err) {
      const e = err as { response?: { data?: { error?: string } } };
      alert(e.response?.data?.error || 'Failed to save script. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="script-form" onSubmit={handleSubmit}>
      <div className="form-group">
        <label htmlFor="script-title">
          Title <span aria-hidden="true" style={{ color: 'var(--danger)' }}>*</span>
        </label>
        <input
          id="script-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Enter script title"
          required
          maxLength={200}
          aria-required="true"
        />
        <small style={{ color: 'var(--text-secondary)', fontSize: '0.875rem' }}>{title.length}/200</small>
      </div>

      <div className="form-group">
        <label htmlFor="script-description">Description</label>
        <textarea
          id="script-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Brief description of this script (optional)"
          rows={3}
          maxLength={500}
        />
        <small
          style={{
            color: description.length > 450 ? 'var(--danger)' : 'var(--text-secondary)',
            fontSize: '0.875rem',
          }}
        >
          {description.length}/500
        </small>
      </div>

      <div className="form-group">
        <label htmlFor="script-file">
          File {!script && <span aria-hidden="true" style={{ color: 'var(--danger)' }}>*</span>}
        </label>
        <input
          id="script-file"
          type="file"
          accept=".pdf,.docx"
          onChange={(e) => setFile(e.target.files?.[0] || null)}
          aria-required={!script}
          aria-describedby="file-hint"
        />
        <small id="file-hint" style={{ color: 'var(--text-secondary)', fontSize: '0.875rem' }}>
          {script
            ? `Current file: ${script.file_name}. Upload a new file to replace it.`
            : 'Accepted formats: PDF, DOCX. Maximum size: 20 MB.'}
        </small>
      </div>

      <div className="form-group">
        <label htmlFor="script-order">Display Order</label>
        <input
          id="script-order"
          type="number"
          value={orderIndex}
          onChange={(e) => setOrderIndex(Number(e.target.value))}
          min={0}
          aria-label="Display order"
        />
      </div>

      <div className="form-actions">
        <button type="button" className="btn-secondary" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? 'Saving…' : script ? 'Save Changes' : 'Upload Script'}
        </button>
      </div>
    </form>
  );
};

export default ScriptsList;
