import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider } from './AuthContext';
import { useAuth } from '../hooks/useAuth';
import { authApi, User } from '../api/client';
import { AxiosResponse } from 'axios';

// Mock the API client
vi.mock('../api/client', () => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    getCurrentUser: vi.fn(),
  },
}));

describe('AuthContext', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
    // Clear localStorage
    localStorage.clear();
    
    // Mock getCurrentUser to return a rejected promise by default
    // This prevents the useEffect from trying to fetch user on mount
    vi.mocked(authApi.getCurrentUser).mockRejectedValue(new Error('Not authenticated'));
  });

  describe('login', () => {
    it('should login successfully and store token', async () => {
      const mockResponse = {
        data: {
          token: 'fake-token-123',
          user: {
            id: 1,
            username: 'testuser',
            email: 'test@example.com',
            is_admin: false,
          },
        },
      };

      vi.mocked(authApi.login).mockResolvedValue(mockResponse);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();

      await act(async () => {
        await result.current.login('testuser', 'password123');
      });

      await waitFor(() => {
        expect(result.current.isAuthenticated).toBe(true);
        expect(result.current.user).toEqual({
          id: 1,
          username: 'testuser',
          email: 'test@example.com',
          is_admin: false,
        });
        expect(result.current.token).toBe('fake-token-123');
        expect(localStorage.getItem('token')).toBe('fake-token-123');
      });

      expect(authApi.login).toHaveBeenCalledWith('testuser', 'password123');
    });

    it('should handle login failure', async () => {
      vi.mocked(authApi.login).mockRejectedValue(new Error('Invalid credentials'));

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await expect(async () => {
        await act(async () => {
          await result.current.login('testuser', 'wrongpassword');
        });
      }).rejects.toThrow('Invalid credentials');

      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();
    });
  });

  describe('register', () => {
    it('should register successfully and store token', async () => {
      const mockResponse = {
        data: {
          token: 'new-user-token',
          user: {
            id: 2,
            username: 'newuser',
            email: 'new@example.com',
            is_admin: false,
          },
        },
      };

      vi.mocked(authApi.register).mockResolvedValue(mockResponse);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await act(async () => {
        await result.current.register('newuser', 'new@example.com', 'password123');
      });

      await waitFor(() => {
        expect(result.current.isAuthenticated).toBe(true);
        expect(result.current.user?.username).toBe('newuser');
        expect(localStorage.getItem('token')).toBe('new-user-token');
      });

      expect(authApi.register).toHaveBeenCalledWith('newuser', 'new@example.com', 'password123');
    });
  });

  describe('logout', () => {
    it('should logout successfully and clear token', async () => {
      // First login
      const mockResponse = {
        data: {
          token: 'fake-token',
          user: {
            id: 1,
            username: 'testuser',
            email: 'test@example.com',
            is_admin: false,
          },
        },
      };

      vi.mocked(authApi.login).mockResolvedValue(mockResponse);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await act(async () => {
        await result.current.login('testuser', 'password123');
      });

      await waitFor(() => {
        expect(result.current.isAuthenticated).toBe(true);
      });

      // Now logout
      act(() => {
        result.current.logout();
      });

      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();
      expect(result.current.token).toBeNull();
      expect(localStorage.getItem('token')).toBeNull();
    });
  });

  describe('isAdmin', () => {
    it('should return true for admin users', async () => {
      const mockResponse = {
        data: {
          token: 'admin-token',
          user: {
            id: 1,
            username: 'admin',
            email: 'admin@example.com',
            is_admin: true,
          },
        },
      };

      vi.mocked(authApi.login).mockResolvedValue(mockResponse);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await act(async () => {
        await result.current.login('admin', 'password');
      });

      await waitFor(() => {
        expect(result.current.isAdmin).toBe(true);
      });
    });

    it('should return false for non-admin users', async () => {
      const mockResponse = {
        data: {
          token: 'user-token',
          user: {
            id: 2,
            username: 'regularuser',
            email: 'user@example.com',
            is_admin: false,
          },
        },
      };

      vi.mocked(authApi.login).mockResolvedValue(mockResponse);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await act(async () => {
        await result.current.login('regularuser', 'password');
      });

      await waitFor(() => {
        expect(result.current.isAdmin).toBe(false);
      });
    });
  });

  describe('token persistence', () => {
    it('should restore session from localStorage on mount', async () => {
      // Set up localStorage with existing token
      localStorage.setItem('token', 'existing-token');

      const mockUser: User = {
        id: 1,
        username: 'existinguser',
        email: 'existing@example.com',
        is_admin: false,
      };

      vi.mocked(authApi.getCurrentUser).mockResolvedValue({
        data: mockUser,
      } as AxiosResponse<User>);

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      // Initially, token should be set but user not yet loaded
      expect(result.current.token).toBe('existing-token');

      // Wait for user to be fetched
      await waitFor(() => {
        expect(result.current.user).toEqual(mockUser);
        expect(result.current.isAuthenticated).toBe(true);
      });

      expect(authApi.getCurrentUser).toHaveBeenCalled();
    });

    it('should clear invalid token from localStorage', async () => {
      localStorage.setItem('token', 'invalid-token');

      vi.mocked(authApi.getCurrentUser).mockRejectedValue(new Error('Unauthorized'));

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitFor(() => {
        expect(result.current.token).toBeNull();
        expect(result.current.user).toBeNull();
        expect(localStorage.getItem('token')).toBeNull();
      });
    });
  });
});
