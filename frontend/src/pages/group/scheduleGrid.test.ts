import { describe, it, expect } from 'vitest';
import {
  isWeekendDay,
  maxHourFor,
  slotDurationMinutes,
  formatHourLabel,
  formatSlotRangeLabel,
  rowHeaderFor,
  nextCadence,
  weekParity,
  upcomingDatesForParity,
} from './scheduleGrid';

describe('day-aware hour rules', () => {
  it('caps weekends at hour 15 and weekdays at hour 17', () => {
    expect(maxHourFor(0)).toBe(15);
    expect(maxHourFor(6)).toBe(15);
    expect(maxHourFor(1)).toBe(17);
    expect(maxHourFor(5)).toBe(17);
  });

  it('flags only Sunday and Saturday as weekend', () => {
    expect(isWeekendDay(0)).toBe(true);
    expect(isWeekendDay(6)).toBe(true);
    expect(isWeekendDay(1)).toBe(false);
  });

  it('the terminal hour is 90 minutes, everything else is 60', () => {
    expect(slotDurationMinutes(3, 14)).toBe(60);
    expect(slotDurationMinutes(3, 17)).toBe(90);
    expect(slotDurationMinutes(0, 10)).toBe(60);
    expect(slotDurationMinutes(6, 15)).toBe(90);
  });

  it('formatSlotRangeLabel matches formatHourLabel for a normal 60-min slot', () => {
    expect(formatSlotRangeLabel(3, 14)).toBe(formatHourLabel(14));
  });

  it('formatSlotRangeLabel shows a range for the terminal weekday slot', () => {
    expect(formatSlotRangeLabel(3, 17)).toBe('5:00–6:30 PM');
  });

  it('formatSlotRangeLabel shows a range for the terminal weekend slot', () => {
    expect(formatSlotRangeLabel(6, 15)).toBe('3:00–4:30 PM');
  });
});

describe('rowHeaderFor', () => {
  it('shows a plain hour range when weekday and weekend agree', () => {
    expect(rowHeaderFor(10)).toEqual({ label: '10:00–11:00 AM' });
  });

  it('shows the weekday-only terminal hour as its own full range, no note', () => {
    expect(rowHeaderFor(17)).toEqual({ label: '5:00–6:30 PM' });
  });

  it('flags the shared hour where weekend terminates but weekday continues', () => {
    expect(rowHeaderFor(15)).toEqual({
      label: '3:00–4:00 PM',
      note: 'Weekends end 4:30 PM',
    });
  });
});

describe('nextCadence cycling', () => {
  it('cycles empty -> weekly -> biweekly_a -> biweekly_b -> empty', () => {
    expect(nextCadence(undefined)).toBe('weekly');
    expect(nextCadence('weekly')).toBe('biweekly_a');
    expect(nextCadence('biweekly_a')).toBe('biweekly_b');
    expect(nextCadence('biweekly_b')).toBeUndefined();
  });
});

describe('weekParity', () => {
  it('matches the Go reference: the reference Sunday is A, one week later is B', () => {
    expect(weekParity('2024-01-07')).toBe('a');
    expect(weekParity('2024-01-14')).toBe('b');
    expect(weekParity('2024-01-21')).toBe('a');
    expect(weekParity('2023-12-31')).toBe('b');
  });
});

describe('upcomingDatesForParity', () => {
  it('returns dates 2 weeks apart, all matching the requested parity', () => {
    const datesA = upcomingDatesForParity('a', 3);
    expect(datesA).toHaveLength(3);
    datesA.forEach(d => expect(weekParity(d)).toBe('a'));
    // consecutive entries are exactly 14 days apart
    const toUtcDays = (iso: string) => Date.UTC(...(iso.split('-').map(Number) as [number, number, number]).map((v, i) => i === 1 ? v - 1 : v) as [number, number, number]) / 86400000;
    expect(toUtcDays(datesA[1]) - toUtcDays(datesA[0])).toBe(14);
  });
});
