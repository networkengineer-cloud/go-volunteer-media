import type { ScheduleCadence } from '../../api/client';

export const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
export const HOURS = [8, 9, 10, 11, 12, 13, 14, 15, 16, 17];

export function slotKey(dayOfWeek: number, hour: number): string {
  return `${dayOfWeek}-${hour}`;
}

function formatClock(hour24: number, minute: number): string {
  const period = hour24 < 12 ? 'AM' : 'PM';
  const displayHour = hour24 % 12 === 0 ? 12 : hour24 % 12;
  return minute === 0 ? `${displayHour}:00 ${period}` : `${displayHour}:${String(minute).padStart(2, '0')} ${period}`;
}

export function formatHourLabel(hour: number): string {
  return formatClock(hour, 0);
}

// isWeekendDay/maxHourFor/slotDurationMinutes mirror
// internal/handlers/schedule_hours.go exactly - keep both in sync if the
// hour rules ever change.
export function isWeekendDay(dayOfWeek: number): boolean {
  return dayOfWeek === 0 || dayOfWeek === 6;
}

export function maxHourFor(dayOfWeek: number): number {
  return isWeekendDay(dayOfWeek) ? 15 : 17;
}

export function slotDurationMinutes(dayOfWeek: number, hour: number): number {
  return hour === maxHourFor(dayOfWeek) ? 90 : 60;
}

// formatSlotRangeLabel returns a plain start-time label ("5:00 PM") for a
// normal 60-min slot, or a "start–end" range ("5:00–6:30 PM") for the
// day's terminal 90-min slot.
export function formatSlotRangeLabel(dayOfWeek: number, hour: number): string {
  const duration = slotDurationMinutes(dayOfWeek, hour);
  if (duration === 60) return formatHourLabel(hour);
  return formatRangeLabel(hour, duration);
}

export interface RowHeaderInfo {
  label: string;
  note?: string;
}

function endClock(hour: number, durationMinutes: number): string {
  const endTotalMinutes = hour * 60 + durationMinutes;
  return formatClock(Math.floor(endTotalMinutes / 60), endTotalMinutes % 60);
}

function formatRangeLabel(hour: number, durationMinutes: number): string {
  const startTime = formatClock(hour, 0).replace(/ (AM|PM)$/, '');
  return `${startTime}–${endClock(hour, durationMinutes)}`;
}

// rowHeaderFor labels a schedule-grid row with its actual time range rather
// than just a bare start time. Most rows are uniform (weekday and weekend
// cells both 60 min), so a plain range suffices. The one row where they
// diverge - weekend's terminal (90-min) hour, which weekdays haven't reached
// yet - can't be truthfully summarized by a single range, so it states the
// weekday case in the label and calls out the weekend's real end time in a
// separate note rather than silently picking one and misleading the other.
export function rowHeaderFor(hour: number): RowHeaderInfo {
  const weekdayEnabled = hour <= maxHourFor(1);
  const weekendEnabled = hour <= maxHourFor(0);
  const weekdayDuration = weekdayEnabled ? slotDurationMinutes(1, hour) : null;
  const weekendDuration = weekendEnabled ? slotDurationMinutes(0, hour) : null;

  if (weekdayDuration !== null && weekendDuration !== null && weekdayDuration !== weekendDuration) {
    return {
      label: formatRangeLabel(hour, weekdayDuration),
      note: `Weekends end ${endClock(hour, weekendDuration)}`,
    };
  }
  const duration = weekdayDuration ?? weekendDuration ?? 60;
  return { label: formatRangeLabel(hour, duration) };
}

// nextCadence advances a grid cell through its 4-state cycle when clicked:
// empty (undefined) -> weekly -> biweekly_a -> biweekly_b -> empty.
export function nextCadence(current: ScheduleCadence | undefined): ScheduleCadence | undefined {
  switch (current) {
    case undefined:
      return 'weekly';
    case 'weekly':
      return 'biweekly_a';
    case 'biweekly_a':
      return 'biweekly_b';
    case 'biweekly_b':
      return undefined;
  }
}

// biweeklyReferenceSunday must stay exactly in sync with
// internal/handlers/schedule_hours.go's biweeklyReferenceSunday.
const BIWEEKLY_REFERENCE_SUNDAY = '2024-01-07';

function isoToUtcDate(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d));
}

function addWeeksIso(iso: string, weeks: number): string {
  const date = isoToUtcDate(iso);
  date.setUTCDate(date.getUTCDate() + weeks * 7);
  return date.toISOString().slice(0, 10);
}

// weekParity classifies weekStartIso (any date within the target week - it
// need not already be a Sunday) as "a" or "b", matching the Go backend's
// weekParity exactly.
export function weekParity(weekStartIso: string): 'a' | 'b' {
  const ref = isoToUtcDate(BIWEEKLY_REFERENCE_SUNDAY);
  const target = isoToUtcDate(weekStartIso);
  const refSunday = new Date(ref);
  refSunday.setUTCDate(refSunday.getUTCDate() - refSunday.getUTCDay());
  const targetSunday = new Date(target);
  targetSunday.setUTCDate(targetSunday.getUTCDate() - targetSunday.getUTCDay());
  const weeks = Math.round((targetSunday.getTime() - refSunday.getTime()) / (7 * 24 * 60 * 60 * 1000));
  return (((weeks % 2) + 2) % 2) === 0 ? 'a' : 'b';
}

// currentWeekStart returns the ISO date (YYYY-MM-DD) of the Sunday that
// starts "this week" in the viewer's local timezone. Shared by ScheduleTab
// (for the cadence legend) and ScheduleOverview (for both the legend and
// its own week-navigation default).
export function currentWeekStart(): string {
  const now = new Date();
  const utcToday = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()));
  utcToday.setUTCDate(utcToday.getUTCDate() - utcToday.getUTCDay());
  return utcToday.toISOString().slice(0, 10);
}

// upcomingDatesForParity returns `count` Sundays, starting from the current
// week (or the next matching week if the current week doesn't match), each
// 2 weeks apart, all classifying as the given parity - for the "Week A: ..."
// / "Week B: ..." legend.
export function upcomingDatesForParity(parity: 'a' | 'b', count: number): string[] {
  const dates: string[] = [];
  let cursor = currentWeekStart();
  if (weekParity(cursor) !== parity) {
    cursor = addWeeksIso(cursor, 1);
  }
  for (let i = 0; i < count; i++) {
    dates.push(cursor);
    cursor = addWeeksIso(cursor, 2);
  }
  return dates;
}
