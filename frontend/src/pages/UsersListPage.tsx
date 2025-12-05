import React from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import { useAuth } from '../hooks/useAuth';
import { usersApi, groupsApi } from '../api/client';
import type { User, Group, GroupMember } from '../api/client';

// Create API instance for authenticated requests
const api = axios.create({
  baseURL: '/api',
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = 'Bearer ' + token;
  }
  return config;
});
import './UsersPage.css'; // Reuse existing styles

const UsersListPage: React.FC = () => {
  const { user: currentUser, isAdmin } = useAuth();
  const [users, setUsers] = React.useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = React.useState<User[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [searchQuery, setSearchQuery] = React.useState('');
  const [filterGroup, setFilterGroup] = React.useState<number | 'all'>('all');
  const [allGroups, setAllGroups] = React.useState<Group[]>([]);

  // Fetch users based on user role
  const fetchUsers = React.useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      if (isAdmin) {
        // Site admins see all users
        const usersRes = await usersApi.getAll();
        setUsers(usersRes.data);
      } else {
        // Regular users see active users from their groups
        // We need to fetch group members from each group they're in
        if (!currentUser?.groups || currentUser.groups.length === 0) {
          setUsers([]);
        } else {
          const allUsers = new Set<User>();

              // Fetch members from each group the user is in
            for (const group of currentUser.groups) {
              try {
                const membersRes = await api.get<GroupMember[]>(`/groups/${group.id}/members`);
                const members: GroupMember[] = membersRes.data;              // Convert GroupMember to User
              members.forEach(member => {
                allUsers.add({
                  id: member.user_id,
                  username: member.username,
                  email: member.email,
                  phone_number: member.phone_number,
                  is_admin: member.is_site_admin,
                  groups: [group],
                });
              });
            } catch (err) {
              console.error(`Failed to fetch members for group ${group.id}:`, err);
            }
          }

          // Merge users from all groups (avoiding duplicates)
          const userMap = new Map<number, User>();
          allUsers.forEach(u => {
            if (userMap.has(u.id)) {
              // User is in multiple groups, merge group arrays
              const existing = userMap.get(u.id)!;
              existing.groups = [...new Set([...(existing.groups || []), ...(u.groups || [])])];
            } else {
              userMap.set(u.id, u);
            }
          });

          setUsers(Array.from(userMap.values()));
        }
      }

      // Fetch all groups for the filter dropdown
      const groupsRes = await groupsApi.getAll();
      setAllGroups(groupsRes.data);

      setLoading(false);
    } catch (err: unknown) {
      setError(
        axios.isAxiosError(err) && err.response?.data?.error
          ? err.response.data.error
          : 'Failed to fetch users'
      );
      setLoading(false);
    }
  }, [isAdmin, currentUser?.groups]);

  React.useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Filter and search users
  React.useEffect(() => {
    let filtered = [...users];

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(user =>
        user.username.toLowerCase().includes(query) ||
        user.email.toLowerCase().includes(query)
      );
    }

    // Apply group filter
    if (filterGroup !== 'all' && !isAdmin) {
      // For regular users, filter by group they selected
      filtered = filtered.filter(user =>
        user.groups?.some(g => g.id === filterGroup)
      );
    } else if (filterGroup !== 'all' && isAdmin) {
      // For admins, filter by selected group
      filtered = filtered.filter(user =>
        user.groups?.some(g => g.id === filterGroup)
      );
    }

    // Sort alphabetically
    filtered.sort((a, b) => a.username.localeCompare(b.username));

    setFilteredUsers(filtered);
  }, [users, searchQuery, filterGroup, isAdmin]);

  const availableGroups = isAdmin ? allGroups : (currentUser?.groups || []);

  return (
    <div className="users-page">
      <h1>Users</h1>
      
      {/* Search and Filter Section */}
      <div className="users-filters">
        <div className="filter-row">
          <div className="search-box">
            <svg className="search-icon" width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M9 17A8 8 0 1 0 9 1a8 8 0 0 0 0 16zM18 18l-4.35-4.35" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
            <input
              type="text"
              className="search-input"
              placeholder="Search by username or email..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              aria-label="Search users"
            />
            {searchQuery && (
              <button
                className="clear-search"
                onClick={() => setSearchQuery('')}
                aria-label="Clear search"
              >
                ×
              </button>
            )}
          </div>

          {availableGroups.length > 1 && (
            <select
              className="filter-select"
              value={filterGroup}
              onChange={(e) => setFilterGroup(e.target.value === 'all' ? 'all' : parseInt(e.target.value))}
              aria-label="Filter by group"
            >
              <option value="all">All Groups</option>
              {availableGroups.map(group => (
                <option key={group.id} value={group.id}>{group.name}</option>
              ))}
            </select>
          )}
        </div>

        <div className="filter-summary">
          Showing {filteredUsers.length} of {users.length} user{users.length !== 1 ? 's' : ''}
          {searchQuery && <span> matching "{searchQuery}"</span>}
        </div>
      </div>

      {loading ? (
        <div className="users-loading">Loading users…</div>
      ) : error ? (
        <div className="users-error">{error}</div>
      ) : filteredUsers.length === 0 ? (
        <div className="users-empty">
          <p>{searchQuery ? 'No users match your search.' : 'No users found.'}</p>
        </div>
      ) : (
        <>
          {/* Desktop table view */}
          <table className="users-table">
            <thead>
              <tr>
                <th>Username</th>
                <th>Email</th>
                <th>Phone</th>
                {isAdmin && <th>Admin</th>}
                <th>Groups</th>
                <th>Profile</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map(user => (
                <tr key={user.id}>
                  <td>{user.username}</td>
                  <td>{user.email}</td>
                  <td>{user.phone_number || '-'}</td>
                  {isAdmin && <td>{user.is_admin ? 'Yes' : 'No'}</td>}
                  <td>
                    {user.groups && user.groups.length > 0 ? (
                      user.groups.map((g, index) => (
                        <React.Fragment key={g.id}>
                          <Link 
                            to={`/groups/${g.id}`}
                            className="group-link"
                            title={`View ${g.name} group`}
                          >
                            {g.name}
                          </Link>
                          {index < user.groups!.length - 1 && ', '}
                        </React.Fragment>
                      ))
                    ) : '-'}
                  </td>
                  <td>
                    <Link 
                      to={`/users/${user.id}/profile`}
                      className="user-action-btn"
                      title={`View ${user.username}'s profile`}
                    >
                      View
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {/* Mobile card view */}
          <div className="users-mobile-cards">
            {filteredUsers.map(user => (
              <div key={user.id} className="user-card">
                <div className="user-card-header">
                  <div className="user-card-title">
                    <div className="user-card-name">{user.username}</div>
                    <div className="user-card-email">{user.email}</div>
                  </div>
                  {isAdmin && user.is_admin && (
                    <span className="role-badge admin">Admin</span>
                  )}
                </div>
                <div className="user-card-info">
                  {user.phone_number && (
                    <div className="user-card-info-row">
                      <span className="user-card-info-label">Phone:</span>
                      <span className="user-card-info-value">{user.phone_number}</span>
                    </div>
                  )}
                  <div className="user-card-info-row">
                    <span className="user-card-info-label">Groups:</span>
                    <span className="user-card-info-value">
                      {user.groups && user.groups.length > 0 ? (
                        user.groups.map((g, index) => (
                          <React.Fragment key={g.id}>
                            <Link 
                              to={`/groups/${g.id}`}
                              className="group-link"
                              title={`View ${g.name} group`}
                            >
                              {g.name}
                            </Link>
                            {index < user.groups!.length - 1 && ', '}
                          </React.Fragment>
                        ))
                      ) : '-'}
                    </span>
                  </div>
                </div>
                <div className="user-card-actions">
                  <Link 
                    to={`/users/${user.id}/profile`}
                    className="user-action-btn"
                    title={`View ${user.username}'s profile`}
                  >
                    View Profile
                  </Link>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default UsersListPage;
