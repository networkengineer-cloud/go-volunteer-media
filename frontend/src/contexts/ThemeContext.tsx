import React, { createContext, useState, useEffect, useContext } from 'react';
import type { ReactNode } from 'react';

type Theme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const ThemeProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  // Single localStorage read shared between both state initializers
  const storedTheme = (() => {
    if (typeof window === 'undefined') return null;
    try {
      const stored = localStorage.getItem('theme');
      return stored === 'light' || stored === 'dark' ? stored : null;
    } catch {
      return null;
    }
  })();

  const [hasUserPreference, setHasUserPreference] = useState<boolean>(
    () => storedTheme !== null,
  );

  const [theme, setTheme] = useState<Theme>(() => {
    if (storedTheme) return storedTheme;
    return typeof window !== 'undefined' &&
      window.matchMedia?.('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const root = document.documentElement;
    if (theme === 'dark') {
      root.setAttribute('data-theme', 'dark');
    } else {
      root.removeAttribute('data-theme');
    }

    try {
      if (hasUserPreference) {
        localStorage.setItem('theme', theme);
      } else {
        localStorage.removeItem('theme');
      }
    } catch {
      // ignore storage errors (e.g., private mode)
    }
  }, [theme, hasUserPreference]);

  useEffect(() => {
    if (typeof window === 'undefined' || hasUserPreference || !window.matchMedia) return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (event: MediaQueryListEvent) => {
      setTheme(event.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [hasUserPreference]);

  // TODO: Add a "Follow system" option in Settings to allow users to clear the stored
  // preference and re-enable OS-level theme tracking (setHasUserPreference(false) +
  // localStorage.removeItem('theme')). Once hasUserPreference is true it persists across
  // sessions with no user-facing escape hatch.
  const toggleTheme = () => {
    setHasUserPreference(true);
    setTheme((currentTheme) => (currentTheme === 'dark' ? 'light' : 'dark'));
  };

  const setThemePreference = (nextTheme: Theme) => {
    setHasUserPreference(true);
    setTheme(nextTheme);
  };

  return (
    <ThemeContext.Provider value={{ theme, setTheme: setThemePreference, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export function useTheme(): ThemeContextType {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}
