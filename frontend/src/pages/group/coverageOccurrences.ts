import type { ScheduleSlot } from '../../api/client';
import { DEFAULT_SHELTER_TIMEZONE, todayInZone } from '../../utils/shelterTime';
import { maxHourFor, weekParity } from './scheduleGrid';

// One calendar occurrence (date + start hour) of a volunteer's recurring shift.
export interface Occurrence {
  date: string;
  hour: number;
}

// computeCandidateOccurrences finds every date within [startDate, endDate]
// (inclusive, both YYYY-MM-DD) whose weekday matches one of the given
// slots, paired with that slot's hour - i.e. every real calendar
// occurrence of the user's recurring pattern within the range. Excludes
// dates before `today` - the shelter's current date (todayInZone with the
// shelter_timezone setting) - mirroring the backend's own past-date rejection so
// the checklist never pre-checks something that would just get skipped
// server-side.
//
// Also excludes a slot/date pairing when either:
//   - the slot's hour is beyond maxHourFor(dayOfWeek) - a legacy slot from
//     before an hour-range tightening (e.g. a pre-existing weekend 5pm slot
//     now that weekends cap at 15) would otherwise be offered as a
//     candidate and then fail CreateCoverageRequestsBatch's per-item parse
//     loop, which rejects an out-of-range hour with a whole-batch 400
//     rather than a per-item skip; or
//   - the slot has a biweekly cadence and the occurrence date falls on a
//     week of the opposite parity, where slotActiveForWeek would report the
//     slot inactive server-side - offering it here would let the user check
//     a box that then silently fails with a "no matching shift slot" skip
//     reason, and needlessly inflates the candidate count against the
//     batch cap.
export function computeCandidateOccurrences(
  slots: ScheduleSlot[],
  startDate: string,
  endDate: string,
  today: string = todayInZone(DEFAULT_SHELTER_TIMEZONE),
): Occurrence[] {
  if (!startDate || !endDate || startDate > endDate) return [];
  const [sy, sm, sd] = startDate.split('-').map(Number);
  const [ey, em, ed] = endDate.split('-').map(Number);
  const start = new Date(Date.UTC(sy, sm - 1, sd));
  const end = new Date(Date.UTC(ey, em - 1, ed));

  const occurrences: Occurrence[] = [];
  for (const cursor = new Date(start); cursor <= end; cursor.setUTCDate(cursor.getUTCDate() + 1)) {
    const dateStr = cursor.toISOString().slice(0, 10);
    if (dateStr < today) continue;
    const dayOfWeek = cursor.getUTCDay();
    for (const slot of slots) {
      if (slot.day_of_week !== dayOfWeek) continue;
      if (slot.hour > maxHourFor(dayOfWeek)) continue;
      if (slot.cadence === 'biweekly_a' && weekParity(dateStr) !== 'a') continue;
      if (slot.cadence === 'biweekly_b' && weekParity(dateStr) !== 'b') continue;
      occurrences.push({ date: dateStr, hour: slot.hour });
    }
  }
  occurrences.sort((a, b) => (a.date === b.date ? a.hour - b.hour : a.date.localeCompare(b.date)));
  return occurrences;
}
