import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RequestCoverageRangeForm, { computeCandidateOccurrences } from './RequestCoverageRangeForm';
import Modal from '../../components/Modal';
import { scheduleApi } from '../../api/client';
import { weekParity } from './scheduleGrid';
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

  it('excludes a legacy slot whose hour is beyond maxHourFor its day of week', () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      // 2026-08-15 is a Saturday (weekend, maxHourFor = 15); hour 17 is a
      // legacy weekday-era slot that's now out of range on weekends.
      const slots: ScheduleSlot[] = [{ day_of_week: 6, hour: 17 }];
      const result = computeCandidateOccurrences(slots, '2026-08-11', '2026-08-20');
      expect(result).toEqual([]);
    } finally {
      vi.useRealTimers();
    }
  });

  it('only offers a biweekly_a slot on its own-parity weeks within the range', () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      // 2026-08-11, 2026-08-18 and 2026-08-25 are consecutive Tuesdays, so
      // they alternate parity; derive the expected "a" dates from
      // weekParity itself rather than hardcoding, so this test doesn't
      // silently rot if the reference date's assignment ever shifts.
      const candidateDates = ['2026-08-11', '2026-08-18', '2026-08-25'];
      const expectedDates = candidateDates.filter(d => weekParity(d) === 'a');
      expect(expectedDates.length).toBeGreaterThan(0);
      expect(expectedDates.length).toBeLessThan(candidateDates.length);

      const slots: ScheduleSlot[] = [{ day_of_week: 2, hour: 10, cadence: 'biweekly_a' }];
      const result = computeCandidateOccurrences(slots, '2026-08-11', '2026-08-25');
      expect(result).toEqual(expectedDates.map(date => ({ date, hour: 10 })));
    } finally {
      vi.useRealTimers();
    }
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

  it('pre-fills the start/end date fields from initialStartDate/initialEndDate, computing candidates immediately', async () => {
    // The Overview popover's "Request coverage" entry point pre-fills both
    // dates to the exact date the volunteer clicked, so they land on an
    // already-populated single-occurrence checklist instead of a blank form.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      render(<RequestCoverageRangeForm groupId={7} slots={slots} initialStartDate="2026-08-11" initialEndDate="2026-08-11" />);

      expect(screen.getByLabelText(/start date/i)).toHaveValue('2026-08-11');
      expect(screen.getByLabelText(/end date/i)).toHaveValue('2026-08-11');
      const checkboxes = await screen.findAllByRole('checkbox', { name: /2026-08-11/ });
      expect(checkboxes).toHaveLength(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows the 90-min start-end range for a terminal-hour occurrence, not just the start time', async () => {
    // 2026-08-11 is a Tuesday (a weekday); hour 17 is that day's terminal
    // (maxHourFor) slot, a 90-min shift ending 6:30 PM.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      const terminalSlots: ScheduleSlot[] = [{ day_of_week: 2, hour: 17 }];
      render(<RequestCoverageRangeForm groupId={7} slots={terminalSlots} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-11' } });

      expect(await screen.findByRole('checkbox', { name: /5:00–6:30 PM/ })).toBeInTheDocument();
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
        created: [{ id: 1, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 10, status: 'open', priority: 'normal', claimed_by_user_id: null }],
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
      ], 'normal'));
      expect(await screen.findByText(/requested coverage for 1 shift/i)).toBeInTheDocument();
      expect(screen.getByText(/1 skipped/i)).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('submits with priority "optional" when the Optional toggle is selected', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      const result: CoverageRequestBatchResult = {
        created: [{ id: 1, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 10, status: 'open', priority: 'normal', claimed_by_user_id: null }],
        skipped: [],
      };
      vi.mocked(scheduleApi.createCoverageRequestsBatch).mockResolvedValue({ data: result } as AxiosResponse<CoverageRequestBatchResult>);

      render(<RequestCoverageRangeForm groupId={7} slots={slots} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-11' } });
      await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });

      fireEvent.click(screen.getByRole('radio', { name: /optional/i }));
      fireEvent.click(screen.getByRole('button', { name: /request coverage/i }));

      await waitFor(() => expect(scheduleApi.createCoverageRequestsBatch).toHaveBeenCalledWith(7, [
        { date: '2026-08-11', hour: 10 },
      ], 'optional'));
    } finally {
      vi.useRealTimers();
    }
  });

  it('defaults the priority toggle to Normal (must-fill)', async () => {
    render(<RequestCoverageRangeForm groupId={7} slots={slots} />);
    expect(screen.getByRole('radio', { name: /normal/i })).toBeChecked();
    expect(screen.getByRole('radio', { name: /optional/i })).not.toBeChecked();
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
        created: [{ id: 1, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 10, status: 'open', priority: 'normal', claimed_by_user_id: null }],
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

  it('shows the selected count and blocks submit once the selection exceeds the batch cap', async () => {
    // Regression test: MAX_RANGE_DAYS (90 days) alone doesn't guarantee the
    // occurrence count stays under the backend's 200-item cap - a busy
    // recurring schedule over a wide-but-legal range can still exceed it.
    // Every weekday at every hour (8am-5pm = 10 hours x 7 days = 70 slots)
    // over a 30-day range produces ~300 occurrences, comfortably over 200.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      const busySlots: ScheduleSlot[] = [];
      for (let day = 0; day <= 6; day++) {
        for (let hour = 8; hour <= 17; hour++) {
          busySlots.push({ day_of_week: day, hour });
        }
      }

      render(<RequestCoverageRangeForm groupId={7} slots={busySlots} />);
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-09-09' } });

      await waitFor(() => expect(screen.getByText(/\d+ selected/i)).toBeInTheDocument());
      expect(screen.getByText(/^(2\d{2}|[3-9]\d{2}) selected$/i)).toBeInTheDocument();
      expect(screen.getByText(/at most 200 shifts/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /request coverage/i })).toBeDisabled();

      // Deselecting enough occurrences to drop back under the cap clears
      // the warning and re-enables submit.
      fireEvent.click(screen.getByRole('checkbox', { name: /select all/i }));
      expect(screen.getByText(/0 selected/i)).toBeInTheDocument();
      expect(screen.queryByText(/at most 200 shifts/i)).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('styles the Cancel button correctly even though Modal renders it via a portal outside .schedule-tab', () => {
    // Regression test: ScheduleTab.css's .schedule-tab .btn-secondary rule
    // is a descendant selector, but Modal renders its children via
    // createPortal(..., document.body) - so when this form is opened from
    // ScheduleTab (as it always is in the real app), the Cancel button ends
    // up as a DOM sibling of .schedule-tab, not a descendant of it, and
    // that scoped rule can never match it regardless of the React
    // component tree. The fix must not depend on DOM-tree ancestry, so
    // this test renders through the real Modal - inside a real
    // .schedule-tab wrapper, matching how ScheduleTab.tsx actually uses
    // it - to prove the button is styled correctly wherever the portal
    // happens to place it.
    render(
      <div className="schedule-tab">
        <Modal isOpen={true} onClose={() => {}} title="Request Coverage for a Date Range">
          <RequestCoverageRangeForm groupId={7} slots={slots} onCancel={() => {}} />
        </Modal>
      </div>
    );

    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    expect(cancelButton).toHaveClass('request-coverage-range-form__cancel-btn');

    // Confirms the portal actually did what this test exists to guard
    // against: the button is NOT inside .schedule-tab in the DOM, so any
    // fix relying on a `.schedule-tab .btn-secondary`-style descendant
    // selector would silently fail to reach it.
    expect(cancelButton.closest('.schedule-tab')).toBeNull();
  });
});
