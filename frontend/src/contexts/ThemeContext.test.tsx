import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { ThemeProvider, useTheme } from './ThemeContext';

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });
  });

  it('throws when useTheme is used outside ThemeProvider', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      renderHook(() => useTheme());
    }).toThrow('useTheme must be used within a ThemeProvider');

    consoleSpy.mockRestore();
  });

  it('initializes from localStorage when theme is stored', async () => {
    localStorage.setItem('theme', 'dark');

    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    });
  });

  it('falls back to system preference when no stored theme exists', async () => {
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });

    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
      expect(localStorage.getItem('theme')).toBeNull();
    });
  });

  it('updates with system preference changes when no stored theme exists', async () => {
    let changeHandler: ((event: MediaQueryListEvent) => void) | undefined;

    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((event: string, callback: (event: MediaQueryListEvent) => void) => {
        if (event === 'change') {
          changeHandler = callback;
        }
      }),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });

    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('light');
      expect(localStorage.getItem('theme')).toBeNull();
    });

    act(() => {
      changeHandler?.({ matches: true } as MediaQueryListEvent);
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
      expect(localStorage.getItem('theme')).toBeNull();
    });
  });

  it('toggles theme and persists changes to localStorage', async () => {
    const { result } = renderHook(() => useTheme(), {
      wrapper: ThemeProvider,
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('light');
      expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
      expect(localStorage.getItem('theme')).toBeNull();
    });

    act(() => {
      result.current.toggleTheme();
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('dark');
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
      expect(localStorage.getItem('theme')).toBe('dark');
    });

    act(() => {
      result.current.toggleTheme();
    });

    await waitFor(() => {
      expect(result.current.theme).toBe('light');
      expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
      expect(localStorage.getItem('theme')).toBe('light');
    });
  });
});
