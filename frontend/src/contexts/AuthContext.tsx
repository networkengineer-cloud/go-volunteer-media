import React, { createContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../api/client';
import type { User } from '../api/client';

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isGroupAdmin: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Export the context so it can be used by the hook in a separate file
export { AuthContext };

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(() => {
    try {
      return localStorage.getItem('token');
    } catch {
      return null;
    }
  });
  const [isLoading, setIsLoading] = useState<boolean>(!!token);

  useEffect(() => {
    if (token) {
      setIsLoading(true);
      authApi.getCurrentUser()
        .then(response => {
          setUser(response.data);
        })
        .catch(() => {
          try {
            localStorage.removeItem('token');
          } catch {
            // ignore storage errors
          }
          setToken(null);
        })
        .finally(() => {
          setIsLoading(false);
        });
    } else {
      setIsLoading(false);
    }
  }, [token]);

  const login = async (username: string, password: string) => {
    const response = await authApi.login(username, password);
    setToken(response.data.token);
    setUser(response.data.user);
    try {
      localStorage.setItem('token', response.data.token);
    } catch {
      // ignore storage errors
    }
  };

  const register = async (username: string, email: string, password: string) => {
    const response = await authApi.register(username, email, password);
    setToken(response.data.token);
    setUser(response.data.user);
    try {
      localStorage.setItem('token', response.data.token);
    } catch {
      // ignore storage errors
    }
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    try {
      localStorage.removeItem('token');
    } catch {
      // ignore storage errors
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isLoading,
        login,
        register,
        logout,
        isAuthenticated: !!token,
        isAdmin: user?.is_admin || false,
        isGroupAdmin: user?.is_group_admin || false,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};
