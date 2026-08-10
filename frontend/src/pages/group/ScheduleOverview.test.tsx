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

vi.mock('../../hooks/useToast', () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}));

function mockOverview(data: ScheduleOverviewResponse) {
  vi.mocked(scheduleApi.getOverview).mockResolvedValue({ data } as unknown as AxiosResponse<ScheduleOverviewResponse>);
}

describe('ScheduleOverview', () => {
  beforeEach(() => {
    mockOverview({ week_start: '2026-08-09', slots: [] });
  });

  it('loads the overview for the given group and week on mount', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);
    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalledWith(7, expect.objectContaining({ signal: expect.any(AbortSignal) })));
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
