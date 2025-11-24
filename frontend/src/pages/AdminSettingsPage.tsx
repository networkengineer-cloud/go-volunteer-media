import React, { useState, useEffect } from 'react';
import './AdminSettingsPage.css';
import SiteSettingsTab from '../components/admin/SiteSettingsTab';
import CommentTagsTab from '../components/admin/CommentTagsTab';
import AnnouncementsTab from '../components/admin/AnnouncementsTab';
import DeveloperTab from '../components/admin/DeveloperTab';
import api from '../api/client';

const AdminSettingsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'site' | 'tags' | 'announcements' | 'developer'>('site');
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
            className={`tab-button ${activeTab === 'tags' ? 'active' : ''}`}
            onClick={() => setActiveTab('tags')}
          >
            Comment Tags
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
          {activeTab === 'tags' && <CommentTagsTab />}
          {activeTab === 'announcements' && <AnnouncementsTab />}
          {activeTab === 'developer' && isDevelopment && <DeveloperTab />}
        </div>
      </div>
    </div>
  );
};

export default AdminSettingsPage;
