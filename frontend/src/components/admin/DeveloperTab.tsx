import React, { useState } from 'react';
import api from '../../api/client';
import { useToast } from '../../hooks/useToast';
import './DeveloperTab.css';

const DeveloperTab: React.FC = () => {
  const toast = useToast();
  const [isSeeding, setIsSeeding] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);

  const handleSeedDatabase = async () => {
    setIsSeeding(true);
    try {
      const response = await api.post('/admin/seed-database');
      toast.showSuccess('Database re-seeded successfully! Demo accounts have been reset.');
      console.log('Seed response:', response.data);
      setShowConfirmModal(false);
    } catch (error: unknown) {
      if (error instanceof Error) {
        toast.showError('Failed to seed database: ' + error.message);
      } else {
        toast.showError('Failed to seed database');
      }
    } finally {
      setIsSeeding(false);
    }
  };

  return (
    <div className="developer-tab">
      <div className="developer-section">
        <h2>🧪 Development Tools</h2>
        <p className="section-description">
          Dangerous operations for development and testing environments only.
        </p>

        <div className="developer-card danger-zone">
          <div className="card-header">
            <h3>⚠️ Database Seeding</h3>
            <span className="dev-only-badge">DEV ONLY</span>
          </div>
          
          <p className="card-description">
            Re-seed the database with fresh demo data. This will:
          </p>
          
          <ul className="seed-info-list">
            <li>Reset all users and create demo accounts</li>
            <li>Create sample ModSquad group with 10 dogs</li>
            <li>Reset all animals, comments, and updates</li>
            <li>Generate fresh animal and comment tags</li>
          </ul>

          <div className="demo-accounts-info">
            <strong>Demo Accounts Created:</strong>
            <ul>
              <li><code>admin</code> / <code>demo1234</code> - Administrator</li>
              <li><code>sarah_modsquad</code> / <code>demo1234</code> - Volunteer</li>
              <li><code>mike_modsquad</code> / <code>demo1234</code> - Volunteer</li>
              <li><code>jake_modsquad</code> / <code>demo1234</code> - Volunteer</li>
              <li><code>lisa_modsquad</code> / <code>demo1234</code> - Volunteer</li>
            </ul>
          </div>

          <button
            className="seed-button danger-button"
            onClick={() => setShowConfirmModal(true)}
            disabled={isSeeding}
          >
            {isSeeding ? '🔄 Seeding Database...' : '🌱 Re-seed Database'}
          </button>
        </div>
      </div>

      {showConfirmModal && (
        <div className="modal-overlay" onClick={() => setShowConfirmModal(false)}>
          <div className="modal-content seed-confirm-modal" onClick={(e) => e.stopPropagation()}>
            <h2>⚠️ Confirm Database Re-seed</h2>
            <p className="confirm-warning">
              This will <strong>completely reset</strong> the database and replace all data with demo content.
            </p>
            <p className="confirm-warning">
              <strong>This action cannot be undone!</strong>
            </p>
            <p className="confirm-info">
              All existing users, animals, comments, and updates will be deleted and replaced with seed data.
            </p>
            <div className="modal-actions">
              <button
                className="cancel-button"
                onClick={() => setShowConfirmModal(false)}
                disabled={isSeeding}
              >
                Cancel
              </button>
              <button
                className="confirm-button danger-button"
                onClick={handleSeedDatabase}
                disabled={isSeeding}
              >
                {isSeeding ? 'Seeding...' : 'Yes, Reset Database'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DeveloperTab;
