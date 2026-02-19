import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useConfirmDialog } from './useConfirmDialog';

describe('useConfirmDialog', () => {
  it('initializes with dialog closed', () => {
    const { result } = renderHook(() => useConfirmDialog());
    expect(result.current.confirmDialog.isOpen).toBe(false);
  });

  it('opens the dialog with the provided title, message, and callback', () => {
    const { result } = renderHook(() => useConfirmDialog());
    const onConfirm = vi.fn();

    act(() => {
      result.current.openConfirmDialog('Delete Item', 'Are you sure?', onConfirm);
    });

    expect(result.current.confirmDialog.isOpen).toBe(true);
    expect(result.current.confirmDialog.title).toBe('Delete Item');
    expect(result.current.confirmDialog.message).toBe('Are you sure?');
    expect(result.current.confirmDialog.onConfirm).toBe(onConfirm);
  });

  it('closes the dialog and resets all fields', () => {
    const { result } = renderHook(() => useConfirmDialog());
    const onConfirm = vi.fn();

    act(() => {
      result.current.openConfirmDialog('Title', 'Message', onConfirm);
    });
    act(() => {
      result.current.closeConfirmDialog();
    });

    expect(result.current.confirmDialog.isOpen).toBe(false);
    expect(result.current.confirmDialog.title).toBe('');
    expect(result.current.confirmDialog.message).toBe('');
  });

  it('openConfirmDialog is stable across renders', () => {
    const { result, rerender } = renderHook(() => useConfirmDialog());
    const firstRef = result.current.openConfirmDialog;
    rerender();
    expect(result.current.openConfirmDialog).toBe(firstRef);
  });

  it('closeConfirmDialog is stable across renders', () => {
    const { result, rerender } = renderHook(() => useConfirmDialog());
    const firstRef = result.current.closeConfirmDialog;
    rerender();
    expect(result.current.closeConfirmDialog).toBe(firstRef);
  });

  it('can open a second dialog after closing the first', () => {
    const { result } = renderHook(() => useConfirmDialog());

    act(() => {
      result.current.openConfirmDialog('First', 'First message', vi.fn());
    });
    act(() => {
      result.current.closeConfirmDialog();
    });
    act(() => {
      result.current.openConfirmDialog('Second', 'Second message', vi.fn());
    });

    expect(result.current.confirmDialog.isOpen).toBe(true);
    expect(result.current.confirmDialog.title).toBe('Second');
  });
});
