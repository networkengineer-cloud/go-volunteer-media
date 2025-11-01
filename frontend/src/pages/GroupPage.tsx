import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { groupsApi, animalsApi } from '../api/client';
import type { Group, Animal } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import './GroupPage.css';




const GroupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { isAdmin } = useAuth();
  const navigate = useNavigate();
  const [group, setGroup] = useState<Group | null>(null);
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [nameSearch, setNameSearch] = useState<string>('');

  useEffect(() => {
    if (id) {
      loadGroupData(Number(id));
    }
  }, [id, statusFilter, nameSearch]);

  const loadGroupData = async (groupId: number) => {
    try {
      setError('');
      const [groupRes, animalsRes] = await Promise.all([
        groupsApi.getById(groupId),
        animalsApi.getAll(groupId, statusFilter, nameSearch),
      ]);
      setGroup(groupRes.data);
      setAnimals(animalsRes.data);
    } catch (error: any) {
      console.error('Failed to load group data:', error);
      setError(error.response?.data?.error || 'Failed to load group information. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="group-page">
        <SkeletonLoader variant="rectangular" height="300px" />
        <div className="group-header">
          <SkeletonLoader variant="text" width="150px" />
          <SkeletonLoader variant="text" width="60%" height="2rem" />
          <SkeletonLoader variant="text" width="80%" />
        </div>
        <div className="skeleton-grid">
          <SkeletonLoader variant="card" count={6} />
        </div>
      </div>
    );
  }

  if (error || !group) {
    return (
      <div className="group-page">
        <ErrorState
          title={error ? 'Unable to Load Group' : 'Group Not Found'}
          message={error || 'The group you\'re looking for doesn\'t exist or you don\'t have permission to view it.'}
          onRetry={error ? () => {
            setLoading(true);
            loadGroupData(Number(id));
          } : undefined}
          onGoBack={() => navigate('/dashboard')}
          goBackLabel="Back to Dashboard"
        />
      </div>
    );
  }

  return (
    <div className="group-page">
      {group.hero_image_url && (
        <div 
          className="group-hero-image" 
          style={{ backgroundImage: `url(${group.hero_image_url})` }}
        />
      )}
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
              <option value="foster">Foster</option>
              <option value="bite_quarantine">Bite Quarantine</option>
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
          <EmptyState
            icon={
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            }
            title={statusFilter || nameSearch ? 'No animals match your filters' : `No animals in ${group.name} yet`}
            description={
              statusFilter || nameSearch
                ? 'Try adjusting your search or filter to see more results.'
                : isAdmin
                  ? `Get started by adding your first animal to ${group.name}. Animals added here will be visible to all group members.`
                  : `This group doesn't have any animals yet. An admin will need to add animals before volunteers can share updates.`
            }
            primaryAction={
              (statusFilter || nameSearch)
                ? {
                    label: 'Clear Filters',
                    onClick: () => {
                      setStatusFilter('');
                      setNameSearch('');
                    },
                  }
                : isAdmin
                  ? {
                      label: 'Add First Animal',
                      onClick: () => navigate(`/groups/${id}/animals/new`),
                    }
                  : undefined
            }
            secondaryAction={
              (statusFilter || nameSearch)
                ? {
                    label: 'View All Animals',
                    onClick: () => {
                      setStatusFilter('');
                      setNameSearch('');
                    },
                  }
                : undefined
            }
          />
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
                  {animal.breed && <p className="breed">{animal.breed}</p>}
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
