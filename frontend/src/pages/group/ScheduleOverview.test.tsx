import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ScheduleOverview from './ScheduleOverview';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { ScheduleOverviewResponse, CoverageRequest } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getOverview: vi.fn(),
    claimCoverageRequest: vi.fn(),
    createCoverageRequest: vi.fn(),
    cancelCoverageRequest: vi.fn(),
  },
}));

const mockShowSuccess = vi.fn();
const mockShowError = vi.fn();
vi.mock('../../hooks/useToast', () => ({
  useToast: () => ({ showSuccess: mockShowSuccess, showError: mockShowError }),
}));

function mockOverview(data: ScheduleOverviewResponse) {
  vi.mocked(scheduleApi.getOverview).mockResolvedValue({ data } as unknown as AxiosResponse<ScheduleOverviewResponse>);
}

describe('ScheduleOverview', () => {
  beforeEach(() => {
    mockOverview({ week_start: '2026-08-09', slots: [] });
    mockShowSuccess.mockClear();
    mockShowError.mockClear();
  });

  it('loads the overview for the given group and week on mount', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);
    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalledWith(7, expect.objectContaining({ signal: expect.any(AbortSignal) })));
  });

  it('styles the week navigation arrows as compact icon buttons', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);
    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalled());

    expect(screen.getByRole('button', { name: 'Previous week' })).toHaveClass('schedule-overview__week-nav-btn');
    expect(screen.getByRole('button', { name: 'Next week' })).toHaveClass('schedule-overview__week-nav-btn');
  });

  it('renders a tier class proportional to how many members are available', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 1, username: 'vol1', status: 'normal' },
          { user_id: 2, username: 'vol2', status: 'normal' },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*2 available/i });
    // 2 of 4 members = 50%, falls in the 26-50% tier.
    expect(cell).toHaveClass('schedule-grid__slot--tier-2');
  });

  it('a cell with nobody available is not clickable and has the zero tier', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);
    const cell = await screen.findByRole('cell', { name: 'Sun 8:00 AM, 0 available' });
    expect(cell).toHaveClass('schedule-grid__slot--tier-0');
    expect(cell.tagName).not.toBe('BUTTON');
  });

  it('clicking a non-empty cell opens a popover listing member names', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol1', first_name: 'Jane', last_name: 'Doe', status: 'normal' },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument();
  });

  it('falls back to username when first/last name are blank', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [{ date: '2026-08-11', day_of_week: 2, hour: 9, members: [{ user_id: 2, username: 'vol1', status: 'normal' }] }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('vol1')).toBeInTheDocument();
  });

  it('falls back to a data-derived denominator when totalMembers is 0 but slots have members', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 1, username: 'vol1', status: 'normal' },
          { user_id: 2, username: 'vol2', status: 'normal' },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={0} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*2 available/i });
    // totalMembers is 0 (unreliable), so the fallback denominator is derived
    // from the fetched data: max slot size of 2, so 2/2 = 100% = tier-4.
    expect(cell).not.toHaveClass('schedule-grid__slot--tier-0');
    expect(cell).toHaveClass('schedule-grid__slot--tier-4');
  });

  it('clicking outside the popover closes it', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [{ date: '2026-08-11', day_of_week: 2, hour: 9, members: [{ user_id: 2, username: 'vol1', status: 'normal' }] }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);
    expect(await screen.findByText('vol1')).toBeInTheDocument();

    fireEvent.mouseDown(document.body);
    await waitFor(() => expect(screen.queryByText('vol1')).not.toBeInTheDocument());
  });

  it('renders every member in a long list even though the popover box is height-capped', async () => {
    // jsdom doesn't compute layout, so this can't verify the CSS
    // max-height/overflow-y clamp actually kicks in visually (that's
    // covered by the Playwright verification instead) - but it does
    // confirm the popover keeps rendering all members in the DOM (for
    // scrolling to reveal) rather than e.g. truncating the list to fit.
    const members = Array.from({ length: 20 }, (_, i) => ({
      user_id: i + 2,
      username: `vol${i + 1}`,
      first_name: `Member`,
      last_name: `${i + 1}`,
      status: 'normal' as const,
    }));
    mockOverview({
      week_start: '2026-08-09',
      slots: [{ date: '2026-08-11', day_of_week: 2, hour: 9, members }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={20} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*20 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('Member 1')).toBeInTheDocument();
    expect(screen.getByText('Member 20')).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(20);
  });

  it('flags a cell with an open coverage request as needing coverage', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', coverage_request_id: 42, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    expect(cell).toHaveClass('schedule-grid__slot--needs-coverage');
  });

  it('shows a Claim button for another member\'s open request and claims it on click', async () => {
    vi.mocked(scheduleApi.claimCoverageRequest).mockResolvedValue({ data: {} as CoverageRequest } as AxiosResponse<CoverageRequest>);
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', coverage_request_id: 42, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);
    const claimButton = await screen.findByRole('button', { name: /claim/i });
    fireEvent.click(claimButton);

    await waitFor(() => expect(scheduleApi.claimCoverageRequest).toHaveBeenCalledWith(7, 42));
  });

  it('shows a Request coverage button next to the current user\'s own name on a future date', async () => {
    // The component only renders the currently-displayed week by default
    // (defaulting to the real current week), and "future" is checked
    // against the real clock - so a hardcoded fixture date is only
    // future-and-in-the-rendered-week on some days of the year, and would
    // start failing permanently once real "today" passes it (or even
    // sooner, on any day where Tuesday of the current week has already
    // passed). Pin the clock instead: only `Date` is faked here (not
    // `setTimeout`/timers), so `findByRole`/`waitFor`'s internal real-timer
    // polling keeps working, while the component's `currentWeekStart()` and
    // "is this date in the future" check both resolve against this fixed
    // Monday - making 2026-08-11 (that Monday's Tuesday) deterministically
    // future, forever.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z')); // a Monday
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 1, username: 'me', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      expect(await screen.findByRole('button', { name: /request coverage/i })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('disables the Request coverage button while its own POST is in flight, so a double-click cannot fire two requests', async () => {
    // Backend fix: a partial unique index backstops the DB against a
    // duplicate-request race, but the friendly first line of defense is the
    // UI never firing a second POST in the first place - mirroring how
    // handleClaim already guards via busyRequestId. Resolve the mocked
    // createCoverageRequest call manually so we can assert the disabled
    // state while the first request is still pending.
    let resolveCreate: (() => void) | undefined;
    vi.mocked(scheduleApi.createCoverageRequest).mockReturnValue(
      new Promise((resolve) => {
        resolveCreate = () => resolve({ data: {} as CoverageRequest } as AxiosResponse<CoverageRequest>);
      })
    );
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z')); // a Monday
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 1, username: 'me', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const requestButton = await screen.findByRole('button', { name: /request coverage/i });

      fireEvent.click(requestButton);
      fireEvent.click(requestButton);

      expect(scheduleApi.createCoverageRequest).toHaveBeenCalledTimes(1);
      expect(requestButton).toBeDisabled();

      // Flush the pending promise and its .then/.finally chain (which
      // closes the popover) so no state update leaks past this test.
      resolveCreate?.();
      await waitFor(() => expect(requestButton).not.toBeInTheDocument());
    } finally {
      vi.useRealTimers();
    }
  });

  it('renders the Claim button disabled with a tooltip for a conflicting request, instead of hiding it', async () => {
    // The backend sets claimable = !conflict, so claimable and conflict are
    // never both true for a needs_coverage entry that isn't the viewer's
    // own - meaning a render condition gated on `member.claimable` can never
    // reach the button's own conflict-disabled/tooltip branch. That left a
    // member with a real scheduling conflict seeing no button and no
    // explanation at all. The fix drops the claimable gate from visibility
    // and lets `conflict` continue to drive disabled/tooltip.
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', coverage_request_id: 42, claimable: false, conflict: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);

    const claimButton = await screen.findByRole('button', { name: /claim/i });
    expect(claimButton).toBeDisabled();
    expect(claimButton).toHaveAttribute('title', 'You already have a conflicting shift at this time');
  });

  it('shows a Cancel request button on the caller\'s own open request and cancels it on click', async () => {
    // The backend CancelCoverageRequest endpoint and scheduleApi client
    // method were already fully tested, but nothing in the UI called them -
    // once a volunteer requested coverage there was no way to undo it. This
    // covers the minimum viable UI path: the requester's own open
    // (needs_coverage) row gets a Cancel request button.
    vi.mocked(scheduleApi.cancelCoverageRequest).mockResolvedValue({ data: {} as CoverageRequest } as AxiosResponse<CoverageRequest>);
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 1, username: 'me', status: 'needs_coverage', coverage_request_id: 99 },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);
    const cancelButton = await screen.findByRole('button', { name: /cancel request/i });
    fireEvent.click(cancelButton);

    await waitFor(() => expect(scheduleApi.cancelCoverageRequest).toHaveBeenCalledWith(7, 99));
  });

  it('shows a Request coverage button for the current user\'s own name on the same day (today counts as future)', async () => {
    // Backend fix: CreateCoverageRequest now allows same-day-or-later
    // (date.Before(today) rejects only the past), matching the product
    // decision that requesting coverage for later today is a normal use
    // case. The frontend's button-visibility condition changed from
    // `date > today` to `date >= today` to match - this pins the clock to
    // exactly the slot's own date/time so `date === today` is exercised.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-11T12:00:00Z')); // the Tuesday itself
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 1, username: 'me', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      expect(await screen.findByRole('button', { name: /request coverage/i })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('disables every Claim button in the popover while any claim is in flight, not just the one clicked', async () => {
    // Backend hardening: idx_coverage_request_claimed_unique now backstops
    // the DB against the same user claiming two different requests at the
    // same (date, hour), but the UI previously only disabled the specific
    // button clicked (busyRequestId === member.coverage_request_id), so a
    // second Claim button in the same popover stayed clickable while the
    // first claim was still in flight.
    let resolveClaim: (() => void) | undefined;
    vi.mocked(scheduleApi.claimCoverageRequest).mockReturnValue(
      new Promise((resolve) => {
        resolveClaim = () => resolve({ data: {} as CoverageRequest } as AxiosResponse<CoverageRequest>);
      })
    );
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', coverage_request_id: 42, claimable: true },
          { user_id: 3, username: 'vol3', status: 'needs_coverage', coverage_request_id: 43, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);
    const claimButtons = await screen.findAllByRole('button', { name: /claim/i });
    expect(claimButtons).toHaveLength(2);

    const overviewCallsBefore = vi.mocked(scheduleApi.getOverview).mock.calls.length;
    fireEvent.click(claimButtons[0]);

    expect(claimButtons[0]).toBeDisabled();
    expect(claimButtons[1]).toBeDisabled();

    resolveClaim?.();
    await waitFor(() => expect(vi.mocked(scheduleApi.getOverview).mock.calls.length).toBe(overviewCallsBefore + 1));
  });

  it('surfaces the backend error message when a claim fails, and reloads the overview', async () => {
    // Established codebase convention (see GroupPage.tsx, Settings.tsx):
    // surface err.response.data.error over a generic fallback string, and
    // reload data on failure so the UI doesn't show a stale state (e.g. a
    // claim that silently failed but still looks available).
    vi.mocked(scheduleApi.claimCoverageRequest).mockRejectedValue({
      response: { data: { error: 'Request is no longer open' } },
    });
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', coverage_request_id: 42, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);
    const claimButton = await screen.findByRole('button', { name: /claim/i });
    const overviewCallsBefore = vi.mocked(scheduleApi.getOverview).mock.calls.length;
    fireEvent.click(claimButton);

    await waitFor(() => expect(mockShowError).toHaveBeenCalledWith('Request is no longer open'));
    // The catch block calls loadOverview() so a failed claim doesn't leave
    // stale UI state (e.g. a button that stays disabled-looking).
    await waitFor(() => expect(vi.mocked(scheduleApi.getOverview).mock.calls.length).toBe(overviewCallsBefore + 1));
  });

  it('does not show a Claim button on the caller\'s own needs_coverage row', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 1, username: 'me', status: 'needs_coverage', coverage_request_id: 99 },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    fireEvent.click(cell);
    await screen.findByRole('button', { name: /cancel request/i });

    expect(screen.queryByRole('button', { name: /^claim$/i })).not.toBeInTheDocument();
  });
});
