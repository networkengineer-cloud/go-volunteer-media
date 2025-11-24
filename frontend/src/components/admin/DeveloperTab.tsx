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
            <strong>⚠️ Warning:</strong> This will completely wipe and reset the database with fresh demo data.
          </p>
          
          <p className="card-description">
            Click the button below to reset the database. This action will:
          </p>
          
          <ul className="seed-info-list">
            <li>Delete all existing data (users, animals, comments, updates)</li>
            <li>Create new demo user accounts with password: <code>demo1234</code></li>
            <li>Create sample ModSquad group with 10 dogs</li>
            <li>Generate fresh animal and comment tags</li>
          </ul>

          <div className="demo-accounts-info">
            <strong>New Demo Accounts (all with password: demo1234):</strong>
            <ul>
              <li><code>admin</code> - Administrator</li>
              <li><code>merry</code> - Volunteer</li>
              <li><code>sophia</code> - Volunteer</li>
              <li><code>terry</code> - Volunteer</li>
            </ul>
          </div>

          <div className="button-instructions">
            <p><strong>To reset the database, click the button below:</strong></p>
          </div>

          <button
            className="seed-button danger-button"
            onClick={() => setShowConfirmModal(true)}
            disabled={isSeeding}
          >
            {isSeeding ? 'Resetting Database...' : 'Reset Database'}
          </button>
        </div>
      </div>

      {showConfirmModal && (
        <div className="modal-overlay" onClick={() => setShowConfirmModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h2>Confirm Database Reset</h2>
            
            <div className="modal-warning">
              <p>This will permanently delete all data and create fresh demo data.</p>
            </div>

            <div className="modal-body">
              <p>This action cannot be undone.</p>
            </div>

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
                {isSeeding ? 'Resetting...' : 'Reset Database'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DeveloperTab;
