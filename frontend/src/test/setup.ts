import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, beforeEach } from 'vitest';

// Mock localStorage with actual storage
const localStorageData: Record<string, string> = {};
const localStorageMock = {
  getItem: (key: string) => localStorageData[key] || null,
  setItem: (key: string, value: string) => {
    localStorageData[key] = value;
  },
  removeItem: (key: string) => {
    delete localStorageData[key];
  },
  clear: () => {
    Object.keys(localStorageData).forEach(key => delete localStorageData[key]);
  },
  get length() {
    return Object.keys(localStorageData).length;
  },
  key: (index: number) => {
    const keys = Object.keys(localStorageData);
    return keys[index] || null;
  },
};
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, writable: true });

// Clear localStorage before each test
beforeEach(() => {
  localStorage.clear();
});

// Cleanup after each test
afterEach(() => {
  cleanup();
});

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => true,
  }),
});

// Mock IntersectionObserver
Object.defineProperty(globalThis, 'IntersectionObserver', {
  value: class IntersectionObserver {
    constructor() {}
    disconnect() {}
    observe() {}
    takeRecords() {
      return [];
    }
    unobserve() {}
  } as unknown as typeof IntersectionObserver,
  writable: true,
});
