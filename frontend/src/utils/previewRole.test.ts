import { describe, it, expect, beforeEach } from 'vitest';
import { previewRoleKey, clearPreviewRoleState } from './previewRole';

describe('previewRole', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  describe('previewRoleKey', () => {
    it('namespaces the key by group id', () => {
      expect(previewRoleKey(5)).toBe('previewRole:5');
      expect(previewRoleKey('5')).toBe('previewRole:5');
    });
  });

  describe('clearPreviewRoleState', () => {
    it('removes every previewRole:* entry, regardless of group', () => {
      sessionStorage.setItem(previewRoleKey(1), 'member');
      sessionStorage.setItem(previewRoleKey(2), 'group_admin');
      sessionStorage.setItem('unrelatedKey', 'keep-me');

      clearPreviewRoleState();

      expect(sessionStorage.getItem(previewRoleKey(1))).toBeNull();
      expect(sessionStorage.getItem(previewRoleKey(2))).toBeNull();
      expect(sessionStorage.getItem('unrelatedKey')).toBe('keep-me');
    });

    it('is a no-op when nothing is stored', () => {
      expect(() => clearPreviewRoleState()).not.toThrow();
    });
  });
});
