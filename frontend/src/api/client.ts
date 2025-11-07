// Users API (admin)
export const usersApi = {
  getAll: () => api.get<User[]>('/admin/users'),
  create: (data: { username: string; email: string; password: string; is_admin?: boolean; group_ids?: number[] }) =>
    api.post<User>('/admin/users', data),
  promote: (userId: number) => api.post(`/admin/users/${userId}/promote`),
  demote: (userId: number) => api.post(`/admin/users/${userId}/demote`),
  delete: (userId: number) => api.delete(`/admin/users/${userId}`),
  assignGroup: (userId: number, groupId: number) => api.post(`/admin/users/${userId}/groups/${groupId}`),
  removeGroup: (userId: number, groupId: number) => api.delete(`/admin/users/${userId}/groups/${groupId}`),
  getDeleted: () => api.get<User[]>('/admin/users/deleted'),
  restore: (userId: number) => api.post(`/admin/users/${userId}/restore`),
  resetPassword: (userId: number, newPassword: string) => api.post(`/admin/users/${userId}/reset-password`, { new_password: newPassword }),
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

export interface User {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  default_group_id?: number;
  groups?: Group[];
  deleted_at?: string | null;
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
  tags?: AnimalTag[];
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

export interface AnimalComment {
  id: number;
  animal_id: number;
  user_id: number;
  content: string;
  image_url: string;
  created_at: string;
  tags?: CommentTag[];
  user?: User;
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
  created_at: string;
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
}

export interface ActivityFeedResponse {
  items: ActivityItem[];
  total: number;
  limit: number;
  offset: number;
  hasMore: boolean;
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
  email: string;
  is_admin: boolean;
  created_at: string;
  default_group_id?: number;
  groups: Group[];
  statistics: UserProfileStatistics;
  recent_comments: UserCommentActivity[];
  recent_announcements: UserAnnouncementActivity[];
  animals_interacted_with: AnimalInteraction[];
}


// Auth API
export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string; user: User }>('/login', { username, password }),
  
  register: (username: string, email: string, password: string) =>
    api.post<{ token: string; user: User }>('/register', { username, email, password }),
  
  getCurrentUser: () => api.get<User>('/me'),
  
  setDefaultGroup: (groupId: number) => api.put('/default-group', { group_id: groupId }),
  
  getDefaultGroup: () => api.get<Group>('/default-group'),
  
  getEmailPreferences: () => api.get<{ email_notifications_enabled: boolean }>('/email-preferences'),
  
  updateEmailPreferences: (emailNotificationsEnabled: boolean) =>
    api.put<{ message: string; email_notifications_enabled: boolean }>('/email-preferences', {
      email_notifications_enabled: emailNotificationsEnabled,
    }),
};

// Groups API
export const groupsApi = {
  getAll: () => api.get<Group[]>('/groups'),
  getById: (id: number) => api.get<Group>('/groups/' + id),
  getLatestComments: (id: number, limit?: number) => {
    const params = limit ? { limit } : {};
    return api.get<CommentWithAnimal[]>('/groups/' + id + '/latest-comments', { params });
  },
  getActivityFeed: (id: number, options?: { limit?: number; offset?: number; type?: 'all' | 'comments' | 'announcements' }) => {
    const params: Record<string, unknown> = {};
    if (options?.limit) params.limit = options.limit;
    if (options?.offset) params.offset = options.offset;
    if (options?.type && options.type !== 'all') params.type = options.type;
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
  // Admin bulk operations
  getAllForAdmin: (status?: string, groupId?: number, name?: string) => {
    const params: Record<string, unknown> = {};
    if (status !== undefined) params.status = status;
    if (groupId !== undefined) params.group_id = groupId;
    if (name) params.name = name;
    return api.get<Animal[]>('/admin/animals', { params });
  },
  bulkUpdate: (animalIds: number[], groupId?: number, status?: string) => {
    const data: Record<string, unknown> = { animal_ids: animalIds };
    if (groupId !== undefined) data.group_id = groupId;
    if (status !== undefined) data.status = status;
    return api.post<{ message: string; count: number }>('/admin/animals/bulk-update', data);
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
  getAll: (groupId: number, animalId: number, tagFilter?: string) => {
    const params = tagFilter ? { tags: tagFilter } : {};
    return api.get<AnimalComment[]>('/groups/' + groupId + '/animals/' + animalId + '/comments', { params });
  },
  create: (groupId: number, animalId: number, content: string, image_url?: string, tag_ids?: number[]) =>
    api.post<AnimalComment>('/groups/' + groupId + '/animals/' + animalId + '/comments', {
      content,
      image_url,
      tag_ids,
    }),
};

// Comment Tags API
export const commentTagsApi = {
  getAll: () => api.get<CommentTag[]>('/comment-tags'),
  create: (name: string, color: string) =>
    api.post<CommentTag>('/admin/comment-tags', { name, color }),
  delete: (tagId: number) => api.delete('/admin/comment-tags/' + tagId),
};

// Animal Tags API
export const animalTagsApi = {
  getAll: () => api.get<AnimalTag[]>('/animal-tags'),
  create: (data: { name: string; category: string; color: string }) =>
    api.post<AnimalTag>('/admin/animal-tags', data),
  update: (tagId: number, data: { name: string; category: string; color: string }) =>
    api.put<AnimalTag>('/admin/animal-tags/' + tagId, data),
  delete: (tagId: number) => api.delete('/admin/animal-tags/' + tagId),
  assignToAnimal: (animalId: number, tagIds: number[]) =>
    api.post<Animal>('/admin/animals/' + animalId + '/tags', { tag_ids: tagIds }),
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

// Statistics API (admin only)
export const statisticsApi = {
  getGroupStatistics: () => api.get<GroupStatistics[]>('/admin/statistics/groups'),
  getUserStatistics: () => api.get<UserStatistics[]>('/admin/statistics/users'),
  getCommentTagStatistics: () => api.get<CommentTagStatistics[]>('/admin/statistics/comment-tags'),
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
