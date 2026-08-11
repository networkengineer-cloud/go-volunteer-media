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
    render(<RequestCoverageRangeForm groupId={7} slots={slots} />);
    fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
    fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-18' } });

    const checkboxes = await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });
    expect(checkboxes).toHaveLength(2);
    checkboxes.forEach(cb => expect(cb).toBeChecked());

    fireEvent.click(screen.getByRole('checkbox', { name: /select all/i }));
    checkboxes.forEach(cb => expect(cb).not.toBeChecked());
  });

  it('submits only the checked occurrences and shows the created/skipped summary', async () => {
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
  });
});
