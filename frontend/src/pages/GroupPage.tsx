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

  useEffect(() => {
    if (id) {
      loadGroupData(Number(id));
    }
  }, [id]);

  const loadGroupData = async (groupId: number) => {
    try {
      const [groupRes, animalsRes] = await Promise.all([
        groupsApi.getById(groupId),
        animalsApi.getAll(groupId),
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
        {animals.length === 0 ? (
          <p>No animals yet. Add one to get started!</p>
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
