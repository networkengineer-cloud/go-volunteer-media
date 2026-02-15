import React from 'react';
import axios from 'axios';
import { Link } from 'react-router-dom';
import './UsersPage.css';
import type { User, Group, UserStatistics, GroupMember } from '../api/client';
import { usersApi, groupsApi, statisticsApi, groupAdminApi } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { getPasswordStrength } from '../utils/passwordStrength';

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

const UsersPage: React.FC = () => {
  const { user: currentUser, isAdmin, isGroupAdmin } = useAuth();
  const canManageUsers = isAdmin || isGroupAdmin;
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

  // Group admin management state
  const [groupMembers, setGroupMembers] = React.useState<Map<number, GroupMember[]>>(new Map());
  const [updatingGroupAdmin, setUpdatingGroupAdmin] = React.useState<{userId: number, groupId: number} | null>(null);

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

  // Show details state - track which user cards have expanded details
  const [expandedDetails, setExpandedDetails] = React.useState<Set<number>>(new Set());

  // Fetch group members with admin status for group admin management
  const fetchGroupMembers = React.useCallback(async (groups: Group[]) => {
    if ((!isAdmin && !isGroupAdmin) || groups.length === 0) return;

    try {
      const membersMap = new Map<number, GroupMember[]>();
      for (const group of groups) {
        const membersRes = await groupAdminApi.getMembers(group.id);
        membersMap.set(group.id, membersRes.data);
      }
      setGroupMembers(membersMap);
    } catch (err) {
      console.error('Failed to fetch group members:', err);
    }
  }, [isAdmin, isGroupAdmin]);

  // Fetch users and statistics (active or deleted)
  const fetchUsers = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      if (isAdmin) {
        // Site admins see all users with full statistics
        // Normalize both endpoints to return User[] so they can run in parallel
        const usersPromise = showDeleted
          ? usersApi.getDeleted().then(res => res.data)
          : usersApi.getAll().then(res => res.data.data);
        const [usersList, statsRes, groupsRes] = await Promise.all([
          usersPromise,
          statisticsApi.getUserStatistics(),
          groupsApi.getAll()
        ]);

        setUsers(usersList);
        setAllGroups(groupsRes.data);

        // Create a map of user_id to statistics
        const statsMap: Record<number, UserStatistics> = {};
        statsRes.data.data.forEach(stat => {
          statsMap[stat.user_id] = stat;
        });
        setStatistics(statsMap);
      } else {
        // Group admins see users from their groups only
        if (!currentUser?.groups || currentUser.groups.length === 0) {
          setUsers([]);
          setAllGroups([]);
        } else {
          const allUsers = new Set<User>();
          
          // Fetch members from each group the user is in
          for (const group of currentUser.groups) {
            try {
              const membersRes = await api.get<GroupMember[]>(`/groups/${group.id}/members`);
              const members: GroupMember[] = membersRes.data;
              
              // Convert GroupMember to User
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
          setAllGroups(currentUser.groups);
        }
      }
      
      setLoading(false);
    } catch (err) {
      setError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to fetch users');
      setLoading(false);
    }
  }, [isAdmin, currentUser?.groups, showDeleted]);

  React.useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Fetch group members when allGroups is populated (for admin group admin management)
  React.useEffect(() => {
    if (allGroups.length > 0 && (isAdmin || isGroupAdmin)) {
      fetchGroupMembers(allGroups);
    }
  }, [allGroups, isAdmin, isGroupAdmin, fetchGroupMembers]);

  // Toggle group admin status
  const handleToggleGroupAdmin = async (userId: number, groupId: number, isCurrentlyAdmin: boolean) => {
    // Site admins can toggle any group, group admins can only toggle their own groups
    if (!isAdmin && !isCurrentUserGroupAdminOf(groupId)) return;

    setUpdatingGroupAdmin({ userId, groupId });
    try {
      if (isCurrentlyAdmin) {
        await groupAdminApi.demoteFromGroupAdmin(groupId, userId);
      } else {
        await groupAdminApi.promoteToGroupAdmin(groupId, userId);
      }
      
      // Refresh the group members for this group
      const membersRes = await groupAdminApi.getMembers(groupId);
      setGroupMembers(prev => {
        const newMap = new Map(prev);
        newMap.set(groupId, membersRes.data);
        return newMap;
      });
    } catch (err) {
      console.error('Failed to toggle group admin status:', err);
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Failed to update group admin status');
      setTimeout(() => setError(null), 5000);
    } finally {
      setUpdatingGroupAdmin(null);
    }
  };

  // Check if a user is a group admin for a specific group
  const isUserGroupAdmin = (userId: number, groupId: number): boolean => {
    const members = groupMembers.get(groupId);
    if (!members) return false;
    const member = members.find(m => m.user_id === userId);
    return member?.is_group_admin || false;
  };

  // Check if current user is a group admin of a specific group
  const isCurrentUserGroupAdminOf = (groupId: number): boolean => {
    if (!currentUser) return false;
    const members = groupMembers.get(groupId);
    if (!members) return false;
    const member = members.find(m => m.user_id === currentUser.id);
    return member?.is_group_admin || false;
  };

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
        case 'last_active': {
          const lastActiveA = statsA?.last_active ? new Date(statsA.last_active).getTime() : 0;
          const lastActiveB = statsB?.last_active ? new Date(statsB.last_active).getTime() : 0;
          comparison = lastActiveB - lastActiveA; // Most recent first
          break;
        }
        case 'most_active': {
          const commentsA = statsA?.comment_count || 0;
          const commentsB = statsB?.comment_count || 0;
          comparison = commentsB - commentsA; // Most comments first
          break;
        }
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
    } catch (err: unknown) {
      setError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to update admin status');
    }
  };


  const handleDelete = async (user: User) => {
    if (!window.confirm(`Delete user ${user.username}? This cannot be undone.`)) return;
    try {
      await usersApi.delete(user.id);
      fetchUsers();
    } catch (err: unknown) {
      setError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to delete user');
    }
  };

  // Restore deleted user
  const handleRestore = async (user: User) => {
    try {
      await usersApi.restore(user.id);
      fetchUsers();
    } catch (err: unknown) {
      setError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to restore user');
    }
  };

  // Group assignment modal
  const openGroupModal = async (user: User) => {
    setGroupModalUser(user);
    setGroupModalLoading(true);
    setGroupModalError(null);
    try {
      const res = await groupsApi.getAll();
      let groupsToShow = res.data;
      
      // If current user is a GroupAdmin (but not site admin), only show groups they admin
      if (isGroupAdmin && !isAdmin && currentUser?.groups) {
        groupsToShow = groupsToShow.filter(group => {
          // Check if current user is a group admin of this group
          const membership = groupMembers.get(group.id);
          if (!membership) return false;
          const currentUserMembership = membership.find(m => m.user_id === currentUser.id);
          return currentUserMembership?.is_group_admin === true;
        });
      }
      
      setAllGroups(groupsToShow);
    } catch (err: unknown) {
      setGroupModalError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to fetch groups');
    } finally {
      setGroupModalLoading(false);
    }
  };

  const handleGroupToggle = async (user: User, group: Group, assigned: boolean) => {
    const dogsGroup = allGroups.find(g => g.name.toLowerCase() === 'dogs');
    const modSquadGroup = allGroups.find(g => g.name.toLowerCase() === 'modsquad');
    
    try {
      // Use appropriate API based on user role
      // Site admins use admin endpoints, GroupAdmins use group admin endpoints
      const useAdminEndpoint = isAdmin;
      const useGroupAdminEndpoint = !isAdmin && isGroupAdmin;
      
      if (assigned) {
        // Removing a group
        if (useAdminEndpoint) {
          await usersApi.removeGroup(user.id, group.id);
        } else if (useGroupAdminEndpoint) {
          await groupAdminApi.removeMemberFromGroup(group.id, user.id);
        }
        
        // If removing Dogs, also remove ModSquad (since ModSquad is a sub-group of Dogs)
        if (dogsGroup && modSquadGroup && group.id === dogsGroup.id) {
          const hasModSquad = user.groups?.some(g => g.id === modSquadGroup.id);
          if (hasModSquad) {
            if (useAdminEndpoint) {
              await usersApi.removeGroup(user.id, modSquadGroup.id);
            } else if (useGroupAdminEndpoint) {
              await groupAdminApi.removeMemberFromGroup(modSquadGroup.id, user.id);
            }
          }
        }
      } else {
        // Adding a group
        if (useAdminEndpoint) {
          await usersApi.assignGroup(user.id, group.id);
        } else if (useGroupAdminEndpoint) {
          await groupAdminApi.addMemberToGroup(group.id, user.id);
        }
        
        // If adding ModSquad, also add Dogs (since ModSquad is a sub-group of Dogs)
        if (modSquadGroup && dogsGroup && group.id === modSquadGroup.id) {
          const hasDogs = user.groups?.some(g => g.id === dogsGroup.id);
          if (!hasDogs) {
            if (useAdminEndpoint) {
              await usersApi.assignGroup(user.id, dogsGroup.id);
            } else if (useGroupAdminEndpoint) {
              await groupAdminApi.addMemberToGroup(dogsGroup.id, user.id);
            }
          }
        }
      }
      // Optimistically update the modal user's groups based on the toggle action.
      // Note: ModSquad↔Dogs auto-linking mirrors backend logic in handleGroupToggle above;
      // if backend group-linking rules change, this must be updated to match.
      const currentGroups = user.groups || [];
      let updatedGroups: Group[];
      if (assigned) {
        // Removed group (and possibly ModSquad if Dogs was removed)
        const removedIds = new Set([group.id]);
        if (dogsGroup && modSquadGroup && group.id === dogsGroup.id) {
          removedIds.add(modSquadGroup.id);
        }
        updatedGroups = currentGroups.filter(g => !removedIds.has(g.id));
      } else {
        // Added group (and possibly Dogs if ModSquad was added)
        updatedGroups = [...currentGroups, group];
        if (modSquadGroup && dogsGroup && group.id === modSquadGroup.id && !currentGroups.some(g => g.id === dogsGroup.id)) {
          updatedGroups.push(dogsGroup);
        }
      }
      setGroupModalUser({ ...user, groups: updatedGroups });

      // Refresh the full user list in the background
      fetchUsers();
    } catch (err: unknown) {
      setGroupModalError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to update group');
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
    } catch (err: unknown) {
      setResetPasswordError(axios.isAxiosError(err) && err.response?.data?.error ? err.response.data.error : 'Failed to reset password');
    } finally {
      setResetPasswordLoading(false);
    }
  };

  // Admin user creation form state
  const [showCreate, setShowCreate] = React.useState(false);
  const [createData, setCreateData] = React.useState({
    username: '',
    first_name: '',
    last_name: '',
    email: '',
    password: '',
    is_admin: false,
    groupIds: [] as number[],
    send_setup_email: true  // Default to sending setup email
  });
  const [createLoading, setCreateLoading] = React.useState(false);
  const [createError, setCreateError] = React.useState<string | null>(null);
  const [createSuccess, setCreateSuccess] = React.useState<string | null>(null);
  const [createWarning, setCreateWarning] = React.useState<string | null>(null);
  
  // Form validation state
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});
  const [touchedFields, setTouchedFields] = React.useState<Set<string>>(new Set());
  const [showPassword, setShowPassword] = React.useState(false);


  // Fetch all groups for create form
  React.useEffect(() => {
    if (showCreate && allGroups.length === 0) {
      groupsApi.getAll().then(res => setAllGroups(res.data)).catch(() => {});
    }
  }, [showCreate, allGroups.length]);

  // Reset form when closing
  React.useEffect(() => {
    if (!showCreate) {
      setCreateData({
        username: '',
        first_name: '',
        last_name: '',
        email: '',
        password: '',
        is_admin: false,
        groupIds: [],
        send_setup_email: true
      });
      setFieldErrors({});
      setTouchedFields(new Set());
      setCreateError(null);
      setCreateSuccess(null);
      setCreateWarning(null);
      setShowPassword(false);
    }
  }, [showCreate]);

  // Validation functions
  const validateUsername = (value: string): string => {
    if (!value) return 'Username is required';
    if (value.length < 3) return 'Username must be at least 3 characters';
    if (value.length > 50) return 'Username must be less than 50 characters';
    if (!/^[a-zA-Z0-9_.-]+$/.test(value)) return 'Username can only contain letters, numbers, dots, hyphens, and underscores';
    return '';
  };

  const validateEmail = (value: string): string => {
    if (!value) return 'Email is required';
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return 'Please enter a valid email address';
    return '';
  };

  const validatePassword = (value: string, sendSetupEmail: boolean): string => {
    // Password is optional if sending setup email
    if (sendSetupEmail && !value) return '';
    if (!sendSetupEmail && !value) return 'Password is required when not sending setup email';
    if (value.length > 0 && value.length < 8) return 'Password must be at least 8 characters';
    if (value.length > 72) return 'Password must be less than 72 characters';
    return '';
  };

  const handleCreateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setCreateData(d => ({ ...d, [name]: type === 'checkbox' ? checked : value }));
    
    // Validate on change if field has been touched
    if (touchedFields.has(name)) {
      let error = '';
      if (name === 'username') error = validateUsername(value);
      else if (name === 'email') error = validateEmail(value);
      else if (name === 'password') error = validatePassword(value, createData.send_setup_email);
      
      setFieldErrors(prev => ({ ...prev, [name]: error }));
    }
  };

  const handleBlur = (field: string) => {
    setTouchedFields(prev => new Set(prev).add(field));
    
    let error = '';
    if (field === 'username') error = validateUsername(createData.username);
    else if (field === 'email') error = validateEmail(createData.email);
    else if (field === 'password') error = validatePassword(createData.password, createData.send_setup_email);
    
    setFieldErrors(prev => ({ ...prev, [field]: error }));
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
    
    // Validate all fields
    const errors: Record<string, string> = {
      username: validateUsername(createData.username),
      email: validateEmail(createData.email),
      password: validatePassword(createData.password, createData.send_setup_email),
    };
    
    // GroupAdmins must select at least one group
    if (!isAdmin && createData.groupIds.length === 0) {
      errors.groups = 'You must assign the user to at least one of your groups';
    }
    
    // Mark all fields as touched
    setTouchedFields(new Set(['username', 'email', 'password']));
    setFieldErrors(errors);
    
    // Check if there are any errors
    const hasErrors = Object.values(errors).some(error => error !== '');
    if (hasErrors) {
      setCreateError('Please fix the errors above before submitting');
      return;
    }
    
    setCreateLoading(true);
    setCreateError(null);
    setCreateSuccess(null);
    setCreateWarning(null);
    try {
      let response;
      
      if (isAdmin) {
        // Site admins use the admin endpoint with full permissions
        response = await usersApi.create({
          username: createData.username,
          first_name: createData.first_name || undefined,
          last_name: createData.last_name || undefined,
          email: createData.email,
          password: createData.password || undefined,
          is_admin: createData.is_admin,
          group_ids: createData.groupIds,
          send_setup_email: createData.send_setup_email,
        });
      } else {
        // GroupAdmins use the group admin endpoint (cannot set is_admin, must provide groups)
        response = await groupAdminApi.createUser({
          username: createData.username,
          first_name: createData.first_name || undefined,
          last_name: createData.last_name || undefined,
          email: createData.email,
          password: createData.password || undefined,
          send_setup_email: createData.send_setup_email,
          group_ids: createData.groupIds, // Required for group admins
        });
      }
      
      // Handle discriminated union response:
      // - User object directly (when password provided)
      // - { user: User, message?: string, warning?: string } (when setup email used)
      const responseData = response.data;
      
      // Type guard: check if response has 'user' property (wrapped format)
      if ('user' in responseData) {
        // Wrapped format from setup email flow
        const { warning, message } = responseData;
        
        if (warning) {
          // User created but email failed - show as warning (not error)
          setCreateWarning(warning);
          setCreateSuccess('User created successfully.');
        } else {
          // Success with email sent
          setCreateSuccess(message || 'User created successfully. Password setup email sent.');
        }
      } else {
        // Direct User object from password flow
        setCreateSuccess('User created successfully!');
      }
      
      setTimeout(() => {
        setShowCreate(false);
        fetchUsers();
      }, 3500); // Longer timeout for messages to be read
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setCreateError(error.response?.data?.error || 'Failed to create user');
    } finally {
      setCreateLoading(false);
    }
  };

  return (
    <div className="users-page">
      <h1>{canManageUsers ? 'Manage Users' : 'Team Members'}</h1>
      {canManageUsers && (
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
      )}

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

            {canManageUsers && (
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
            )}

            {canManageUsers && (
              <>
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
              </>
            )}
          </div>
          
          <div className="filter-summary">
            Showing {filteredUsers.length} of {users.length} users
            {searchQuery && <span> matching "{searchQuery}"</span>}
          </div>
        </div>
      )}

      {canManageUsers && showCreate && !showDeleted && (
        <form className="users-create-form" onSubmit={handleCreateSubmit}>
          <div className="create-form-header">
            <h2>Add New User</h2>
            <p className="form-description">Create a new user account with optional group assignments and admin privileges.</p>
          </div>

          <div className="form-grid">
            {/* Username Field */}
            <div className="form-field">
              <label htmlFor="create-username" className="form-label">
                Username <span className="required">*</span>
              </label>
              <input
                id="create-username"
                name="username"
                type="text"
                value={createData.username}
                onChange={handleCreateChange}
                onBlur={() => handleBlur('username')}
                className={`form-input ${fieldErrors.username && touchedFields.has('username') ? 'input-error' : ''}`}
                placeholder="e.g. john_doe"
                autoComplete="off"
                aria-invalid={!!fieldErrors.username && touchedFields.has('username')}
                aria-describedby={fieldErrors.username && touchedFields.has('username') ? 'username-error' : 'username-hint'}
              />
              {!fieldErrors.username && (
                <span id="username-hint" className="field-hint">
                  3-50 characters, letters, numbers, dots, hyphens, and underscores only
                </span>
              )}
              {fieldErrors.username && touchedFields.has('username') && (
                <span id="username-error" className="field-error" role="alert">
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM7 4h2v5H7V4zm0 6h2v2H7v-2z" fill="currentColor"/>
                  </svg>
                  {fieldErrors.username}
                </span>
              )}
            </div>

            {/* First Name Field */}
            <div className="form-field">
              <label htmlFor="create-first-name" className="form-label">
                First Name
              </label>
              <input
                id="create-first-name"
                name="first_name"
                type="text"
                value={createData.first_name}
                onChange={handleCreateChange}
                className="form-input"
                placeholder="e.g. Jane"
                autoComplete="off"
              />
            </div>

            {/* Last Name Field */}
            <div className="form-field">
              <label htmlFor="create-last-name" className="form-label">
                Last Name
              </label>
              <input
                id="create-last-name"
                name="last_name"
                type="text"
                value={createData.last_name}
                onChange={handleCreateChange}
                className="form-input"
                placeholder="e.g. Doe"
                autoComplete="off"
              />
            </div>

            {/* Email Field */}
            <div className="form-field">
              <label htmlFor="create-email" className="form-label">
                Email Address <span className="required">*</span>
              </label>
              <input
                id="create-email"
                name="email"
                type="email"
                value={createData.email}
                onChange={handleCreateChange}
                onBlur={() => handleBlur('email')}
                className={`form-input ${fieldErrors.email && touchedFields.has('email') ? 'input-error' : ''}`}
                placeholder="user@example.com"
                autoComplete="off"
                aria-invalid={!!fieldErrors.email && touchedFields.has('email')}
                aria-describedby={fieldErrors.email && touchedFields.has('email') ? 'email-error' : 'email-hint'}
              />
              {!fieldErrors.email && (
                <span id="email-hint" className="field-hint">
                  {createData.send_setup_email 
                    ? 'User will receive a password setup email at this address'
                    : 'User will receive login credentials at this email'}
                </span>
              )}
              {fieldErrors.email && touchedFields.has('email') && (
                <span id="email-error" className="field-error" role="alert">
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM7 4h2v5H7V4zm0 6h2v2H7v-2z" fill="currentColor"/>
                  </svg>
                  {fieldErrors.email}
                </span>
              )}
            </div>

            {/* Send Setup Email Toggle */}
            <div className="form-field checkbox-field" style={{ marginBottom: '1.5rem', padding: '1rem', backgroundColor: 'var(--surface, #f8fafc)', borderRadius: '0.5rem', border: '1px solid var(--border, #e5e7eb)' }}>
              <label className="checkbox-label">
                <input
                  name="send_setup_email"
                  type="checkbox"
                  checked={createData.send_setup_email}
                  onChange={handleCreateChange}
                  className="checkbox-input"
                />
                <span className="checkbox-box">
                  <svg className="checkbox-icon" width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M10 3L4.5 8.5L2 6" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                </span>
                <span className="checkbox-text">
                  <strong>Send password setup email</strong>
                  <span className="checkbox-description">User will receive an email to set their own password (recommended)</span>
                </span>
              </label>
            </div>

            {/* Password Field - Only show if not sending setup email */}
            {!createData.send_setup_email && (
              <div className="form-field">
                <label htmlFor="create-password" className="form-label">
                  Password <span className="required">*</span>
                </label>
                <div className="password-input-wrapper">
                  <input
                    id="create-password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    value={createData.password}
                    onChange={handleCreateChange}
                    onBlur={() => handleBlur('password')}
                    className={`form-input ${fieldErrors.password && touchedFields.has('password') ? 'input-error' : ''}`}
                    placeholder="Minimum 8 characters"
                    autoComplete="new-password"
                    aria-invalid={!!fieldErrors.password && touchedFields.has('password')}
                    aria-describedby={fieldErrors.password && touchedFields.has('password') ? 'password-error' : 'password-hint'}
                  />
                  <button
                    type="button"
                    className="password-toggle"
                    onClick={() => setShowPassword(!showPassword)}
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                  >
                    {showPassword ? (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24M1 1l22 22"/>
                      </svg>
                    ) : (
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                        <circle cx="12" cy="12" r="3"/>
                      </svg>
                    )}
                  </button>
                </div>
                {!fieldErrors.password && createData.password && (
                  <div className="password-strength">
                    <div className="strength-bar">
                      <div
                        className={`strength-fill strength-${getPasswordStrength(createData.password).strength}`}
                        style={{ width: getPasswordStrength(createData.password).strength === 'weak' ? '33%' : getPasswordStrength(createData.password).strength === 'medium' ? '66%' : '100%' }}
                      />
                    </div>
                    <span className="strength-label" style={{ color: getPasswordStrength(createData.password).color }}>
                      {getPasswordStrength(createData.password).label}
                    </span>
                  </div>
                )}
                {!fieldErrors.password && !createData.password && (
                  <span id="password-hint" className="field-hint">
                    At least 8 characters. Use a mix of letters, numbers, and symbols for better security.
                  </span>
                )}
                {fieldErrors.password && touchedFields.has('password') && (
                  <span id="password-error" className="field-error" role="alert">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM7 4h2v5H7V4zm0 6h2v2H7v-2z" fill="currentColor"/>
                    </svg>
                    {fieldErrors.password}
                  </span>
                )}
              </div>
            )}
          </div>

          {/* Admin Checkbox - Only visible to site admins */}
          {isAdmin && (
            <div className="form-field checkbox-field">
              <label className="checkbox-label">
                <input
                  name="is_admin"
                  type="checkbox"
                  checked={createData.is_admin}
                  onChange={handleCreateChange}
                  className="checkbox-input"
                />
                <span className="checkbox-box">
                  <svg className="checkbox-icon" width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M10 3L4.5 8.5L2 6" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                </span>
                <span className="checkbox-text">
                  <strong>Grant admin privileges</strong>
                  <span className="checkbox-description">Admins can manage users, groups, and all animals</span>
                </span>
              </label>
            </div>
          )}

          {/* Group Assignment */}
          <div className="form-field">
            <label className="form-label">
              Assign to Groups {!isAdmin && <span className="required">*</span>}
            </label>
            <p className="field-hint" style={{ marginTop: '-0.25rem', marginBottom: '0.75rem' }}>
              {isAdmin 
                ? "Select which groups this user can access. Leave unselected for no group access."
                : "Select which of your groups this user can access. You must select at least one group."}
            </p>
            <div className="group-checkboxes">
              {allGroups.length === 0 ? (
                <div className="no-groups-message">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="12" y1="8" x2="12" y2="12"/>
                    <line x1="12" y1="16" x2="12.01" y2="16"/>
                  </svg>
                  No groups available
                </div>
              ) : (
                allGroups.map(group => (
                  <label key={group.id} className="group-checkbox-label">
                    <input
                      type="checkbox"
                      checked={createData.groupIds.includes(group.id)}
                      onChange={() => handleCreateGroupToggle(group.id)}
                      className="checkbox-input"
                    />
                    <span className="checkbox-box">
                      <svg className="checkbox-icon" width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M10 3L4.5 8.5L2 6" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                      </svg>
                    </span>
                    <span className="group-name">{group.name}</span>
                    {group.description && <span className="group-description">{group.description}</span>}
                  </label>
                ))
              )}
            </div>
          </div>

          {/* Error, Warning, and Success Messages */}
          {createError && (
            <div className="form-alert alert-error" role="alert">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M10 0a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm-1 5h2v6H9V5zm0 8h2v2H9v-2z" fill="currentColor"/>
              </svg>
              <span>{createError}</span>
            </div>
          )}
          {createWarning && (
            <div className="form-alert alert-warning" role="alert">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M10 0a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm-1 5h2v6H9V5zm0 8h2v2H9v-2z" fill="currentColor"/>
              </svg>
              <span>{createWarning}</span>
            </div>
          )}
          {createSuccess && (
            <div className="form-alert alert-success" role="alert">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M10 0a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm-2 14l-4-4 1.41-1.41L8 11.17l6.59-6.59L16 6l-8 8z" fill="currentColor"/>
              </svg>
              <span>{createSuccess}</span>
            </div>
          )}

          {/* Form Actions */}
          <div className="form-actions">
            <button
              type="button"
              className="btn-secondary"
              onClick={() => setShowCreate(false)}
              disabled={createLoading}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn-primary"
              disabled={createLoading}
            >
              {createLoading ? (
                <>
                  <svg className="spinner" width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path opacity="0.2" d="M10 0a10 10 0 1 0 0 20 10 10 0 0 0 0-20z" fill="currentColor"/>
                    <path d="M10 0a10 10 0 0 1 10 10h-2A8 8 0 0 0 10 2V0z" fill="currentColor"/>
                  </svg>
                  Creating User...
                </>
              ) : (
                <>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                    <circle cx="8.5" cy="7" r="4"/>
                    <line x1="20" y1="8" x2="20" y2="14"/>
                    <line x1="23" y1="11" x2="17" y2="11"/>
                  </svg>
                  Create User
                </>
              )}
            </button>
          </div>
        </form>
      )}
      {loading ? (
        <div className="users-loading">Loading users…</div>
      ) : error ? (
        <div className="users-error">{error}</div>
      ) : (
        <>
          {/* Unified card-based user list */}
          <div className="users-grid">
            {filteredUsers.map(user => {
              const stats = statistics[user.id];
              const showDetails = expandedDetails.has(user.id);
              const toggleDetails = () => {
                setExpandedDetails(prev => {
                  const next = new Set(prev);
                  if (next.has(user.id)) {
                    next.delete(user.id);
                  } else {
                    next.add(user.id);
                  }
                  return next;
                });
              };
              return (
                <div key={user.id} className={`user-card-new ${user.deleted_at ? 'deleted' : ''}`}>
                  {/* Header with user info and badges */}
                  <div className="user-card-header-new">
                    <div className="user-avatar">
                      {user.username.charAt(0).toUpperCase()}
                    </div>
                    <div className="user-info">
                      <div className="user-name-row">
                        <Link 
                          to={`/users/${user.id}/profile`}
                          className="user-name-link"
                        >
                          {user.username}
                        </Link>
                        {user.is_admin && <span className="badge badge-admin">Admin</span>}
                        {user.deleted_at && <span className="badge badge-deleted">Deleted</span>}
                      </div>
                      <div className="user-email">{user.email}</div>
                    </div>
                  </div>

                  {/* Groups - simpler display */}
                  <div className="user-groups-section">
                    {user.groups && user.groups.length > 0 ? (
                      <div className="group-list-simple">
                        {user.groups.map(g => {
                          const isGroupAdmin = isUserGroupAdmin(user.id, g.id);
                          const canManageThisGroup = isAdmin || isCurrentUserGroupAdminOf(g.id);
                          return (
                            <div key={g.id} className="group-item-simple">
                              <Link to={`/groups/${g.id}`} className="group-name">
                                {g.name}
                              </Link>
                              {canManageThisGroup && !user.is_admin && (
                                <button
                                  onClick={() => handleToggleGroupAdmin(user.id, g.id, isGroupAdmin)}
                                  className={`admin-toggle ${isGroupAdmin ? 'is-admin' : ''}`}
                                  disabled={updatingGroupAdmin?.userId === user.id && updatingGroupAdmin?.groupId === g.id}
                                  title={isGroupAdmin ? 'Remove group admin privileges' : 'Grant group admin privileges'}
                                >
                                  {updatingGroupAdmin?.userId === user.id && updatingGroupAdmin?.groupId === g.id 
                                    ? '...' 
                                    : (isGroupAdmin ? 'Group Admin ✓' : 'Make Group Admin')}
                                </button>
                              )}
                              {!canManageThisGroup && isGroupAdmin && (
                                <span className="admin-label">Group Admin</span>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <span className="no-groups">No groups assigned</span>
                    )}
                  </div>

                  {/* Activity stats - collapsed by default for admins */}
                  {canManageUsers && stats && showDetails && (
                    <div className="user-stats-row">
                      <div className="stat-item" title={`${stats.comment_count} comments`}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                        </svg>
                        <span>{stats.comment_count}</span>
                      </div>
                      <div className="stat-item" title={`${stats.animals_interacted_with} animals`}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <circle cx="12" cy="12" r="3"></circle>
                          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2"></path>
                        </svg>
                        <span>{stats.animals_interacted_with}</span>
                      </div>
                      <div className="stat-item last-seen">
                        {stats.last_active ? (
                          <span title={new Date(stats.last_active).toLocaleString()}>
                            {formatRelativeTime(stats.last_active)}
                          </span>
                        ) : (
                          <span className="inactive">Never active</span>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Actions */}
                  <div className="user-card-actions-new">
                    {canManageUsers ? (
                      showDeleted ? (
                        <button
                          className="action-btn primary full-width"
                          onClick={() => handleRestore(user)}
                        >
                          Restore User
                        </button>
                      ) : (
                        <>
                          {isAdmin && (
                            <button
                              className="action-btn secondary"
                              onClick={() => openGroupModal(user)}
                              disabled={user.deleted_at}
                            >
                              Manage Groups
                            </button>
                          )}
                          <button
                            className="action-btn secondary"
                            onClick={() => openPasswordResetModal(user)}
                            disabled={user.deleted_at}
                          >
                            Reset Password
                          </button>
                          {isAdmin && (
                            <>
                              <button
                                className="action-btn secondary"
                                onClick={() => handlePromoteDemote(user)}
                                disabled={user.deleted_at}
                              >
                                {user.is_admin ? 'Demote Admin' : 'Make Admin'}
                              </button>
                              <button
                                className="action-btn danger"
                                onClick={() => handleDelete(user)}
                                disabled={user.deleted_at}
                              >
                                Delete
                              </button>
                            </>
                          )}
                          {stats && (
                            <button
                              className="action-btn secondary"
                              onClick={toggleDetails}
                            >
                              {showDetails ? 'Hide Details' : 'Show Details'}
                            </button>
                          )}
                        </>
                      )
                    ) : (
                      <Link 
                        to={`/users/${user.id}/profile`}
                        className="action-btn primary full-width"
                      >
                        View Profile
                      </Link>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
          
          {filteredUsers.length === 0 && (
            <div className="users-empty">
              <p>No users found matching your criteria.</p>
            </div>
          )}
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
