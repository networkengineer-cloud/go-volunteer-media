import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import './AdminSettingsPage.css';
import SiteSettingsTab from '../components/admin/SiteSettingsTab';
import AnnouncementsTab from '../components/admin/AnnouncementsTab';
import DeveloperTab from '../components/admin/DeveloperTab';
import api from '../api/client';

const AdminSettingsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'site' | 'announcements' | 'developer'>('site');
  const [isDevelopment, setIsDevelopment] = useState(false);

  useEffect(() => {
    // Check environment from backend
    const checkEnvironment = async () => {
      try {
        const response = await api.get('/environment');
        setIsDevelopment(response.data.is_development);
      } catch (error) {
        console.error('Failed to check environment:', error);
      }
    };
    checkEnvironment();
  }, []);

  return (
    <div className="admin-settings-page">
      <div className="admin-settings-container">
        <h1>Admin Settings</h1>
        <p className="settings-subtitle">Manage site configuration and content</p>

        <div className="settings-tabs">
          <button
            className={`tab-button ${activeTab === 'site' ? 'active' : ''}`}
            onClick={() => setActiveTab('site')}
          >
            Site Settings
          </button>
          <button
            className={`tab-button ${activeTab === 'announcements' ? 'active' : ''}`}
            onClick={() => setActiveTab('announcements')}
          >
            Announcements
          </button>
          {isDevelopment && (
            <button
              className={`tab-button ${activeTab === 'developer' ? 'active' : ''}`}
              onClick={() => setActiveTab('developer')}
            >
              Developer
            </button>
          )}
        </div>

        <div className="tab-content">
          {activeTab === 'site' && <SiteSettingsTab />}
          {activeTab === 'announcements' && <AnnouncementsTab />}
          {activeTab === 'developer' && isDevelopment && <DeveloperTab />}
        </div>

        {/* Link to Tag Management */}
        <div className="settings-link-card">
          <div className="link-card-content">
            <span className="link-card-icon">🏷️</span>
            <div className="link-card-text">
              <h3>Tag Management</h3>
              <p>Manage animal tags and comment tags in one place</p>
            </div>
          </div>
          <Link to="/admin/animal-tags" className="link-card-button">
            Go to Tags →
          </Link>
        </div>
      </div>
    </div>
  );
};

export default AdminSettingsPage;
