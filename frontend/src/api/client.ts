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
  groups?: Group[];
}

export interface Group {
  id: number;
  name: string;
  description: string;
  image_url: string;
  hero_image_url: string;
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
}

export interface Update {
  id: number;
  group_id: number;
  user_id: number;
  title: string;
  content: string;
  image_url: string;
  created_at: string;
  user?: User;
}

export interface Announcement {
  id: number;
  user_id: number;
  title: string;
  content: string;
  send_email: boolean;
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

export interface CommentTag {
  id: number;
  name: string;
  color: string;
  is_system: boolean;
  created_at: string;
}

// Auth API
export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string; user: User }>('/login', { username, password }),
  
  register: (username: string, email: string, password: string) =>
    api.post<{ token: string; user: User }>('/register', { username, email, password }),
  
  getCurrentUser: () => api.get<User>('/me'),
};

// Groups API
export const groupsApi = {
  getAll: () => api.get<Group[]>('/groups'),
  getById: (id: number) => api.get<Group>('/groups/' + id),
  create: (name: string, description: string, image_url?: string, hero_image_url?: string) =>
    api.post<Group>('/admin/groups', { name, description, image_url, hero_image_url }),
  update: (id: number, name: string, description: string, image_url?: string, hero_image_url?: string) =>
    api.put<Group>('/admin/groups/' + id, { name, description, image_url, hero_image_url }),
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
    const params: any = {};
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
    const params: any = {};
    if (status !== undefined) params.status = status;
    if (groupId !== undefined) params.group_id = groupId;
    if (name) params.name = name;
    return api.get<Animal[]>('/admin/animals', { params });
  },
  bulkUpdate: (animalIds: number[], groupId?: number, status?: string) => {
    const data: any = { animal_ids: animalIds };
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
  exportCommentsCSV: (groupId?: number) => {
    const params = groupId ? { group_id: groupId } : {};
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

// Updates API
export const updatesApi = {
  getAll: (groupId: number) => api.get<Update[]>('/groups/' + groupId + '/updates'),
  create: (groupId: number, title: string, content: string, image_url?: string) =>
    api.post<Update>('/groups/' + groupId + '/updates', { title, content, image_url }),
};

// Announcements API
export const announcementsApi = {
  getAll: () => api.get<Announcement[]>('/announcements'),
  create: (title: string, content: string, send_email: boolean) =>
    api.post<Announcement>('/admin/announcements', { title, content, send_email }),
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

export default api;
