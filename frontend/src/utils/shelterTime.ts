// Shelter time zone helpers (roadmap AR-5). "Today" and "this week" for
// scheduling are the shelter's calendar, not the viewer's and not UTC, so
// that the UI agrees with the backend (internal/shelterclock). The zone is
// the `shelter_timezone` site setting, an IANA name such as
// "America/Chicago".

export const DEFAULT_SHELTER_TIMEZONE = 'UTC';

// isValidTimeZone reports whether the runtime recognizes an IANA zone name.
export function isValidTimeZone(timeZone: string): boolean {
  if (!timeZone) return false;
  try {
    new Intl.DateTimeFormat('en-US', { timeZone });
    return true;
  } catch {
    return false;
  }
}

// todayInZone returns the calendar date (YYYY-MM-DD) that `now` falls on in
// `timeZone`, falling back to UTC for an unknown zone.
export function todayInZone(timeZone: string, now: Date = new Date()): string {
  const zone = isValidTimeZone(timeZone) ? timeZone : DEFAULT_SHELTER_TIMEZONE;
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: zone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(now);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')}`;
}

// weekStartOfIso returns the Sunday (YYYY-MM-DD) on or before an ISO date.
export function weekStartOfIso(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number);
  const date = new Date(Date.UTC(y, m - 1, d));
  date.setUTCDate(date.getUTCDate() - date.getUTCDay());
  return date.toISOString().slice(0, 10);
}

// listTimeZones returns the IANA zones the runtime knows, for the admin
// picker. Always includes UTC.
export function listTimeZones(): string[] {
  const supported =
    typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : [];
  return supported.includes('UTC') ? supported : ['UTC', ...supported];
}
