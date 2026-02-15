// Users API (admin)
// TODO: Implement proper pagination in admin UI; limit=100 is a temporary workaround
export const usersApi = {
  getAll: () => api.get<PaginatedResponse<User>>('/admin/users?limit=100'),
  create: (data: { username: string; email: string; password?: string; is_admin?: boolean; group_ids?: number[]; send_setup_email?: boolean }) =>
    api.post<CreateUserResponse>('/admin/users', data),
  promote: (userId: number) => api.post(`/admin/users/${userId}/promote`),
  demote: (userId: number) => api.post(`/admin/users/${userId}/demote`),
  delete: (userId: number) => api.delete(`/admin/users/${userId}`),
  assignGroup: (userId: number, groupId: number) => api.post(`/admin/users/${userId}/groups/${groupId}`),
  removeGroup: (userId: number, groupId: number) => api.delete(`/admin/users/${userId}/groups/${groupId}`),
  getDeleted: () => api.get<User[]>('/admin/users/deleted'),
  restore: (userId: number) => api.post(`/admin/users/${userId}/restore`),
  resetPassword: (userId: number, newPassword: string) => api.post(`/users/${userId}/reset-password`, { new_password: newPassword }),
};

// Group Admin API (accessible by site admins and group admins)
export const groupAdminApi = {
  // Get members of a group with their admin status
  getMembers: (groupId: number) => api.get<GroupMember[]>(`/groups/${groupId}/members`),
  // Promote a user to group admin (site admins and group admins can do this for their groups)
  promoteToGroupAdmin: (groupId: number, userId: number) => api.post(`/groups/${groupId}/admins/${userId}`),
  // Demote a user from group admin (site admins and group admins can do this for their groups)
  demoteFromGroupAdmin: (groupId: number, userId: number) => api.delete(`/groups/${groupId}/admins/${userId}`),
  // Add a user to a group (site admins and group admins can do this for their groups)
  addMemberToGroup: (groupId: number, userId: number) => api.post(`/groups/${groupId}/members/${userId}`),
  // Remove a user from a group (site admins and group admins can do this for their groups)
  removeMemberFromGroup: (groupId: number, userId: number) => api.delete(`/groups/${groupId}/members/${userId}`),
  // Create a new user (group admins can create users and assign them to groups they admin)
  createUser: (data: { username: string; email: string; password?: string; send_setup_email?: boolean; group_ids: number[] }) =>
    api.post<CreateUserResponse>('/users', data),
};
import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

// Add token to requests if available
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = 'Bearer ' + token;
  }
  return config;
});

// Handle 401 responses (expired/invalid token)
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear invalid token
      localStorage.removeItem('token');
      
      // Redirect to login if not already there
      if (window.location.pathname !== '/login') {
        // Store the current path to redirect back after login
        const currentPath = window.location.pathname + window.location.search;
        if (currentPath !== '/' && currentPath !== '/login') {
          sessionStorage.setItem('redirectAfterLogin', currentPath);
        }
        window.location.href = '/login?expired=true';
      }
    }
    return Promise.reject(error);
  }
);

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
}

export interface User {
  id: number;
  username: string;
  email: string;
  phone_number?: string;
  hide_email?: boolean;
  hide_phone_number?: boolean;
  is_admin: boolean;
  is_group_admin?: boolean; // True if user is group admin of at least one group
  default_group_id?: number;
  groups?: Group[];
  deleted_at?: string | null;
}

// User creation response - backend returns different formats depending on setup email
// Using discriminated union for type safety:
// - When send_setup_email=false: returns User directly
// - When send_setup_email=true: returns wrapped response with optional message/warning
export type CreateUserResponse = 
  | User  // Direct user object when password provided
  | { user: User; message?: string; warning?: string };  // Wrapped response with setup email

// GroupMember represents a user's membership in a group with admin status
export interface GroupMember {
  user_id: number;
  username: string;
  email: string;
  phone_number?: string;
  is_group_admin: boolean;
  is_site_admin: boolean;
}

export interface Group {
  id: number;
  name: string;
  description: string;
  image_url: string;
  hero_image_url: string;
  has_protocols: boolean;
  groupme_bot_id: string;
  groupme_enabled: boolean;
}

// GroupMembership represents the current user's membership status in a group
export interface GroupMembership {
  user_id: number;
  group_id: number;
  is_member: boolean;
  is_group_admin: boolean;
  is_site_admin: boolean;
}

export interface Protocol {
  id: number;
  group_id: number;
  title: string;
  content: string;
  image_url: string;
  order_index: number;
  created_at: string;
  updated_at: string;
}

export interface AnimalNameHistory {
  id: number;
  created_at: string;
  animal_id: number;
  old_name: string;
  new_name: string;
  changed_by: number;
}

export interface Animal {
  id: number;
  group_id: number;
  name: string;
  species: string;
  breed: string;
  age: number;
  description: string;
  image_url: string;
  status: string;
  arrival_date?: string;
  foster_start_date?: string;
  quarantine_start_date?: string;
  archived_date?: string;
  last_status_change?: string;
  return_count: number;
  is_returned: boolean;
  protocol_document_url?: string;
  protocol_document_name?: string;
  protocol_document_type?: string;
  protocol_document_size?: number;
  protocol_document_user_id?: number;
  tags?: AnimalTag[];
  name_history?: AnimalNameHistory[];
}

export interface Update {
  id: number;
  group_id: number;
  user_id: number;
  title: string;
  content: string;
  image_url: string;
  send_groupme: boolean;
  created_at: string;
  user?: User;
}

export interface Announcement {
  id: number;
  user_id: number;
  title: string;
  content: string;
  send_email: boolean;
  send_groupme: boolean;
  created_at: string;
  user?: User;
}

export interface AnimalImage {
  id: number;
  animal_id: number;
  user_id: number;
  image_url: string;
  caption: string;
  is_profile_picture: boolean;
  width: number;
  height: number;
  file_size: number;
  created_at: string;
  deleted_at?: string | null;
  user?: User;
}

export interface AnimalComment {
  id: number;
  animal_id: number;
  user_id: number;
  content: string;
  image_url: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  metadata?: SessionMetadata;
  tags?: CommentTag[];
  user?: User;
}

export interface SessionMetadata {
  session_goal?: string;
  session_outcome?: string;
  behavior_notes?: string;
  medical_notes?: string;
  session_rating?: number; // 1-5 (Poor, Fair, Okay, Good, Great)
  other_notes?: string;
}

export interface CommentHistory {
  id: number;
  created_at: string;
  comment_id: number;
  content: string;
  image_url: string;
  metadata?: SessionMetadata;
  edited_by: number;
  user?: User;
}

export interface PaginatedCommentsResponse {
  comments: AnimalComment[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
}

export interface CommentWithAnimal extends AnimalComment {
  animal: Animal;
}

export interface CommentTag {
  id: number;
  name: string;
  color: string;
  is_system: boolean;
  created_at: string;
}

export interface AnimalTag {
  id: number;
  name: string;
  category: string; // 'behavior' or 'walker_status'
  color: string;
  icon: string; // Unicode emoji
  created_at: string;
}

export interface DuplicateNameInfo {
  name: string;
  count: number;
  animals: Animal[];
  has_duplicates: boolean;
}

export interface ActivityItem {
  id: number;
  type: 'comment' | 'announcement';
  created_at: string;
  user_id: number;
  user?: User;
  content: string;
  title?: string;
  image_url?: string;
  animal_id?: number;
  animal?: Animal;
  tags?: CommentTag[];
  metadata?: SessionMetadata;
}

export interface ActivityFeedResponse {
  items: ActivityItem[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
  summary?: {
    behavior_concerns_count: number;
    medical_concerns_count: number;
    poor_sessions_count: number;
  };
}

export interface GroupStatistics {
  group_id: number;
  user_count: number;
  animal_count: number;
  last_activity?: string;
}

export interface UserStatistics {
  user_id: number;
  comment_count: number;
  last_active?: string;
  animals_interacted_with: number;
}

export interface CommentTagStatistics {
  tag_id: number;
  usage_count: number;
  last_used?: string;
  most_tagged_animal_id?: number;
  most_tagged_animal_name?: string;
}

// User Profile interfaces
export interface UserProfileStatistics {
  total_comments: number;
  total_announcements: number;
  animals_interacted: number;
  most_active_group?: {
    group_id: number;
    group_name: string;
    comment_count: number;
  };
  last_active_date?: string;
}

export interface UserCommentActivity {
  id: number;
  animal_id: number;
  animal_name: string;
  group_id: number;
  group_name: string;
  content: string;
  image_url: string;
  created_at: string;
}

export interface UserAnnouncementActivity {
  id: number;
  group_id: number;
  group_name: string;
  content: string;
  created_at: string;
}

export interface AnimalInteraction {
  animal_id: number;
  animal_name: string;
  group_id: number;
  group_name: string;
  image_url: string;
  comment_count: number;
  last_comment_at: string;
}

export interface UserProfile {
  id: number;
  username: string;
  email?: string;  // Optional for limited profiles
  phone_number?: string;  // Optional for limited profiles
  is_admin?: boolean;  // Optional for limited/group admin profiles
  created_at?: string;  // Optional for limited profiles
  default_group_id?: number;
  groups?: Group[];  // Optional for limited profiles
  statistics?: UserProfileStatistics;  // Optional for limited/group admin profiles
  recent_comments?: UserCommentActivity[];  // Optional for limited/group admin profiles
  recent_announcements?: UserAnnouncementActivity[];  // Optional for limited/group admin profiles
  animals_interacted_with?: AnimalInteraction[];  // Optional for limited/group admin profiles
}


// Auth API
export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string; user: User }>('/login', { username, password }),
  
  register: (username: string, email: string, password: string) =>
    api.post<{ token: string; user: User }>('/register', { username, email, password }),
  
  getCurrentUser: () => api.get<User>('/me'),
  
  updateCurrentUserProfile: (profile: {
    email: string;
    phone_number?: string;
    hide_email?: boolean;
    hide_phone_number?: boolean;
  }) =>
    api.put<{
      message: string;
      id: number;
      email: string;
      phone_number?: string;
      hide_email?: boolean;
      hide_phone_number?: boolean;
    }>('/me/profile', profile),
  
  setDefaultGroup: (groupId: number) => api.put('/default-group', { group_id: groupId }),
  
  getDefaultGroup: () => api.get<Group>('/default-group'),
  
  getEmailPreferences: () => api.get<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>('/email-preferences'),
  
  updateEmailPreferences: (emailNotificationsEnabled: boolean, showLengthOfStay: boolean) =>
    api.put<{ message: string; email_notifications_enabled: boolean; show_length_of_stay: boolean }>('/email-preferences', {
      email_notifications_enabled: emailNotificationsEnabled,
      show_length_of_stay: showLengthOfStay,
    }),
};

// Groups API
export const groupsApi = {
  getAll: () => api.get<Group[]>('/groups'),
  getById: (id: number) => api.get<Group>('/groups/' + id),
  getMembership: (id: number) => api.get<GroupMembership>('/groups/' + id + '/membership'),
  getLatestComments: (id: number, limit?: number) => {
    const params = limit ? { limit } : {};
    return api.get<CommentWithAnimal[]>('/groups/' + id + '/latest-comments', { params });
  },
  getActivityFeed: (id: number, options?: { 
    limit?: number; 
    offset?: number; 
    type?: 'all' | 'comments' | 'announcements';
    animal?: number;
    tags?: string;
    rating?: string;
    from?: string;
    to?: string;
  }) => {
    const params: Record<string, unknown> = {};
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    if (options?.type && options.type !== 'all') params.type = options.type;
    if (options?.animal) params.animal = options.animal;
    if (options?.tags) params.tags = options.tags;
    if (options?.rating) params.rating = options.rating;
    if (options?.from) params.from = options.from;
    if (options?.to) params.to = options.to;
    return api.get<ActivityFeedResponse>('/groups/' + id + '/activity-feed', { params });
  },
  create: (name: string, description: string, image_url?: string, hero_image_url?: string, has_protocols?: boolean, groupme_bot_id?: string, groupme_enabled?: boolean) =>
    api.post<Group>('/admin/groups', { name, description, image_url, hero_image_url, has_protocols, groupme_bot_id, groupme_enabled }),
  update: (id: number, name: string, description: string, image_url?: string, hero_image_url?: string, has_protocols?: boolean, groupme_bot_id?: string, groupme_enabled?: boolean) =>
    api.put<Group>('/admin/groups/' + id, { name, description, image_url, hero_image_url, has_protocols, groupme_bot_id, groupme_enabled }),
  delete: (id: number) => api.delete('/admin/groups/' + id),
  uploadImage: (file: File) => {
    const formData = new FormData();
    formData.append('image', file);
    return api.post<{ url: string }>('/admin/groups/upload-image', formData);
  },
};

// Animals API
export const animalsApi = {
  getAll: (groupId: number, status?: string, name?: string) => {
    const params: Record<string, unknown> = {};
    if (status !== undefined) params.status = status;
    if (name) params.name = name;
    return api.get<Animal[]>('/groups/' + groupId + '/animals', { params });
  },
  getById: (groupId: number, id: number) =>
    api.get<Animal>('/groups/' + groupId + '/animals/' + id),
  checkDuplicates: (groupId: number, name: string) =>
    api.get<DuplicateNameInfo>('/groups/' + groupId + '/animals/check-duplicates', { params: { name } }),
  create: (groupId: number, data: Partial<Animal>) =>
    api.post<Animal>('/groups/' + groupId + '/animals', data),
  update: (groupId: number, id: number, data: Partial<Animal>) =>
    api.put<Animal>('/groups/' + groupId + '/animals/' + id, data),
  delete: (groupId: number, id: number) =>
    api.delete('/groups/' + groupId + '/animals/' + id),
  uploadImage: (file: File) => {
    const formData = new FormData();
    formData.append('image', file);
    return api.post<{ url: string }>('/animals/upload-image', formData);
  },
  // Image gallery API
  getImages: (groupId: number, animalId: number) =>
    api.get<AnimalImage[]>('/groups/' + groupId + '/animals/' + animalId + '/images'),
  uploadImageToGallery: (groupId: number, animalId: number, file: File, caption?: string) => {
    const formData = new FormData();
    formData.append('image', file);
    if (caption) formData.append('caption', caption);
    return api.post<AnimalImage>('/groups/' + groupId + '/animals/' + animalId + '/images', formData);
  },
  deleteImage: (groupId: number, animalId: number, imageId: number) =>
    api.delete('/groups/' + groupId + '/animals/' + animalId + '/images/' + imageId),
  getDeletedImages: (groupId: number) =>
    api.get<AnimalImage[]>('/admin/groups/' + groupId + '/deleted-images'),
  setProfilePicture: (groupId: number, animalId: number, imageId: number) =>
    api.put<AnimalImage>(`/groups/${groupId}/animals/${animalId}/images/${imageId}/set-profile`),
  // Protocol document API
  uploadProtocolDocument: (groupId: number, animalId: number, file: File) => {
    const formData = new FormData();
    formData.append('document', file);
    return api.post<{
      url: string;
      name: string;
      size: number;
      type: string;
      uploaded_by: number;
    }>(`/groups/${groupId}/animals/${animalId}/protocol-document`, formData);
  },
  deleteProtocolDocument: (groupId: number, animalId: number) =>
    api.delete(`/groups/${groupId}/animals/${animalId}/protocol-document`),
  getProtocolDocument: (uuid: string) =>
    api.get(`/documents/${uuid}`, { responseType: 'blob' }),
  // Admin and group admin bulk operations
  getAllForAdmin: (status?: string, groupId?: number, name?: string) => {
    const params: Record<string, unknown> = {};
    if (status !== undefined) params.status = status;
    if (groupId !== undefined) params.group_id = groupId;
    if (name) params.name = name;
    return api.get<Animal[]>('/bulk-animals', { params });
  },
  bulkUpdate: (animalIds: number[], groupId?: number, status?: string) => {
    const data: Record<string, unknown> = { animal_ids: animalIds };
    if (groupId !== undefined) data.group_id = groupId;
    if (status !== undefined) data.status = status;
    return api.post<{ message: string; count: number }>('/bulk-animals/bulk-update', data);
  },
  importCSV: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post<{ message: string; count: number; warnings?: string[] }>('/admin/animals/import-csv', formData);
  },
  exportCSV: (groupId?: number) => {
    const params = groupId ? { group_id: groupId } : {};
    return api.get('/admin/animals/export-csv', { 
      params,
      responseType: 'blob' 
    });
  },
  exportCommentsCSV: (groupId?: number, animalId?: number, tags?: string) => {
    const params: Record<string, unknown> = {};
    if (groupId) params.group_id = groupId;
    if (animalId) params.animal_id = animalId;
    if (tags) params.tags = tags;
    return api.get('/admin/animals/export-comments-csv', {
      params,
      responseType: 'blob'
    });
  },
  updateAnimal: (animalId: number, data: Partial<Animal>) => {
    return api.put<Animal>(`/admin/animals/${animalId}`, data);
  },
};

// Animal Comments API
export const animalCommentsApi = {
  getAll: (groupId: number, animalId: number, options?: {
    tagFilter?: string;
    limit?: number;
    offset?: number;
    order?: 'asc' | 'desc';
  }) => {
    const params: Record<string, string | number> = {};
    if (options?.tagFilter) params.tags = options.tagFilter;
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    if (options?.order) params.order = options.order;
    return api.get<PaginatedCommentsResponse>('/groups/' + groupId + '/animals/' + animalId + '/comments', { params });
  },
  create: (groupId: number, animalId: number, content: string, image_url?: string, tag_ids?: number[], metadata?: SessionMetadata) =>
    api.post<AnimalComment>('/groups/' + groupId + '/animals/' + animalId + '/comments', {
      content,
      image_url,
      tag_ids,
      metadata,
    }),
  update: (groupId: number, animalId: number, commentId: number, content: string, image_url?: string, tag_ids?: number[], metadata?: SessionMetadata) =>
    api.put<AnimalComment>('/groups/' + groupId + '/animals/' + animalId + '/comments/' + commentId, {
      content,
      image_url,
      tag_ids,
      metadata,
    }),
  delete: (groupId: number, animalId: number, commentId: number) =>
    api.delete('/groups/' + groupId + '/animals/' + animalId + '/comments/' + commentId),
  getDeleted: (groupId: number) =>
    api.get<AnimalComment[]>('/admin/groups/' + groupId + '/deleted-comments'),
  getHistory: (groupId: number, animalId: number, commentId: number) =>
    api.get<CommentHistory[]>('/groups/' + groupId + '/animals/' + animalId + '/comments/' + commentId + '/history'),
};

// Comment Tags API - Group-specific tags
export const commentTagsApi = {
  getAll: (groupId: number) => api.get<CommentTag[]>('/groups/' + groupId + '/comment-tags'),
  create: (groupId: number, name: string, color: string) =>
    api.post<CommentTag>('/groups/' + groupId + '/comment-tags', { name, color }),
  delete: (groupId: number, tagId: number) => api.delete('/groups/' + groupId + '/comment-tags/' + tagId),
};

// Animal Tags API - Group-specific tags
export const animalTagsApi = {
  getAll: (groupId: number) => api.get<AnimalTag[]>('/groups/' + groupId + '/animal-tags'),
  create: (groupId: number, data: { name: string; category: string; color: string }) =>
    api.post<AnimalTag>('/groups/' + groupId + '/animal-tags', data),
  update: (groupId: number, tagId: number, data: { name: string; category: string; color: string }) =>
    api.put<AnimalTag>('/groups/' + groupId + '/animal-tags/' + tagId, data),
  delete: (groupId: number, tagId: number) => api.delete('/groups/' + groupId + '/animal-tags/' + tagId),
  assignToAnimal: (groupId: number, animalId: number, tagIds: number[]) =>
    api.post<Animal>('/groups/' + groupId + '/animals/' + animalId + '/tags', { tag_ids: tagIds }),
};

// Protocols API
export const protocolsApi = {
  getAll: (groupId: number) => api.get<Protocol[]>('/groups/' + groupId + '/protocols'),
  getById: (groupId: number, protocolId: number) => api.get<Protocol>('/groups/' + groupId + '/protocols/' + protocolId),
  create: (groupId: number, data: { title: string; content: string; image_url?: string; order_index?: number }) =>
    api.post<Protocol>('/groups/' + groupId + '/protocols', data),
  update: (groupId: number, protocolId: number, data: { title: string; content: string; image_url?: string; order_index?: number }) =>
    api.put<Protocol>('/groups/' + groupId + '/protocols/' + protocolId, data),
  delete: (groupId: number, protocolId: number) => api.delete('/groups/' + groupId + '/protocols/' + protocolId),
  uploadImage: (groupId: number, file: File) => {
    const formData = new FormData();
    formData.append('image', file);
    return api.post<{ url: string }>('/groups/' + groupId + '/protocols/upload-image', formData);
  },
};

// Updates API
export const updatesApi = {
  getAll: (groupId: number) => api.get<Update[]>('/groups/' + groupId + '/updates'),
  create: (groupId: number, title: string, content: string, send_groupme: boolean, image_url?: string) =>
    api.post<Update>('/groups/' + groupId + '/updates', { title, content, image_url, send_groupme }),
};

// Announcements API
export const announcementsApi = {
  getAll: () => api.get<Announcement[]>('/announcements'),
  create: (title: string, content: string, send_email: boolean, send_groupme: boolean) =>
    api.post<Announcement>('/admin/announcements', { title, content, send_email, send_groupme }),
  delete: (id: number) => api.delete('/admin/announcements/' + id),
};

// Site Settings API
export const settingsApi = {
  getAll: () => api.get<Record<string, string>>('/settings'),
  update: (key: string, value: string) => api.put('/admin/settings/' + key, { value }),
  uploadHeroImage: (file: File) => {
    const formData = new FormData();
    formData.append('image', file);
    return api.post<{ url: string }>('/admin/settings/upload-hero-image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
};

// Statistics API
// TODO: Implement proper pagination in admin UI; limit=100 is a temporary workaround
export const statisticsApi = {
  getGroupStatistics: () => api.get<PaginatedResponse<GroupStatistics>>('/admin/statistics/groups?limit=100'),
  getUserStatistics: () => api.get<PaginatedResponse<UserStatistics>>('/admin/statistics/users?limit=100'),
  getCommentTagStatistics: (groupId?: number) => {
    const params = groupId ? `?group_id=${groupId}&limit=100` : '?limit=100';
    return api.get<PaginatedResponse<CommentTagStatistics>>(`/statistics/comment-tags${params}`);
  },
};

// User Profile API
export const userProfileApi = {
  getProfile: (userId: number) => api.get<UserProfile>(`/users/${userId}/profile`),
};

// Admin Dashboard interfaces
export interface AdminDashboardStats {
  total_users: number;
  total_groups: number;
  total_animals: number;
  total_comments: number;
  recent_users: RecentUser[];
  most_active_groups: ActiveGroupInfo[];
  animals_needing_attention: AnimalAlert[];
  system_health: SystemHealthInfo;
}

export interface RecentUser {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  created_at: string;
}

export interface ActiveGroupInfo {
  group_id: number;
  group_name: string;
  user_count: number;
  animal_count: number;
  comment_count: number;
  last_activity: string;
}

export interface AnimalAlert {
  animal_id: number;
  animal_name: string;
  group_id: number;
  group_name: string;
  image_url: string;
  alert_tags: string[];
  last_comment: string;
}

export interface SystemHealthInfo {
  active_users_last_24h: number;
  comments_last_24h: number;
  new_users_last_7_days: number;
  average_comments_per_day: number;
}

// Admin Dashboard API
export const adminDashboardApi = {
  getStats: () => api.get<AdminDashboardStats>('/admin/dashboard/stats'),
};

export default api;
