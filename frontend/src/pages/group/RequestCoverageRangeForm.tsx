import React, { useEffect, useMemo, useState } from 'react';
import { scheduleApi } from '../../api/client';
import type { ScheduleSlot, CoverageRequestBatchItem, CoverageRequestBatchResult, CoverageRequestPriority } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { formatSlotRangeLabel, maxHourFor, weekParity } from './scheduleGrid';
import './RequestCoverageRangeForm.css';

export interface RequestCoverageRangeFormProps {
  groupId: number;
  slots: ScheduleSlot[];
  initialStartDate?: string;
  initialEndDate?: string;
  onSuccess?: () => void;
  onCancel?: () => void;
}

const MAX_RANGE_DAYS = 90;
// Must match maxBatchItems in internal/handlers/schedule_coverage.go's
// CreateCoverageRequestsBatch - a wide date range with a busy recurring
// schedule (e.g. 90 days x several shifts/week) can exceed the backend's
// cap even though it's within MAX_RANGE_DAYS, so this is enforced
// separately, with the selected count always visible so an over-cap
// selection is something the user can actually act on rather than a bare
// rejected-batch error.
const MAX_BATCH_ITEMS = 200;

interface Occurrence {
  date: string;
  hour: number;
}

function occurrenceKey(o: Occurrence): string {
  return `${o.date}-${o.hour}`;
}

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

// Occurrences/skipped items only carry a `date` (not a day_of_week), so the
// day-of-week needed by formatSlotRangeLabel (to know whether this item's
// hour is a day's terminal 90-min slot) is derived here from that date.
function dayOfWeekFromIso(isoDate: string): number {
  const [y, m, d] = isoDate.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d)).getUTCDay();
}

// computeCandidateOccurrences finds every date within [startDate, endDate]
// (inclusive, both YYYY-MM-DD) whose weekday matches one of the given
// slots, paired with that slot's hour - i.e. every real calendar
// occurrence of the user's recurring pattern within the range. Excludes
// dates before today, mirroring the backend's own past-date rejection so
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
export function computeCandidateOccurrences(slots: ScheduleSlot[], startDate: string, endDate: string): Occurrence[] {
  if (!startDate || !endDate || startDate > endDate) return [];
  const today = todayIso();
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

function rangeExceedsMaxDays(startDate: string, endDate: string): boolean {
  if (!startDate || !endDate) return false;
  const [sy, sm, sd] = startDate.split('-').map(Number);
  const [ey, em, ed] = endDate.split('-').map(Number);
  const start = Date.UTC(sy, sm - 1, sd);
  const end = Date.UTC(ey, em - 1, ed);
  return (end - start) / (1000 * 60 * 60 * 24) > MAX_RANGE_DAYS;
}

const RequestCoverageRangeForm: React.FC<RequestCoverageRangeFormProps> = ({ groupId, slots, initialStartDate, initialEndDate, onSuccess, onCancel }) => {
  const toast = useToast();
  const [startDate, setStartDate] = useState(initialStartDate ?? '');
  const [endDate, setEndDate] = useState(initialEndDate ?? '');
  const [checkedKeys, setCheckedKeys] = useState<Set<string>>(new Set());
  const [priority, setPriority] = useState<CoverageRequestPriority>('normal');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<CoverageRequestBatchResult | null>(null);

  const rangeTooLong = rangeExceedsMaxDays(startDate, endDate);
  const candidates = useMemo(
    () => (rangeTooLong ? [] : computeCandidateOccurrences(slots, startDate, endDate)),
    [slots, startDate, endDate, rangeTooLong]
  );

  // Re-derive which occurrences are checked whenever the candidate list
  // itself changes (start/end date edited, or the range became too long) -
  // a side effect, so it belongs in useEffect, not inside the useMemo above.
  useEffect(() => {
    setCheckedKeys(new Set(candidates.map(occurrenceKey)));
  }, [candidates]);

  const toggleOccurrence = (key: string) => {
    setCheckedKeys(prev => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const allChecked = candidates.length > 0 && candidates.every(o => checkedKeys.has(occurrenceKey(o)));
  const toggleAll = () => {
    setCheckedKeys(allChecked ? new Set() : new Set(candidates.map(occurrenceKey)));
  };

  const handleSubmit = () => {
    const requests: CoverageRequestBatchItem[] = candidates
      .filter(o => checkedKeys.has(occurrenceKey(o)))
      .map(o => ({ date: o.date, hour: o.hour }));
    if (requests.length === 0) {
      toast.showError('Select at least one shift.');
      return;
    }
    if (requests.length > MAX_BATCH_ITEMS) {
      toast.showError(`You can request coverage for at most ${MAX_BATCH_ITEMS} shifts at once.`);
      return;
    }
    setSubmitting(true);
    scheduleApi.createCoverageRequestsBatch(groupId, requests, priority)
      .then(res => {
        setResult(res.data);
        if (res.data.created.length > 0) {
          toast.showSuccess(`Requested coverage for ${res.data.created.length} shift${res.data.created.length === 1 ? '' : 's'}.`);
        }
        onSuccess?.();
      })
      .catch(err => {
        const message = (err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to request coverage.';
        toast.showError(message);
      })
      .finally(() => setSubmitting(false));
  };

  if (result) {
    return (
      <div className="request-coverage-range-form">
        <p>Requested coverage for {result.created.length} shift{result.created.length === 1 ? '' : 's'}.</p>
        {result.skipped.length > 0 && (
          <>
            <p>{result.skipped.length} skipped:</p>
            <ul>
              {result.skipped.map(s => (
                <li key={`${s.date}-${s.hour}`}>
                  {s.date} at {formatSlotRangeLabel(dayOfWeekFromIso(s.date), s.hour)} — {s.reason}
                </li>
              ))}
            </ul>
          </>
        )}
        <button type="button" className="btn-primary" onClick={onCancel}>
          Done
        </button>
      </div>
    );
  }

  return (
    <div className="request-coverage-range-form">
      <div className="request-coverage-range-form__dates">
        <label htmlFor="coverage-range-start">Start date</label>
        <input
          id="coverage-range-start"
          type="date"
          value={startDate}
          min={todayIso()}
          onChange={e => setStartDate(e.target.value)}
        />
        <label htmlFor="coverage-range-end">End date</label>
        <input
          id="coverage-range-end"
          type="date"
          value={endDate}
          min={startDate || todayIso()}
          onChange={e => setEndDate(e.target.value)}
        />
      </div>

      {rangeTooLong && (
        <p className="request-coverage-range-form__warning">Please choose a range of {MAX_RANGE_DAYS} days or fewer.</p>
      )}

      <fieldset className="request-coverage-range-form__priority">
        <legend>Priority</legend>
        <label>
          <input
            type="radio"
            name="coverage-range-priority"
            value="normal"
            checked={priority === 'normal'}
            onChange={() => setPriority('normal')}
          />
          Normal (must-fill)
        </label>
        <label>
          <input
            type="radio"
            name="coverage-range-priority"
            value="optional"
            checked={priority === 'optional'}
            onChange={() => setPriority('optional')}
          />
          Optional (nice-to-have)
        </label>
      </fieldset>

      {candidates.length > 0 && (
        <>
          <div className="request-coverage-range-form__select-all">
            <label>
              <input type="checkbox" checked={allChecked} onChange={toggleAll} aria-label="Select all" />
              Select all
            </label>
            <span className="request-coverage-range-form__count">
              {checkedKeys.size} selected
            </span>
          </div>
          {checkedKeys.size > MAX_BATCH_ITEMS && (
            <p className="request-coverage-range-form__warning">
              You can request coverage for at most {MAX_BATCH_ITEMS} shifts at once — uncheck {checkedKeys.size - MAX_BATCH_ITEMS} to continue.
            </p>
          )}
          <ul className="request-coverage-range-form__list">
            {candidates.map(o => {
              const key = occurrenceKey(o);
              const label = `${o.date} — ${formatSlotRangeLabel(dayOfWeekFromIso(o.date), o.hour)}`;
              return (
                <li key={key}>
                  <label>
                    <input
                      type="checkbox"
                      checked={checkedKeys.has(key)}
                      onChange={() => toggleOccurrence(key)}
                      aria-label={label}
                    />
                    {label}
                  </label>
                </li>
              );
            })}
          </ul>
        </>
      )}

      {startDate && endDate && !rangeTooLong && candidates.length === 0 && (
        <p>No recurring shifts fall in that range.</p>
      )}

      <div className="request-coverage-range-form__actions">
        <button type="button" className="request-coverage-range-form__cancel-btn" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button
          type="button"
          className="btn-primary"
          onClick={handleSubmit}
          disabled={submitting || rangeTooLong || checkedKeys.size === 0 || checkedKeys.size > MAX_BATCH_ITEMS}
        >
          {submitting ? 'Requesting…' : 'Request Coverage'}
        </button>
      </div>
    </div>
  );
};

export default RequestCoverageRangeForm;
