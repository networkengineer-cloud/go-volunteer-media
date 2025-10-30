import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { groupsApi, animalsApi, updatesApi } from '../api/client';
import type { Group, Animal, Update } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import './GroupPage.css';




const GroupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { isAdmin } = useAuth();
  const [group, setGroup] = useState<Group | null>(null);
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [updates, setUpdates] = useState<Update[]>([]);
  const [activeTab, setActiveTab] = useState<'animals' | 'updates'>('animals');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      loadGroupData(Number(id));
    }
    // eslint-disable-next-line
  }, [id]);

  const loadGroupData = async (groupId: number) => {
    try {
      const [groupRes, animalsRes, updatesRes] = await Promise.all([
        groupsApi.getById(groupId),
        animalsApi.getAll(groupId),
        updatesApi.getAll(groupId),
      ]);
      setGroup(groupRes.data);
      setAnimals(animalsRes.data);
      setUpdates(updatesRes.data);
    } catch (error) {
      console.error('Failed to load group data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  if (!group) {
    return <div className="error">Group not found</div>;
  }

  return (
    <div className="group-page">
      <div className="group-header">
        <Link to="/" className="back-link">← Back to Dashboard</Link>
        <h1>{group.name}</h1>
        <p>{group.description}</p>
      </div>

      <div className="tabs">
        <button
          className={activeTab === 'animals' ? 'tab active' : 'tab'}
          onClick={() => setActiveTab('animals')}
        >
          Animals ({animals.length})
        </button>
        <button
          className={activeTab === 'updates' ? 'tab active' : 'tab'}
          onClick={() => setActiveTab('updates')}
        >
          Updates ({updates.length})
        </button>
      </div>

      {activeTab === 'animals' && (
        <div className="animals-section">
          <div className="section-header">
            <h2>Animals</h2>
            {isAdmin && (
              <Link to={`/groups/${id}/animals/new`} className="btn-primary">
                Add Animal
              </Link>
            )}
          </div>
          {animals.length === 0 ? (
            <p>No animals yet. Add one to get started!</p>
          ) : (
            <div className="animals-grid">
              {animals.map((animal) => (
                <Link
                  key={animal.id}
                  to={`/groups/${id}/animals/${animal.id}`}
                  className="animal-card"
                >
                  {animal.image_url && (
                    <img
                      src={animal.image_url}
                      alt={animal.name}
                    />
                  )}
                  <div className="animal-info">
                    <p className="species">{animal.species} {animal.breed && `- ${animal.breed}`}</p>
                    <p className="age">{animal.age} years old</p>
                    <span className={`status ${animal.status}`}>{animal.status}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'updates' && (
        <div className="updates-section">
          <div className="section-header">
            <h2>Updates</h2>
            <Link to={`/groups/${id}/updates/new`} className="btn-primary">
              New Update
            </Link>
          </div>
          {updates.length === 0 ? (
            <p>No updates yet. Be the first to share!</p>
          ) : (
            <div className="updates-list">
              {updates.map((update) => (
                <div key={update.id} className="update-card">
                  <div className="update-header">
                    <h3>{update.title}</h3>
                    <span className="update-meta">
                      by {update.user?.username} • {new Date(update.created_at).toLocaleDateString()}
                    </span>
                  </div>
                  <p className="update-content">{update.content}</p>
                  {update.image_url && (
                    <img src={update.image_url} alt={update.title} className="update-image" />
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Only one export default should exist. Removed duplicate.
export default GroupPage;
