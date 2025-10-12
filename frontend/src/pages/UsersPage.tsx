import React from 'react';
import './UsersPage.css';
import type { User, Group } from '../api/client';
import api from '../api/client';
import { usersApi, groupsApi } from '../api/client';

const UsersPage: React.FC = () => {
  const [users, setUsers] = React.useState<User[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Group modal state
  const [groupModalUser, setGroupModalUser] = React.useState<User | null>(null);
  const [allGroups, setAllGroups] = React.useState<Group[]>([]);
  const [groupModalLoading, setGroupModalLoading] = React.useState(false);
  const [groupModalError, setGroupModalError] = React.useState<string | null>(null);

  // Fetch users
  const fetchUsers = React.useCallback(() => {
    setLoading(true);
    setError(null);
    api.get<User[]>('/admin/users')
      .then(res => {
        setUsers(res.data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.response?.data?.error || 'Failed to fetch users');
        setLoading(false);
      });
  }, []);

  React.useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

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

  const handleDeactivate = async (user: User) => {
    try {
      await usersApi.deactivate(user.id);
      fetchUsers();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to deactivate user');
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
    try {
      if (assigned) {
        await usersApi.removeGroup(user.id, group.id);
      } else {
        await usersApi.assignGroup(user.id, group.id);
      }
      fetchUsers();
      // Refresh modal user
      setGroupModalUser(prev => prev && prev.id === user.id ? {
        ...prev,
        groups: assigned
          ? (prev.groups || []).filter(g => g.id !== group.id)
          : [...(prev.groups || []), group],
      } : prev);
    } catch (err: any) {
      setGroupModalError(err.response?.data?.error || 'Failed to update group');
    }
  };

  const closeGroupModal = () => {
    setGroupModalUser(null);
    setGroupModalError(null);
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
    setCreateData(d => ({
      ...d,
      groupIds: d.groupIds.includes(groupId)
        ? d.groupIds.filter(id => id !== groupId)
        : [...d.groupIds, groupId],
    }));
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
      <div className="users-create-bar">
        <button className="user-action-btn" onClick={() => setShowCreate(s => !s)}>
          {showCreate ? 'Cancel' : 'Add User'}
        </button>
      </div>
      {showCreate && (
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
        <table className="users-table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Email</th>
              <th>Admin</th>
              <th>Groups</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {users.map(user => (
              <tr key={user.id}>
                <td>{user.username}</td>
                <td>{user.email}</td>
                <td>{user.is_admin ? 'Yes' : 'No'}</td>
                <td>{user.groups?.map(g => g.name).join(', ') || '-'}</td>
                <td>{(user as any).deleted_at ? 'Deleted' : 'Active'}</td>
                <td>
                  <div className="user-actions">
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
                      title="Deactivate user"
                      disabled={(user as any).deleted_at}
                      onClick={() => handleDeactivate(user)}
                    >
                      Deactivate
                    </button>
                    <button
                      className="user-action-btn danger"
                      title="Delete user"
                      disabled={(user as any).deleted_at}
                      onClick={() => handleDelete(user)}
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
    </div>
  );
};
// (removed duplicate export default)
export default UsersPage;
