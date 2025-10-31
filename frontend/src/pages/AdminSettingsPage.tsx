import React, { useState } from 'react';
import './AdminSettingsPage.css';
import SiteSettingsTab from '../components/admin/SiteSettingsTab';
import CommentTagsTab from '../components/admin/CommentTagsTab';

const AdminSettingsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'site' | 'tags'>('site');

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
        </div>

        <div className="tab-content">
          {activeTab === 'site' && <SiteSettingsTab />}
          {activeTab === 'tags' && <CommentTagsTab />}
        </div>
      </div>
    </div>
  );
};

export default AdminSettingsPage;
