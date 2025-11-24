import React, { useState } from 'react';
import './AdminSettingsPage.css';
import SiteSettingsTab from '../components/admin/SiteSettingsTab';
import CommentTagsTab from '../components/admin/CommentTagsTab';
import AnnouncementsTab from '../components/admin/AnnouncementsTab';
import DeveloperTab from '../components/admin/DeveloperTab';

const AdminSettingsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'site' | 'tags' | 'announcements' | 'developer'>('site');

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
          {import.meta.env.DEV && (
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
          {activeTab === 'developer' && import.meta.env.DEV && <DeveloperTab />}
        </div>
      </div>
    </div>
  );
};

export default AdminSettingsPage;
