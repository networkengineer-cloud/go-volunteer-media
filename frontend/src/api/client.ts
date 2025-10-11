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
  create: (name: string, description: string) =>
    api.post<Group>('/admin/groups', { name, description }),
  update: (id: number, name: string, description: string) =>
    api.put<Group>('/admin/groups/' + id, { name, description }),
  delete: (id: number) => api.delete('/admin/groups/' + id),
};

// Animals API
export const animalsApi = {
  getAll: (groupId: number) => api.get<Animal[]>('/groups/' + groupId + '/animals'),
  getById: (groupId: number, id: number) =>
    api.get<Animal>('/groups/' + groupId + '/animals/' + id),
  create: (groupId: number, data: Partial<Animal>) =>
    api.post<Animal>('/groups/' + groupId + '/animals', data),
  update: (groupId: number, id: number, data: Partial<Animal>) =>
    api.put<Animal>('/groups/' + groupId + '/animals/' + id, data),
  delete: (groupId: number, id: number) =>
    api.delete('/groups/' + groupId + '/animals/' + id),
};

// Updates API
export const updatesApi = {
  getAll: (groupId: number) => api.get<Update[]>('/groups/' + groupId + '/updates'),
  create: (groupId: number, title: string, content: string, image_url?: string) =>
    api.post<Update>('/groups/' + groupId + '/updates', { title, content, image_url }),
};

export default api;
