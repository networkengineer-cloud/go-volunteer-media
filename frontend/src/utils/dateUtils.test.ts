import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  formatDateShort,
  formatDateLong,
  formatRelativeTime,
  calculateQuarantineEndDate,
  calculateDaysSince,
} from './dateUtils';

describe('dateUtils', () => {
  describe('formatDateShort', () => {
    it('returns "-" for undefined', () => {
      expect(formatDateShort(undefined)).toBe('-');
    });

    it('returns "-" for empty string', () => {
      expect(formatDateShort('')).toBe('-');
    });

    it('returns "-" for invalid date string', () => {
      expect(formatDateShort('not-a-date')).toBe('-');
    });

    it('formats a valid date', () => {
      // Use noon UTC to avoid timezone-shift to previous day in western timezones
      expect(formatDateShort('2024-06-15T12:00:00Z')).toMatch(/Jun\s+15,\s+2024/);
    });
  });

  describe('formatDateLong', () => {
    it('returns "-" for undefined', () => {
      expect(formatDateLong(undefined)).toBe('-');
    });

    it('returns "-" for invalid date string', () => {
      expect(formatDateLong('garbage')).toBe('-');
    });

    it('formats a valid date in long form', () => {
      expect(formatDateLong('2024-06-15T12:00:00Z')).toMatch(/June\s+15,\s+2024/);
    });
  });

  describe('formatRelativeTime', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns "-" for an invalid date string', () => {
      expect(formatRelativeTime('not-a-date')).toBe('-');
    });

    it('returns the formatted date for future dates', () => {
      const result = formatRelativeTime('2024-06-16T12:00:00Z');
      expect(result).not.toBe('Just now');
      expect(result).not.toMatch(/ago/);
    });

    it('returns "Just now" for less than 1 minute ago', () => {
      expect(formatRelativeTime('2024-06-15T11:59:30Z')).toBe('Just now');
    });

    it('returns minutes for less than 1 hour ago', () => {
      expect(formatRelativeTime('2024-06-15T11:30:00Z')).toBe('30m ago');
    });

    it('returns hours for less than 24 hours ago', () => {
      expect(formatRelativeTime('2024-06-15T06:00:00Z')).toBe('6h ago');
    });

    it('returns days within cutoff window', () => {
      expect(formatRelativeTime('2024-06-10T12:00:00Z')).toBe('5d ago');
    });

    it('returns formatted date beyond default 30-day cutoff', () => {
      const result = formatRelativeTime('2024-05-01T12:00:00Z');
      expect(result).toMatch(/May/);
      expect(result).not.toMatch(/ago/);
    });

    it('respects a custom cutoffDays parameter', () => {
      // 5 days ago, with cutoff of 3 → should show formatted date
      const result = formatRelativeTime('2024-06-10T12:00:00Z', 3);
      expect(result).toMatch(/Jun/);
      expect(result).not.toMatch(/ago/);
    });
  });

  describe('calculateQuarantineEndDate', () => {
    it('returns "-" for undefined', () => {
      expect(calculateQuarantineEndDate(undefined)).toBe('-');
    });

    it('returns "-" for invalid date string', () => {
      expect(calculateQuarantineEndDate('bad-date')).toBe('-');
    });

    it('adds 10 days to a start date that lands on a weekday', () => {
      // Use noon UTC to avoid timezone-shift in western timezones
      // 2024-06-03T12:00Z (Mon local) + 10 = 2024-06-13 (Thu)
      expect(calculateQuarantineEndDate('2024-06-03T12:00:00Z')).toMatch(/Jun\s+13,\s+2024/);
    });

    it('skips Saturday and returns Monday', () => {
      // 2024-05-15T12:00Z (Wed local) + 10 = 2024-05-25 (Sat) → skip to Mon 2024-05-27
      const result = calculateQuarantineEndDate('2024-05-15T12:00:00Z');
      expect(result).toMatch(/May\s+27,\s+2024/);
    });

    it('skips Sunday and returns Monday', () => {
      // 2024-05-16T12:00Z (Thu local) + 10 = 2024-05-26 (Sun) → skip to Mon 2024-05-27
      const result = calculateQuarantineEndDate('2024-05-16T12:00:00Z');
      expect(result).toMatch(/May\s+27,\s+2024/);
    });

    it('returns long format when requested', () => {
      const result = calculateQuarantineEndDate('2024-06-03T12:00:00Z', 'long');
      expect(result).toMatch(/June\s+13,\s+2024/);
    });
  });

  describe('calculateDaysSince', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns 0 for undefined', () => {
      expect(calculateDaysSince(undefined)).toBe(0);
    });

    it('returns 0 for invalid date string', () => {
      expect(calculateDaysSince('garbage')).toBe(0);
    });

    it('returns 1 for a date 1 second in the past (ceil behaviour)', () => {
      expect(calculateDaysSince('2024-06-15T11:59:59Z')).toBe(1);
    });

    it('returns correct days for a past date', () => {
      expect(calculateDaysSince('2024-06-10T12:00:00Z')).toBe(5);
    });
  });
});
