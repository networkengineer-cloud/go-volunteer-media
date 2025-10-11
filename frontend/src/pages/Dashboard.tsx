import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { groupsApi } from '../api/client';
import type { Group } from '../api/client';
import './Dashboard.css';

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadGroups();
  }, []);

  const loadGroups = async () => {
    try {
      const response = await groupsApi.getAll();
      setGroups(response.data);
    } catch (error) {
      console.error('Failed to load groups:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  return (
    <div className="dashboard">
      <h1>Welcome, {user?.username}!</h1>
      <h2>Your Groups</h2>
      {groups.length === 0 ? (
        <p>You haven't been assigned to any groups yet. Contact an admin to get started!</p>
      ) : (
        <div className="groups-grid">
          {groups.map((group) => (
            <Link key={group.id} to={`/groups/${group.id}`} className="group-card">
              <h3>{group.name}</h3>
              <p>{group.description}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export default Dashboard;
