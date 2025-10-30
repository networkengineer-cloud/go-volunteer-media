import React from 'react';
import './UsersPage.css';
import type { User, Group } from '../api/client';
import { usersApi, groupsApi } from '../api/client';

const UsersPage: React.FC = () => {
  const [users, setUsers] = React.useState<User[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [showDeleted, setShowDeleted] = React.useState(false);

  // Group modal state
  const [groupModalUser, setGroupModalUser] = React.useState<User | null>(null);
  const [allGroups, setAllGroups] = React.useState<Group[]>([]);
  const [groupModalLoading, setGroupModalLoading] = React.useState(false);
  const [groupModalError, setGroupModalError] = React.useState<string | null>(null);

  // Fetch users (active or deleted)
  const fetchUsers = React.useCallback(() => {
    setLoading(true);
    setError(null);
    const apiCall = showDeleted ? usersApi.getDeleted() : usersApi.getAll();
    apiCall
      .then(res => {
        setUsers(res.data);
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
        <table className="users-table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Email</th>
              <th>Admin</th>
              <th>Groups</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map(user => (
              <tr key={user.id}>
                <td>{user.username}</td>
                <td>{user.email}</td>
                <td>{user.is_admin ? 'Yes' : 'No'}</td>
                <td>{user.groups?.map(g => g.name).join(', ') || '-'}</td>
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
