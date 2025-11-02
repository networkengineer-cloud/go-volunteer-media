import React from 'react';
import { Link } from 'react-router-dom';
import './UsersPage.css';
import type { User, Group, UserStatistics } from '../api/client';
import { usersApi, groupsApi, statisticsApi } from '../api/client';

const UsersPage: React.FC = () => {
  const [users, setUsers] = React.useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = React.useState<User[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [showDeleted, setShowDeleted] = React.useState(false);
  const [statistics, setStatistics] = React.useState<Record<number, UserStatistics>>({});

  // Filter and search state
  const [searchQuery, setSearchQuery] = React.useState('');
  const [filterGroup, setFilterGroup] = React.useState<number | 'all'>('all');
  const [filterAdmin, setFilterAdmin] = React.useState<'all' | 'admin' | 'user'>('all');
  const [sortBy, setSortBy] = React.useState<'name' | 'email' | 'last_active' | 'most_active'>('name');
  const [sortOrder, setSortOrder] = React.useState<'asc' | 'desc'>('asc');

  // Group modal state
  const [groupModalUser, setGroupModalUser] = React.useState<User | null>(null);
  const [allGroups, setAllGroups] = React.useState<Group[]>([]);
  const [groupModalLoading, setGroupModalLoading] = React.useState(false);
  const [groupModalError, setGroupModalError] = React.useState<string | null>(null);

  // Password reset modal state
  const [resetPasswordUser, setResetPasswordUser] = React.useState<User | null>(null);
  const [newPassword, setNewPassword] = React.useState('');
  const [resetPasswordLoading, setResetPasswordLoading] = React.useState(false);
  const [resetPasswordError, setResetPasswordError] = React.useState<string | null>(null);
  const [resetPasswordSuccess, setResetPasswordSuccess] = React.useState<string | null>(null);

  // Fetch users and statistics (active or deleted)
  const fetchUsers = React.useCallback(() => {
    setLoading(true);
    setError(null);
    const apiCall = showDeleted ? usersApi.getDeleted() : usersApi.getAll();
    
    Promise.all([
      apiCall,
      statisticsApi.getUserStatistics(),
      groupsApi.getAll() // Fetch groups for filter
    ])
      .then(([usersRes, statsRes, groupsRes]) => {
        setUsers(usersRes.data);
        setAllGroups(groupsRes.data);
        
        // Create a map of user_id to statistics
        const statsMap: Record<number, UserStatistics> = {};
        statsRes.data.forEach(stat => {
          statsMap[stat.user_id] = stat;
        });
        setStatistics(statsMap);
        
        setLoading(false);
      })
      .catch(err => {
        setError(err.response?.data?.error || 'Failed to fetch users');
        setLoading(false);
      });
  }, [showDeleted]);

  React.useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Filter and sort users
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
    if (filterGroup !== 'all') {
      filtered = filtered.filter(user =>
        user.groups?.some(g => g.id === filterGroup)
      );
    }

    // Apply admin filter
    if (filterAdmin !== 'all') {
      filtered = filtered.filter(user =>
        filterAdmin === 'admin' ? user.is_admin : !user.is_admin
      );
    }

    // Apply sorting
    filtered.sort((a, b) => {
      let comparison = 0;
      const statsA = statistics[a.id];
      const statsB = statistics[b.id];

      switch (sortBy) {
        case 'name':
          comparison = a.username.localeCompare(b.username);
          break;
        case 'email':
          comparison = a.email.localeCompare(b.email);
          break;
        case 'last_active':
          const lastActiveA = statsA?.last_active ? new Date(statsA.last_active).getTime() : 0;
          const lastActiveB = statsB?.last_active ? new Date(statsB.last_active).getTime() : 0;
          comparison = lastActiveB - lastActiveA; // Most recent first
          break;
        case 'most_active':
          const commentsA = statsA?.comment_count || 0;
          const commentsB = statsB?.comment_count || 0;
          comparison = commentsB - commentsA; // Most comments first
          break;
      }

      return sortOrder === 'asc' ? comparison : -comparison;
    });

    setFilteredUsers(filtered);
  }, [users, searchQuery, filterGroup, filterAdmin, sortBy, sortOrder, statistics]);

  // Admin actions
  const handlePromoteDemote = async (user: User) => {
    try {
      if (user.is_admin) {
        await usersApi.demote(user.id);
      } else {
        await usersApi.promote(user.id);
      }
      fetchUsers();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to update admin status');
    }
  };


  const handleDelete = async (user: User) => {
    if (!window.confirm(`Delete user ${user.username}? This cannot be undone.`)) return;
    try {
      await usersApi.delete(user.id);
      fetchUsers();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete user');
    }
  };

  // Restore deleted user
  const handleRestore = async (user: User) => {
    try {
      await usersApi.restore(user.id);
      fetchUsers();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to restore user');
    }
  };

  // Group assignment modal
  const openGroupModal = async (user: User) => {
    setGroupModalUser(user);
    setGroupModalLoading(true);
    setGroupModalError(null);
    try {
      const res = await groupsApi.getAll();
      setAllGroups(res.data);
    } catch (err: any) {
      setGroupModalError(err.response?.data?.error || 'Failed to fetch groups');
    } finally {
      setGroupModalLoading(false);
    }
  };

  const handleGroupToggle = async (user: User, group: Group, assigned: boolean) => {
    const dogsGroup = allGroups.find(g => g.name.toLowerCase() === 'dogs');
    const modSquadGroup = allGroups.find(g => g.name.toLowerCase() === 'modsquad');
    
    try {
      if (assigned) {
        // Removing a group
        await usersApi.removeGroup(user.id, group.id);
        
        // If removing Dogs, also remove ModSquad (since ModSquad is a sub-group of Dogs)
        if (dogsGroup && modSquadGroup && group.id === dogsGroup.id) {
          const hasModSquad = user.groups?.some(g => g.id === modSquadGroup.id);
          if (hasModSquad) {
            await usersApi.removeGroup(user.id, modSquadGroup.id);
          }
        }
      } else {
        // Adding a group
        await usersApi.assignGroup(user.id, group.id);
        
        // If adding ModSquad, also add Dogs (since ModSquad is a sub-group of Dogs)
        if (modSquadGroup && dogsGroup && group.id === modSquadGroup.id) {
          const hasDogs = user.groups?.some(g => g.id === dogsGroup.id);
          if (!hasDogs) {
            await usersApi.assignGroup(user.id, dogsGroup.id);
          }
        }
      }
      fetchUsers();
      // Refresh modal user after all operations
      const userRes = await usersApi.getAll();
      const updatedUser = userRes.data.find((u: User) => u.id === user.id);
      if (updatedUser) {
        setGroupModalUser(updatedUser);
      }
    } catch (err: any) {
      setGroupModalError(err.response?.data?.error || 'Failed to update group');
    }
  };

  const closeGroupModal = () => {
    setGroupModalUser(null);
    setGroupModalError(null);
  };

  // Password reset modal functions
  const openPasswordResetModal = (user: User) => {
    setResetPasswordUser(user);
    setNewPassword('');
    setResetPasswordError(null);
    setResetPasswordSuccess(null);
  };

  const closePasswordResetModal = () => {
    setResetPasswordUser(null);
    setNewPassword('');
    setResetPasswordError(null);
    setResetPasswordSuccess(null);
  };

  const handlePasswordReset = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!resetPasswordUser) return;
    
    setResetPasswordLoading(true);
    setResetPasswordError(null);
    setResetPasswordSuccess(null);
    
    try {
      await usersApi.resetPassword(resetPasswordUser.id, newPassword);
      setResetPasswordSuccess('Password reset successfully');
      setTimeout(() => {
        closePasswordResetModal();
      }, 1500);
    } catch (err: any) {
      setResetPasswordError(err.response?.data?.error || 'Failed to reset password');
    } finally {
      setResetPasswordLoading(false);
    }
  };

  // Admin user creation form state
  const [showCreate, setShowCreate] = React.useState(false);
  const [createData, setCreateData] = React.useState({ username: '', email: '', password: '', is_admin: false, groupIds: [] as number[] });
  const [createLoading, setCreateLoading] = React.useState(false);
  const [createError, setCreateError] = React.useState<string | null>(null);
  const [createSuccess, setCreateSuccess] = React.useState<string | null>(null);


  // Fetch all groups for create form
  React.useEffect(() => {
    if (showCreate && allGroups.length === 0) {
      groupsApi.getAll().then(res => setAllGroups(res.data)).catch(() => {});
    }
  }, [showCreate, allGroups.length]);

  const handleCreateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setCreateData(d => ({ ...d, [name]: type === 'checkbox' ? checked : value }));
  };

  const handleCreateGroupToggle = (groupId: number) => {
    const dogsGroup = allGroups.find(g => g.name.toLowerCase() === 'dogs');
    const modSquadGroup = allGroups.find(g => g.name.toLowerCase() === 'modsquad');
    
    setCreateData(d => {
      const isSelected = d.groupIds.includes(groupId);
      let newGroupIds: number[];
      
      if (isSelected) {
        // Deselecting a group
        newGroupIds = d.groupIds.filter(id => id !== groupId);
        
        // If deselecting Dogs, also deselect ModSquad (since ModSquad is a sub-group of Dogs)
        if (dogsGroup && modSquadGroup && groupId === dogsGroup.id && d.groupIds.includes(modSquadGroup.id)) {
          newGroupIds = newGroupIds.filter(id => id !== modSquadGroup.id);
        }
      } else {
        // Selecting a group
        newGroupIds = [...d.groupIds, groupId];
        
        // If selecting ModSquad, also select Dogs (since ModSquad is a sub-group of Dogs)
        if (modSquadGroup && dogsGroup && groupId === modSquadGroup.id && !d.groupIds.includes(dogsGroup.id)) {
          newGroupIds.push(dogsGroup.id);
        }
      }
      
      return { ...d, groupIds: newGroupIds };
    });
  };

  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateLoading(true);
    setCreateError(null);
    setCreateSuccess(null);
    try {
      await usersApi.create({
        username: createData.username,
        email: createData.email,
        password: createData.password,
        is_admin: createData.is_admin,
        group_ids: createData.groupIds,
      });
      setCreateSuccess('User created successfully');
      setCreateData({ username: '', email: '', password: '', is_admin: false, groupIds: [] });
      fetchUsers();
      setShowCreate(false);
    } catch (err: any) {
      setCreateError(err.response?.data?.error || 'Failed to create user');
    } finally {
      setCreateLoading(false);
    }
  };

  return (
    <div className="users-page">
      <h1>Manage Users</h1>
      <div className="users-create-bar" style={{display: 'flex', gap: '1rem', alignItems: 'center'}}>
        <button className="user-action-btn" onClick={() => setShowCreate(s => !s)}>
          {showCreate ? 'Cancel' : 'Add User'}
        </button>
        <button
          className="user-action-btn"
          style={{background: showDeleted ? 'var(--brand, #0e6c55)' : undefined, color: showDeleted ? '#fff' : undefined}}
          onClick={() => setShowDeleted(v => !v)}
        >
          {showDeleted ? 'Show Active Users' : 'Show Deleted Users'}
        </button>
      </div>

      {/* Search and Filter Section */}
      {!showDeleted && (
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

            <select
              className="filter-select"
              value={filterGroup}
              onChange={(e) => setFilterGroup(e.target.value === 'all' ? 'all' : parseInt(e.target.value))}
              aria-label="Filter by group"
            >
              <option value="all">All Groups</option>
              {allGroups.map(group => (
                <option key={group.id} value={group.id}>{group.name}</option>
              ))}
            </select>

            <select
              className="filter-select"
              value={filterAdmin}
              onChange={(e) => setFilterAdmin(e.target.value as 'all' | 'admin' | 'user')}
              aria-label="Filter by role"
            >
              <option value="all">All Roles</option>
              <option value="admin">Admins</option>
              <option value="user">Users</option>
            </select>

            <select
              className="filter-select"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as 'name' | 'email' | 'last_active' | 'most_active')}
              aria-label="Sort by"
            >
              <option value="name">Sort by Name</option>
              <option value="email">Sort by Email</option>
              <option value="last_active">Sort by Last Active</option>
              <option value="most_active">Sort by Most Active</option>
            </select>

            <button
              className="sort-order-btn"
              onClick={() => setSortOrder(order => order === 'asc' ? 'desc' : 'asc')}
              aria-label={`Sort order: ${sortOrder === 'asc' ? 'ascending' : 'descending'}`}
              title={sortOrder === 'asc' ? 'Ascending' : 'Descending'}
            >
              {sortOrder === 'asc' ? (
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M10 5v10M5 10l5-5 5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              ) : (
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M10 15V5M15 10l-5 5-5-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              )}
            </button>
          </div>
          
          <div className="filter-summary">
            Showing {filteredUsers.length} of {users.length} users
            {searchQuery && <span> matching "{searchQuery}"</span>}
          </div>
        </div>
      )}

      {showCreate && !showDeleted && (
        <form className="users-create-form" onSubmit={handleCreateSubmit}>
          <div>
            <label>
              Username
              <input name="username" value={createData.username} onChange={handleCreateChange} required minLength={3} maxLength={50} autoComplete="off" />
            </label>
          </div>
          <div>
            <label>
              Email
              <input name="email" type="email" value={createData.email} onChange={handleCreateChange} required autoComplete="off" />
            </label>
          </div>
          <div>
            <label>
              Password
              <input name="password" type="password" value={createData.password} onChange={handleCreateChange} required minLength={8} maxLength={72} autoComplete="new-password" />
            </label>
          </div>
          <div>
            <label>
              <input name="is_admin" type="checkbox" checked={createData.is_admin} onChange={handleCreateChange} />
              Admin
            </label>
          </div>
          <div>
            <label>Assign Groups</label>
            <ul className="group-list">
              {allGroups.map(group => (
                <li key={group.id}>
                  <label>
                    <input
                      type="checkbox"
                      checked={createData.groupIds.includes(group.id)}
                      onChange={() => handleCreateGroupToggle(group.id)}
                    />
                    {group.name}
                  </label>
                </li>
              ))}
            </ul>
          </div>
          <button className="user-action-btn" type="submit" disabled={createLoading}>
            {createLoading ? 'Creating…' : 'Create User'}
          </button>
          {createError && <div className="users-error">{createError}</div>}
          {createSuccess && <div className="users-success">{createSuccess}</div>}
        </form>
      )}
      {loading ? (
        <div className="users-loading">Loading users…</div>
      ) : error ? (
        <div className="users-error">{error}</div>
      ) : (
        <>
          {/* Desktop table view */}
          <table className="users-table">
            <thead>
              <tr>
                <th>Username</th>
                <th>Email</th>
                <th>Admin</th>
                <th>Groups</th>
                <th>Comments</th>
                <th>Animals</th>
                <th>Last Active</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map(user => {
                const stats = statistics[user.id];
                return (
                  <tr key={user.id}>
                    <td>
                      <Link 
                        to={`/users/${user.id}/profile`}
                        className="username-link"
                        title={`View ${user.username}'s profile`}
                      >
                        {user.username}
                      </Link>
                    </td>
                    <td>{user.email}</td>
                    <td>{user.is_admin ? 'Yes' : 'No'}</td>
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
                    <td className="user-stat">
                      {stats ? (
                        <span className="stat-badge" title={`${stats.comment_count} comment${stats.comment_count !== 1 ? 's' : ''}`}>
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                          </svg>
                          {stats.comment_count}
                        </span>
                      ) : '—'}
                    </td>
                    <td className="user-stat">
                      {stats ? (
                        <span className="stat-badge" title={`Interacted with ${stats.animals_interacted_with} animal${stats.animals_interacted_with !== 1 ? 's' : ''}`}>
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <circle cx="11" cy="4" r="2"></circle>
                            <circle cx="18" cy="8" r="2"></circle>
                            <circle cx="20" cy="16" r="2"></circle>
                            <circle cx="4" cy="16" r="2"></circle>
                            <circle cx="4" cy="8" r="2"></circle>
                          </svg>
                          {stats.animals_interacted_with}
                        </span>
                      ) : '—'}
                    </td>
                    <td className="user-stat">
                      {stats?.last_active ? (
                        <span className="last-activity" title={new Date(stats.last_active).toLocaleString()}>
                          {formatRelativeTime(stats.last_active)}
                        </span>
                      ) : (
                        <span className="no-activity">No activity</span>
                      )}
                    </td>
                    <td>{showDeleted ? 'Deleted' : 'Active'}</td>
                    <td>
                      <div className="user-actions">
                        {showDeleted ? (
                          <button
                            className="user-action-btn"
                            title="Restore user"
                            onClick={() => handleRestore(user)}
                          >
                            Restore
                          </button>
                        ) : (
                          <>
                            <button
                              className="user-action-btn"
                              title={user.is_admin ? 'Demote from admin' : 'Promote to admin'}
                              disabled={(user as any).deleted_at}
                              onClick={() => handlePromoteDemote(user)}
                            >
                              {user.is_admin ? 'Demote' : 'Promote'}
                            </button>
                            <button
                              className="user-action-btn"
                              title="Assign/Remove Group"
                              disabled={(user as any).deleted_at}
                              onClick={() => openGroupModal(user)}
                            >
                              Groups
                            </button>
                            <button
                              className="user-action-btn"
                              title="Reset password"
                            disabled={(user as any).deleted_at}
                            onClick={() => openPasswordResetModal(user)}
                          >
                            Reset Password
                          </button>
                          {/* Deactivate button removed; Delete now performs soft-delete/deactivation */}
                          <button
                            className="user-action-btn danger"
                            title="Delete user"
                            disabled={(user as any).deleted_at}
                            onClick={() => handleDelete(user)}
                          >
                            Delete
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              );
              })}
            </tbody>
          </table>

          {/* Mobile card view */}
          <div className="users-mobile-cards">
            {filteredUsers.map(user => (
              <div key={user.id} className="user-card">
                <div className="user-card-header">
                  <div className="user-card-title">
                    <div className="user-card-name">
                      <Link 
                        to={`/users/${user.id}/profile`}
                        className="username-link"
                        title={`View ${user.username}'s profile`}
                      >
                        {user.username}
                      </Link>
                    </div>
                    <div className="user-card-email">{user.email}</div>
                  </div>
                  {user.is_admin && (
                    <span className="role-badge admin">Admin</span>
                  )}
                </div>
                <div className="user-card-info">
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
                  <div className="user-card-info-row">
                    <span className="user-card-info-label">Status:</span>
                    <span className="user-card-info-value">
                      {showDeleted ? 'Deleted' : 'Active'}
                    </span>
                  </div>
                </div>
                <div className="user-card-actions">
                  {showDeleted ? (
                    <button
                      className="user-action-btn"
                      onClick={() => handleRestore(user)}
                    >
                      Restore
                    </button>
                  ) : (
                    <>
                      <button
                        className="user-action-btn"
                        disabled={(user as any).deleted_at}
                        onClick={() => handlePromoteDemote(user)}
                      >
                        {user.is_admin ? 'Demote' : 'Promote'}
                      </button>
                      <button
                        className="user-action-btn"
                        disabled={(user as any).deleted_at}
                        onClick={() => openGroupModal(user)}
                      >
                        Groups
                      </button>
                      <button
                        className="user-action-btn"
                        disabled={(user as any).deleted_at}
                        onClick={() => openPasswordResetModal(user)}
                      >
                        Reset Password
                      </button>
                      <button
                        className="user-action-btn danger"
                        disabled={(user as any).deleted_at}
                        onClick={() => handleDelete(user)}
                      >
                        Delete
                      </button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
      {/* Group assignment modal */}
      {groupModalUser && (
        <div className="group-modal-backdrop" onClick={closeGroupModal}>
          <div className="group-modal" onClick={e => e.stopPropagation()}>
            <h2>Manage Groups for {groupModalUser.username}</h2>
            {groupModalLoading ? (
              <div className="users-loading">Loading groups…</div>
            ) : groupModalError ? (
              <div className="users-error">{groupModalError}</div>
            ) : (
              <ul className="group-list">
                {allGroups.map(group => {
                  const assigned = !!groupModalUser.groups?.some(g => g.id === group.id);
                  return (
                    <li key={group.id}>
                      <label>
                        <input
                          type="checkbox"
                          checked={assigned}
                          onChange={() => handleGroupToggle(groupModalUser, group, assigned)}
                        />
                        {group.name}
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
            <button className="user-action-btn" onClick={closeGroupModal} style={{marginTop: '1rem'}}>Close</button>
          </div>
        </div>
      )}
      {/* Password reset modal */}
      {resetPasswordUser && (
        <div className="group-modal-backdrop" onClick={closePasswordResetModal}>
          <div className="group-modal" onClick={e => e.stopPropagation()}>
            <h2>Reset Password for {resetPasswordUser.username}</h2>
            <form onSubmit={handlePasswordReset}>
              <div style={{marginBottom: '1rem'}}>
                <label>
                  New Password
                  <input
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    required
                    minLength={8}
                    maxLength={72}
                    autoComplete="new-password"
                    style={{width: '100%', padding: '0.5rem', marginTop: '0.5rem'}}
                  />
                </label>
              </div>
              {resetPasswordError && <div className="users-error">{resetPasswordError}</div>}
              {resetPasswordSuccess && <div className="users-success">{resetPasswordSuccess}</div>}
              <div style={{display: 'flex', gap: '0.5rem', marginTop: '1rem'}}>
                <button
                  type="submit"
                  className="user-action-btn"
                  disabled={resetPasswordLoading || !newPassword}
                >
                  {resetPasswordLoading ? 'Resetting…' : 'Reset Password'}
                </button>
                <button
                  type="button"
                  className="user-action-btn"
                  onClick={closePasswordResetModal}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

// Helper function to format relative time
function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export default UsersPage;
