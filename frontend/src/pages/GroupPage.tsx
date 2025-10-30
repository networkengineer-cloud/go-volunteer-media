import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { groupsApi, animalsApi } from '../api/client';
import type { Group, Animal } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import './GroupPage.css';




const GroupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { isAdmin } = useAuth();
  const [group, setGroup] = useState<Group | null>(null);
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [nameSearch, setNameSearch] = useState<string>('');

  useEffect(() => {
    if (id) {
      loadGroupData(Number(id));
    }
  }, [id, statusFilter, nameSearch]);

  const loadGroupData = async (groupId: number) => {
    try {
      const [groupRes, animalsRes] = await Promise.all([
        groupsApi.getById(groupId),
        animalsApi.getAll(groupId, statusFilter, nameSearch),
      ]);
      setGroup(groupRes.data);
      setAnimals(animalsRes.data);
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

      <div className="animals-section">
        <div className="section-header">
          <h2>Animals</h2>
          {isAdmin && (
            <Link to={`/groups/${id}/animals/new`} className="btn-primary">
              Add Animal
            </Link>
          )}
        </div>
        <div className="filters-section">
          <div className="filter-group">
            <label htmlFor="status-filter">Status:</label>
            <select
              id="status-filter"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
            >
              <option value="">Available (default)</option>
              <option value="all">All Animals</option>
              <option value="available">Available</option>
              <option value="adopted">Adopted</option>
              <option value="fostered">Fostered</option>
              <option value="archived">Archived</option>
            </select>
          </div>
          <div className="filter-group">
            <label htmlFor="name-search">Search by name:</label>
            <input
              id="name-search"
              type="text"
              placeholder="Enter animal name..."
              value={nameSearch}
              onChange={(e) => setNameSearch(e.target.value)}
            />
          </div>
        </div>
        {animals.length === 0 ? (
          <p>No animals found matching the filters.</p>
        ) : (
          <div className="animals-grid">
            {animals.map((animal) => (
              <Link
                key={animal.id}
                to={`/groups/${id}/animals/${animal.id}/view`}
                className="animal-card"
              >
                {animal.image_url && (
                  <img
                    src={animal.image_url}
                    alt={animal.name}
                    className="animal-card-image"
                  />
                )}
                <div className="animal-info">
                  <h3>{animal.name}</h3>
                  <p className="species">{animal.species} {animal.breed && `- ${animal.breed}`}</p>
                  <p className="age">{animal.age} years old</p>
                  <span className={`status ${animal.status}`}>{animal.status}</span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

// Only one export default should exist. Removed duplicate.
export default GroupPage;
