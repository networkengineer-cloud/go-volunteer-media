import React, { useEffect, useMemo, useState } from 'react';
import { scheduleApi } from '../../api/client';
import type { ScheduleSlot, CoverageRequestBatchItem, CoverageRequestBatchResult, CoverageRequestPriority } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { useShelterTimeZone } from '../../hooks/useShelterTimeZone';
import { todayInZone } from '../../utils/shelterTime';
import DateRangePicker from '../../components/DateRangePicker';
import { formatSlotRangeLabel, formatDateLabel } from './scheduleGrid';
import { computeCandidateOccurrences, type Occurrence } from './coverageOccurrences';
import './RequestCoverageRangeForm.css';

export interface RequestCoverageRangeFormProps {
  groupId: number;
  slots: ScheduleSlot[];
  initialStartDate?: string;
  initialEndDate?: string;
  // When set, only occurrences at these hours start checked; everything else
  // in range is offered but unticked. The schedule popover uses it so that
  // clicking one cell still SHOWS the viewer's other shifts that day - which
  // is what lets a run of consecutive hours go out as one request - without
  // silently widening a one-cell click into a multi-shift submission.
  // Omitted (the date-range entry point) means check everything, which is
  // what "I'm away all next week" wants.
  initialCheckedHours?: number[];
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

function occurrenceKey(o: Occurrence): string {
  return `${o.date}-${o.hour}`;
}

// Occurrences/skipped items only carry a `date` (not a day_of_week), so the
// day-of-week needed by formatSlotRangeLabel (to know whether this item's
// hour is a day's terminal 90-min slot) is derived here from that date.
function dayOfWeekFromIso(isoDate: string): number {
  const [y, m, d] = isoDate.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d)).getUTCDay();
}

function rangeExceedsMaxDays(startDate: string, endDate: string): boolean {
  if (!startDate || !endDate) return false;
  const [sy, sm, sd] = startDate.split('-').map(Number);
  const [ey, em, ed] = endDate.split('-').map(Number);
  const start = Date.UTC(sy, sm - 1, sd);
  const end = Date.UTC(ey, em - 1, ed);
  return (end - start) / (1000 * 60 * 60 * 24) > MAX_RANGE_DAYS;
}

const RequestCoverageRangeForm: React.FC<RequestCoverageRangeFormProps> = ({ groupId, slots, initialStartDate, initialEndDate, initialCheckedHours, onSuccess, onCancel }) => {
  const toast = useToast();
  const today = todayInZone(useShelterTimeZone());
  const [startDate, setStartDate] = useState(initialStartDate ?? '');
  const [endDate, setEndDate] = useState(initialEndDate ?? '');
  const [checkedKeys, setCheckedKeys] = useState<Set<string>>(new Set());
  const [priorities, setPriorities] = useState<Map<string, CoverageRequestPriority>>(new Map());
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<CoverageRequestBatchResult | null>(null);

  const rangeTooLong = rangeExceedsMaxDays(startDate, endDate);
  const candidates = useMemo(
    () => (rangeTooLong ? [] : computeCandidateOccurrences(slots, startDate, endDate, today)),
    [slots, startDate, endDate, rangeTooLong, today]
  );

  // Re-derive which occurrences are checked whenever the candidate list
  // itself changes (start/end date edited, or the range became too long) -
  // a side effect, so it belongs in useEffect, not inside the useMemo above.
  // Stable across renders so the effect below doesn't re-run on every one
  // just because a fresh array literal was passed in.
  const checkedHoursKey = initialCheckedHours ? initialCheckedHours.join(',') : '';
  useEffect(() => {
    const only = checkedHoursKey === '' ? null : new Set(checkedHoursKey.split(',').map(Number));
    setCheckedKeys(new Set(
      candidates.filter(o => only === null || only.has(o.hour)).map(occurrenceKey)
    ));
    setPriorities(new Map(candidates.map(o => [occurrenceKey(o), 'normal' as CoverageRequestPriority])));
  }, [candidates, checkedHoursKey]);

  const setPriorityFor = (key: string, value: CoverageRequestPriority) => {
    setPriorities(prev => new Map(prev).set(key, value));
  };

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
  // From the schedule popover the range is pinned to one date, so "select
  // all" really means "the rest of my shifts that day" - worth saying, since
  // only the clicked hour starts ticked and the neighbours are easy to miss.
  // The date-range entry point can span weeks, where the generic label is
  // the honest one.
  const selectAllLabel = startDate !== '' && startDate === endDate && candidates.length > 1
    ? `Select all ${candidates.length} shifts on ${formatDateLabel(startDate)}`
    : 'Select all';
  const toggleAll = () => {
    setCheckedKeys(allChecked ? new Set() : new Set(candidates.map(occurrenceKey)));
  };

  const handleSubmit = () => {
    const requests: CoverageRequestBatchItem[] = candidates
      .filter(o => checkedKeys.has(occurrenceKey(o)))
      .map(o => ({ date: o.date, hour: o.hour, priority: priorities.get(occurrenceKey(o)) ?? 'normal' }));
    if (requests.length === 0) {
      toast.showError('Select at least one shift.');
      return;
    }
    if (requests.length > MAX_BATCH_ITEMS) {
      toast.showError(`You can request coverage for at most ${MAX_BATCH_ITEMS} shifts at once.`);
      return;
    }
    setSubmitting(true);
    scheduleApi.createCoverageRequestsBatch(groupId, requests)
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
      <DateRangePicker
        startDate={startDate}
        endDate={endDate}
        min={today}
        onChange={(start, end) => {
          setStartDate(start);
          setEndDate(end);
        }}
      />

      {rangeTooLong && (
        <p className="request-coverage-range-form__warning">Please choose a range of {MAX_RANGE_DAYS} days or fewer.</p>
      )}

      {candidates.length > 0 && (
        <>
          <div className="request-coverage-range-form__select-all">
            <label>
              <input type="checkbox" checked={allChecked} onChange={toggleAll} aria-label={selectAllLabel} />
              {selectAllLabel}
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
              const itemPriority = priorities.get(key) ?? 'normal';
              return (
                <li key={key} className="request-coverage-range-form__list-item">
                  <label>
                    <input
                      type="checkbox"
                      checked={checkedKeys.has(key)}
                      onChange={() => toggleOccurrence(key)}
                      aria-label={label}
                    />
                    {label}
                  </label>
                  <label className="request-coverage-range-form__item-priority">
                    <input
                      type="checkbox"
                      checked={itemPriority === 'optional'}
                      onChange={e => setPriorityFor(key, e.target.checked ? 'optional' : 'normal')}
                    />
                    Optional
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
          className="request-coverage-range-form__submit-btn"
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
