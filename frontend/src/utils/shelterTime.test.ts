import { describe, it, expect } from 'vitest';
import { isValidTimeZone, listTimeZones, todayInZone, weekStartOfIso } from './shelterTime';

describe('todayInZone', () => {
  // 03:30 UTC on Sep 29 is still Sep 28 in Chicago (UTC-5 in September).
  const instant = new Date(Date.UTC(2026, 8, 29, 3, 30));

  it("returns the shelter's calendar date, not UTC's", () => {
    expect(todayInZone('America/Chicago', instant)).toBe('2026-09-28');
    expect(todayInZone('UTC', instant)).toBe('2026-09-29');
  });

  it('falls back to UTC for an unknown zone', () => {
    expect(todayInZone('Mars/Olympus', instant)).toBe('2026-09-29');
    expect(todayInZone('', instant)).toBe('2026-09-29');
  });
});

describe('weekStartOfIso', () => {
  it('snaps to the Sunday on or before the date', () => {
    expect(weekStartOfIso('2026-09-27')).toBe('2026-09-27'); // Sunday
    expect(weekStartOfIso('2026-09-28')).toBe('2026-09-27'); // Monday
    expect(weekStartOfIso('2026-10-03')).toBe('2026-09-27'); // Saturday
  });
});

describe('isValidTimeZone / listTimeZones', () => {
  it('validates IANA names', () => {
    expect(isValidTimeZone('America/Chicago')).toBe(true);
    expect(isValidTimeZone('Mars/Olympus')).toBe(false);
    expect(isValidTimeZone('')).toBe(false);
  });

  it('lists zones including UTC and common US zones', () => {
    const zones = listTimeZones();
    expect(zones).toContain('UTC');
    expect(zones).toContain('America/Chicago');
  });
});
