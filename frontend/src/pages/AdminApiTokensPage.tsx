import React, { useState, useEffect, useCallback } from 'react';
import { apiTokensApi, type ApiToken, type CreatedApiToken } from '../api/client';
import Modal from '../components/Modal';
import ConfirmDialog from '../components/ConfirmDialog';
import { useConfirmDialog } from '../hooks/useConfirmDialog';
import SkeletonLoader from '../components/SkeletonLoader';
import './AdminApiTokensPage.css';

const expiryPresets = [
  { label: '30 days', days: 30 },
  { label: '90 days', days: 90 },
  { label: '1 year', days: 365 },
];

interface CreateTokenModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { name: string; expires_at: string }) => Promise<void>;
}

const CreateTokenModal: React.FC<CreateTokenModalProps> = ({ isOpen, onClose, onSubmit }) => {
  const [name, setName] = useState('');
  const [presetDays, setPresetDays] = useState(90);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setName('');
      setPresetDays(90);
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    const expiresAt = new Date(Date.now() + presetDays * 24 * 60 * 60 * 1000).toISOString();
    setIsSubmitting(true);
    try {
      await onSubmit({ name: name.trim(), expires_at: expiresAt });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Generate API Token">
      <form onSubmit={handleSubmit} className="token-form">
        <div className="form-group">
          <label htmlFor="tokenName">Name</label>
          <input
            type="text"
            id="tokenName"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Zapier integration, reporting script"
            required
            autoComplete="off"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label htmlFor="tokenExpiry">Expires in</label>
          <select
            id="tokenExpiry"
            value={presetDays}
            onChange={(e) => setPresetDays(Number(e.target.value))}
          >
            {expiryPresets.map((preset) => (
              <option key={preset.days} value={preset.days}>
                {preset.label}
              </option>
            ))}
          </select>
        </div>

        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="btn-primary" disabled={isSubmitting || !name.trim()}>
            {isSubmitting ? 'Creating...' : 'Create'}
          </button>
        </div>
      </form>
    </Modal>
  );
};

const AdminApiTokensPage: React.FC = () => {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [revealedToken, setRevealedToken] = useState<CreatedApiToken | null>(null);
  const { confirmDialog, openConfirmDialog, closeConfirmDialog } = useConfirmDialog();

  const fetchTokens = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await apiTokensApi.list();
      setTokens(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load API tokens');
      console.error('Error fetching API tokens:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  const handleCreate = useCallback(async (data: { name: string; expires_at: string }) => {
    try {
      const response = await apiTokensApi.create(data);
      setRevealedToken(response.data);
      setIsCreateModalOpen(false);
      await fetchTokens();
    } catch (err) {
      console.error('Error creating API token:', err);
      throw err;
    }
  }, [fetchTokens]);

  const handleRevoke = useCallback((token: ApiToken) => {
    openConfirmDialog(
      'Revoke API Token',
      `Are you sure you want to revoke "${token.name}"? Anything using this token will immediately lose access.`,
      async () => {
        try {
          await apiTokensApi.revoke(token.id);
          await fetchTokens();
        } catch (err) {
          console.error('Error revoking API token:', err);
          setError('Failed to revoke API token');
        }
      },
    );
  }, [fetchTokens, openConfirmDialog]);

  if (isLoading) {
    return (
      <div className="admin-api-tokens-page">
        <SkeletonLoader variant="rectangular" height="40px" count={3} />
      </div>
    );
  }

  return (
    <div className="admin-api-tokens-page">
      <div className="page-header">
        <div className="page-header-content">
          <h1>🔑 API Tokens</h1>
          <p className="page-subtitle">
            Generate tokens for scripts or integrations to call the API on your behalf. A token
            has the same access you do and can be revoked at any time.
          </p>
        </div>
        <button className="btn-primary" onClick={() => setIsCreateModalOpen(true)}>
          + Generate Token
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}

      {revealedToken && (
        <div className="token-secret-reveal">
          <p>
            <strong>Copy this token now — it won&apos;t be shown again.</strong>
          </p>
          <p className="token-secret-value">{revealedToken.token}</p>
          <button className="btn-secondary btn-sm" onClick={() => setRevealedToken(null)}>
            Done
          </button>
        </div>
      )}

      {tokens.length === 0 ? (
        <div className="token-empty-state">
          <p>No API tokens yet. Generate one to get started.</p>
        </div>
      ) : (
        <table className="token-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Token</th>
              <th>Created</th>
              <th>Expires</th>
              <th>Last Used</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {tokens.map((token) => (
              <tr key={token.id}>
                <td>{token.name}</td>
                <td><code>{token.token_prefix}…</code></td>
                <td>{new Date(token.created_at).toLocaleDateString()}</td>
                <td>{new Date(token.expires_at).toLocaleDateString()}</td>
                <td>{token.last_used_at ? new Date(token.last_used_at).toLocaleDateString() : 'Never'}</td>
                <td>
                  <button
                    className="btn-text btn-danger"
                    onClick={() => handleRevoke(token)}
                    aria-label={`Revoke ${token.name}`}
                  >
                    Revoke
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <CreateTokenModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={handleCreate}
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

export default AdminApiTokensPage;
