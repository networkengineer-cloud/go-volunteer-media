import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RequestCoverageRangeForm, { computeCandidateOccurrences } from './RequestCoverageRangeForm';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { CoverageRequestBatchResult, ScheduleSlot } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    createCoverageRequestsBatch: vi.fn(),
  },
}));

vi.mock('../../hooks/useToast', () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}));

describe('computeCandidateOccurrences', () => {
  it('finds every occurrence of the recurring slots within the range', () => {
    // Pin "now" safely before the fixture range so the past-date exclusion
    // in computeCandidateOccurrences never drops 2026-08-11 depending on
    // the real wall clock (matches the vi.useFakeTimers({toFake: ['Date']})
    // + try/finally convention used in ScheduleOverview.test.tsx).
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      // 2026-08-11 is a Tuesday; 2026-08-13 is a Thursday.
      const slots: ScheduleSlot[] = [
        { day_of_week: 2, hour: 10 }, // Tuesday 10am
        { day_of_week: 4, hour: 14 }, // Thursday 2pm
      ];
      const result = computeCandidateOccurrences(slots, '2026-08-11', '2026-08-20');
      // Expect Tuesdays 8/11 and 8/18, Thursdays 8/13 and 8/20.
      expect(result).toEqual([
        { date: '2026-08-11', hour: 10 },
        { date: '2026-08-13', hour: 14 },
        { date: '2026-08-18', hour: 10 },
        { date: '2026-08-20', hour: 14 },
      ]);
    } finally {
      vi.useRealTimers();
    }
  });

  it('excludes dates before today even if within the given range', () => {
    const slots: ScheduleSlot[] = [{ day_of_week: 2, hour: 10 }];
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
    const result = computeCandidateOccurrences(slots, yesterday, yesterday);
    expect(result).toEqual([]);
  });

  it('returns an empty list when start is after end', () => {
    const slots: ScheduleSlot[] = [{ day_of_week: 2, hour: 10 }];
    expect(computeCandidateOccurrences(slots, '2026-08-20', '2026-08-11')).toEqual([]);
  });
});

describe('RequestCoverageRangeForm', () => {
  const slots: ScheduleSlot[] = [{ day_of_week: 2, hour: 10 }];

  beforeEach(() => {
    vi.mocked(scheduleApi.createCoverageRequestsBatch).mockReset();
  });

  it('pre-checks every computed occurrence and lets select-all toggle them', async () => {
    // Same pinning rationale as the computeCandidateOccurrences test above:
    // this test's expected checkbox count (2) depends on 2026-08-11 not
    // being excluded by the component's own past-date filtering, which
    // reads the real wall clock unless pinned.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      render(<RequestCoverageRangeForm groupId={7} slots={slots} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-18' } });

      const checkboxes = await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });
      expect(checkboxes).toHaveLength(2);
      checkboxes.forEach(cb => expect(cb).toBeChecked());

      fireEvent.click(screen.getByRole('checkbox', { name: /select all/i }));
      checkboxes.forEach(cb => expect(cb).not.toBeChecked());
    } finally {
      vi.useRealTimers();
    }
  });

  it('submits only the checked occurrences and shows the created/skipped summary', async () => {
    // Pinned for the same reason: the submitted payload asserted below
    // includes 2026-08-11, which only survives the component's past-date
    // filtering when "today" is pinned on or before that date.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      const result: CoverageRequestBatchResult = {
        created: [{ id: 1, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 10, status: 'open', claimed_by_user_id: null }],
        skipped: [{ date: '2026-08-18', hour: 10, reason: 'a coverage request already exists for that date and hour' }],
      };
      vi.mocked(scheduleApi.createCoverageRequestsBatch).mockResolvedValue({ data: result } as AxiosResponse<CoverageRequestBatchResult>);

      render(<RequestCoverageRangeForm groupId={7} slots={slots} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-18' } });
      await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });

      fireEvent.click(screen.getByRole('button', { name: /request coverage/i }));

      await waitFor(() => expect(scheduleApi.createCoverageRequestsBatch).toHaveBeenCalledWith(7, [
        { date: '2026-08-11', hour: 10 },
        { date: '2026-08-18', hour: 10 },
      ]));
      expect(await screen.findByText(/requested coverage for 1 shift/i)).toBeInTheDocument();
      expect(screen.getByText(/1 skipped/i)).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows the result screen even when a consumer passes onSuccess, and Done calls onCancel not onSuccess', async () => {
    // Regression test for the onSuccess/result-screen race: ScheduleTab used
    // to pass an onSuccess that closed its modal, and because React batches
    // setResult(...) and onSuccess?.() from the same submit handler, the
    // parent's onSuccess-driven unmount raced the result screen's paint -
    // in practice the modal closed before the created/skipped summary (and
    // its "Done" button) ever became visible. ScheduleTab has since stopped
    // passing onSuccess at all (relying solely on onCancel to close), but
    // this component still accepts onSuccess as an optional prop for other
    // future consumers, so it must keep behaving correctly when one is
    // supplied: the result screen must actually render (not be skipped or
    // torn down by onSuccess having fired), and re-reading the component's
    // current code confirms the "Done" button's onClick invokes onCancel,
    // not onSuccess - so clicking it must not fire onSuccess again.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      const result: CoverageRequestBatchResult = {
        created: [{ id: 1, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 10, status: 'open', claimed_by_user_id: null }],
        skipped: [],
      };
      vi.mocked(scheduleApi.createCoverageRequestsBatch).mockResolvedValue({ data: result } as AxiosResponse<CoverageRequestBatchResult>);

      const onSuccess = vi.fn();
      const onCancel = vi.fn();
      render(<RequestCoverageRangeForm groupId={7} slots={slots} onSuccess={onSuccess} onCancel={onCancel} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-11' } });
      await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });

      fireEvent.click(screen.getByRole('button', { name: /request coverage/i }));

      // The result screen must actually be visible - this is the part that
      // was previously dead code in the real app because ScheduleTab's old
      // onSuccess handler unmounted the form (via closing the Modal) before
      // this ever painted.
      expect(await screen.findByText(/requested coverage for 1 shift/i)).toBeInTheDocument();
      expect(onSuccess).toHaveBeenCalledTimes(1);
      expect(onCancel).not.toHaveBeenCalled();

      fireEvent.click(screen.getByRole('button', { name: /done/i }));
      expect(onCancel).toHaveBeenCalledTimes(1);
      // Done must not invoke onSuccess again - it only calls onCancel.
      expect(onSuccess).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });
});
