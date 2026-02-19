import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDebounce } from './useDebounce';

describe('useDebounce', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns the initial value immediately', () => {
    const { result } = renderHook(() => useDebounce('hello', 300));
    expect(result.current).toBe('hello');
  });

  it('does not update the value before the delay elapses', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'initial', delay: 300 } },
    );

    rerender({ value: 'updated', delay: 300 });
    expect(result.current).toBe('initial');
  });

  it('updates the value after the delay elapses', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'initial', delay: 300 } },
    );

    rerender({ value: 'updated', delay: 300 });
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(result.current).toBe('updated');
  });

  it('resets the timer when value changes before delay elapses', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'initial', delay: 300 } },
    );

    rerender({ value: 'first', delay: 300 });
    act(() => { vi.advanceTimersByTime(200); });

    rerender({ value: 'second', delay: 300 });
    act(() => { vi.advanceTimersByTime(200); }); // only 200ms since 'second'

    expect(result.current).toBe('initial'); // not yet committed

    act(() => { vi.advanceTimersByTime(100); }); // now 300ms since 'second'
    expect(result.current).toBe('second');
  });

  it('works with non-string types', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 0, delay: 100 } },
    );

    rerender({ value: 42, delay: 100 });
    act(() => { vi.advanceTimersByTime(100); });
    expect(result.current).toBe(42);
  });
});
