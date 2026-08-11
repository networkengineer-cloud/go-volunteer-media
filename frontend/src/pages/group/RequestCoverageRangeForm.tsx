import React, { useEffect, useMemo, useState } from 'react';
import { scheduleApi } from '../../api/client';
import type { ScheduleSlot, CoverageRequestBatchItem, CoverageRequestBatchResult } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { formatHourLabel } from './scheduleGrid';
import './RequestCoverageRangeForm.css';

export interface RequestCoverageRangeFormProps {
  groupId: number;
  slots: ScheduleSlot[];
  onSuccess?: () => void;
  onCancel?: () => void;
}

const MAX_RANGE_DAYS = 90;

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

// computeCandidateOccurrences finds every date within [startDate, endDate]
// (inclusive, both YYYY-MM-DD) whose weekday matches one of the given
// slots, paired with that slot's hour - i.e. every real calendar
// occurrence of the user's recurring pattern within the range. Excludes
// dates before today, mirroring the backend's own past-date rejection so
// the checklist never pre-checks something that would just get skipped
// server-side.
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
      if (slot.day_of_week === dayOfWeek) {
        occurrences.push({ date: dateStr, hour: slot.hour });
      }
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

const RequestCoverageRangeForm: React.FC<RequestCoverageRangeFormProps> = ({ groupId, slots, onSuccess, onCancel }) => {
  const toast = useToast();
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [checkedKeys, setCheckedKeys] = useState<Set<string>>(new Set());
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
                  {s.date} at {formatHourLabel(s.hour)} — {s.reason}
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

      {candidates.length > 0 && (
        <>
          <div className="request-coverage-range-form__select-all">
            <label>
              <input type="checkbox" checked={allChecked} onChange={toggleAll} aria-label="Select all" />
              Select all
            </label>
          </div>
          <ul className="request-coverage-range-form__list">
            {candidates.map(o => {
              const key = occurrenceKey(o);
              const label = `${o.date} — ${formatHourLabel(o.hour)}`;
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
        <button type="button" className="btn-secondary" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button
          type="button"
          className="btn-primary"
          onClick={handleSubmit}
          disabled={submitting || rangeTooLong || checkedKeys.size === 0}
        >
          {submitting ? 'Requesting…' : 'Request Coverage'}
        </button>
      </div>
    </div>
  );
};

export default RequestCoverageRangeForm;
