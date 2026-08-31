import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import ScheduleOverview from './ScheduleOverview';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { ScheduleOverviewResponse, CoverageRequest, GroupMember, ReassignShiftsBatchResult } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getOverview: vi.fn(),
    claimCoverageRequest: vi.fn(),
    cancelCoverageRequest: vi.fn(),
    createCoverageRequestsBatch: vi.fn(),
    reassignShiftsBatch: vi.fn(),
  },
}));

const testMembers: GroupMember[] = [
  { user_id: 1, username: 'me', first_name: 'Mia', last_name: 'Example', email: '', is_group_admin: false, is_site_admin: false, skill_tags: [] },
  { user_id: 2, username: 'vol2', first_name: 'Vic', last_name: 'Two', email: '', is_group_admin: false, is_site_admin: false, skill_tags: [] },
  { user_id: 3, username: 'vol3', first_name: 'Val', last_name: 'Three', email: '', is_group_admin: false, is_site_admin: false, skill_tags: [] },
];

// DateRangePicker's own calendar-click mechanics are covered by its own
// test suite; standing in a plain two-input control here (matching
// RequestCoverageRangeForm.test.tsx's convention) keeps this file's
// existing getByLabelText(/start date/i)/(/end date/i) queries working.
vi.mock('../../components/DateRangePicker', () => ({
  default: ({ startDate, endDate, onChange }: { startDate: string; endDate: string; onChange: (s: string, e: string) => void }) => (
    <div>
      <label htmlFor="mock-start-date">Start date</label>
      <input id="mock-start-date" value={startDate} onChange={e => onChange(e.target.value, endDate)} />
      <label htmlFor="mock-end-date">End date</label>
      <input id="mock-end-date" value={endDate} onChange={e => onChange(startDate, e.target.value)} />
    </div>
  ),
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

  it('shows a legend with the currently-viewed week highlighted', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2024-01-10T12:00:00Z')); // an "A" week
    try {
      vi.mocked(scheduleApi.getOverview).mockResolvedValue({
        data: { week_start: '2024-01-07', slots: [] },
      } as unknown as AxiosResponse<ScheduleOverviewResponse>);

      render(<ScheduleOverview groupId={1} totalMembers={1} currentUserId={1} />);

      await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalled());
      expect(screen.getByText(/Week A:/).closest('.cadence-legend__entry')).toHaveClass('cadence-legend__entry--current');
    } finally {
      vi.useRealTimers();
    }
  });

  it('tags a biweekly member\'s name in a cell', async () => {
    vi.mocked(scheduleApi.getOverview).mockResolvedValue({
      data: {
        week_start: '2024-01-07',
        slots: [{
          date: '2024-01-09',
          day_of_week: 2,
          hour: 10,
          // last_name deliberately avoids the letter "A" so the assertion
          // below can't accidentally pass off the member's own name -
          // it must come from the dedicated cadence-tag element.
          members: [{ user_id: 5, username: 'alice', first_name: 'Alice', last_name: 'Lee', cadence: 'biweekly_a', status: 'normal' }],
        }],
      },
    } as unknown as AxiosResponse<ScheduleOverviewResponse>);

    render(<ScheduleOverview groupId={1} totalMembers={1} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 10:00 AM, 1 available/ });
    expect(within(cell).getByText('A', { selector: '.schedule-grid__cadence-tag' })).toBeInTheDocument();
  });

  it('does not tag a member with weekly cadence', async () => {
    vi.mocked(scheduleApi.getOverview).mockResolvedValue({
      data: {
        week_start: '2024-01-07',
        slots: [{
          date: '2024-01-09',
          day_of_week: 2,
          hour: 10,
          members: [{ user_id: 6, username: 'bob', first_name: 'Bob', last_name: 'Ray', cadence: 'weekly', status: 'normal' }],
        }],
      },
    } as unknown as AxiosResponse<ScheduleOverviewResponse>);

    render(<ScheduleOverview groupId={1} totalMembers={1} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 10:00 AM, 1 available/ });
    expect(within(cell).queryByText('A', { selector: '.schedule-grid__cadence-tag' })).not.toBeInTheDocument();
    expect(within(cell).queryByText('B', { selector: '.schedule-grid__cadence-tag' })).not.toBeInTheDocument();
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

  it('renders weekend hour 16/17 as non-interactive disabled cells, distinct from an ordinary unstaffed tier-0 cell', async () => {
    // Weekend hours 16/17 aren't valid shift slots at all (see maxHourFor in
    // scheduleGrid.ts) - previously they rendered as ordinary "0 available"
    // tier-0 cells, indistinguishable from a valid-but-unstaffed hour.
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);
    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalled());

    const table = screen.getByRole('table', { name: /weekly availability overview/i });
    const rows = within(table).getAllByRole('row');
    // First row is the header row; the rest are the 10 hour rows
    // (8,9,10,11,12,13,14,15,16,17) - hour 16 is row index 8, hour 17 is 9.
    const hour16Row = rows[1 + 8];
    const hour17Row = rows[1 + 9];

    // Sunday (dayOfWeek 0) is the first day column after the row header.
    const sundayHour16Cell = within(hour16Row).getAllByRole('cell')[0];
    const sundayHour17Cell = within(hour17Row).getAllByRole('cell')[0];

    for (const cell of [sundayHour16Cell, sundayHour17Cell]) {
      expect(cell).toHaveAttribute('aria-disabled', 'true');
      expect(cell).toHaveClass('schedule-grid__slot--disabled');
      // No "X available" count and no popover trigger - it's a plain,
      // non-interactive element, not a button.
      expect(cell.tagName).not.toBe('BUTTON');
      expect(cell).not.toHaveAttribute('aria-label');
      fireEvent.click(cell);
      expect(screen.queryByRole('list')).not.toBeInTheDocument();
    }

    // Every hour row (weekday or weekend) still exposes the same number of
    // role="cell" elements (7 days), matching ScheduleTab's ARIA-table fix.
    const hourRows = rows.slice(1);
    const cellCounts = hourRows.map(row => within(row).getAllByRole('cell').length);
    expect(cellCounts).toEqual(cellCounts.map(() => 7));
  });

  it("uses the 90-min range in a terminal-hour cell's aria-label, and the shared row header shows the same real range", async () => {
    mockOverview({
      week_start: '2026-08-09',
      // 2026-08-09 is a Sunday; day_of_week 2 (Tuesday) hour 17 is a
      // weekday terminal (90-min) slot: 5:00-6:30pm.
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 17, members: [
          { user_id: 1, username: 'vol1', status: 'normal' },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    // The per-cell aria-label reflects the 90-min range for the terminal row...
    expect(await screen.findByRole('cell', { name: 'Tue 5:00–6:30 PM, 1 available' })).toBeInTheDocument();

    // ...and the shared row header (same row for all 7 day columns) now
    // shows that same real range too, matching ScheduleTab's Individual
    // grid (rowHeaderFor) instead of a bare start time - no weekend cell
    // exists at hour 17 to be misled by it.
    const table = screen.getByRole('table', { name: /weekly availability overview/i });
    expect(within(table).getByRole('rowheader', { name: '5:00–6:30 PM' })).toBeInTheDocument();
  });

  it('flags the shared hour where weekend terminates but weekday continues, same as the Individual grid', async () => {
    mockOverview({ week_start: '2026-08-09', slots: [] });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const table = await screen.findByRole('table', { name: /weekly availability overview/i });
    const sharedRowHeader = within(table).getByRole('rowheader', { name: /3:00–4:00 PM/ });
    expect(within(sharedRowHeader).getByText('Weekends end 4:30 PM')).toBeInTheDocument();
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
    // The cell shows who's scheduled directly, without needing a click.
    expect(within(cell).getByText('vol1')).toBeInTheDocument();

    fireEvent.click(cell);

    // The popover repeats the same fallback name alongside its actions.
    const popover = await screen.findByRole('list');
    expect(within(popover).getByText('vol1')).toBeInTheDocument();
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
    const popover = await screen.findByRole('list');
    expect(within(popover).getByText('vol1')).toBeInTheDocument();

    fireEvent.mouseDown(document.body);
    await waitFor(() => expect(screen.queryByRole('list')).not.toBeInTheDocument());
  });

  it('caps the names shown directly in a cell and collapses the rest into a "+N more" indicator', async () => {
    const members = Array.from({ length: 5 }, (_, i) => ({
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
    render(<ScheduleOverview groupId={7} totalMembers={5} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*5 available/i });

    // Only the first MAX_VISIBLE_NAMES (3) render inline in the cell...
    expect(within(cell).getByText('Member 1.')).toBeInTheDocument();
    expect(within(cell).getByText('Member 2.')).toBeInTheDocument();
    expect(within(cell).getByText('Member 3.')).toBeInTheDocument();
    expect(within(cell).queryByText('Member 4.')).not.toBeInTheDocument();
    expect(within(cell).getByText('+2 more')).toBeInTheDocument();

    // ...but all 5 are still available via the popover.
    fireEvent.click(cell);
    const popover = await screen.findByRole('list');
    expect(within(popover).getAllByRole('listitem')).toHaveLength(5);
    expect(within(popover).getByText('Member 4')).toBeInTheDocument();
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

  it('does not apply urgent styling when the only open request is priority "optional"', async () => {
    // The whole point of the priority flag is that an optional request
    // shouldn't read as urgent in the group's busiest view - if this cell
    // still carried the needs-coverage class/label, the feature would exist
    // in the data model without changing anything a viewer actually sees.
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', priority: 'optional', coverage_request_id: 42, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM, 1 available/i });
    expect(cell).not.toHaveClass('schedule-grid__slot--needs-coverage');
    expect(screen.queryByRole('cell', { name: /needs coverage/i })).not.toBeInTheDocument();
  });

  it('still applies urgent styling when a mix of normal and optional requests share a cell', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', priority: 'optional', coverage_request_id: 42, claimable: true },
          { user_id: 3, username: 'vol3', status: 'needs_coverage', priority: 'normal', coverage_request_id: 43, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /needs coverage/i });
    expect(cell).toHaveClass('schedule-grid__slot--needs-coverage');
  });

  it('marks an optional needs_coverage member distinctly in the popover, without the warning icon', async () => {
    mockOverview({
      week_start: '2026-08-09',
      slots: [
        { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
          { user_id: 2, username: 'vol2', status: 'needs_coverage', priority: 'optional', coverage_request_id: 42, claimable: true },
        ] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM, 1 available/i });
    fireEvent.click(cell);

    const popover = await screen.findByRole('list');
    expect(within(popover).getByText(/needs coverage \(optional\)/i)).toBeInTheDocument();
    expect(within(cell).queryByText('⚠')).not.toBeInTheDocument();
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

  it('a Request coverage button in the popover opens the date-range form pre-filled to that exact date, instead of creating instantly', async () => {
    // The single-click quick-create was removed in favor of always going
    // through the date-range form (so a priority can be chosen), but the
    // popover is where a volunteer naturally notices "I can't make this
    // shift" - so it still gets an entry point, just one that opens the
    // form pre-filled to this one date rather than posting immediately.
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
      const popover = await screen.findByRole('list');
      expect(within(popover).getByText('me')).toBeInTheDocument();

      fireEvent.click(within(popover).getByRole('button', { name: /request coverage/i }));

      // No instant API call - opening the form is not the same as submitting it.
      expect(scheduleApi.createCoverageRequestsBatch).not.toHaveBeenCalled();

      expect(screen.getByLabelText(/start date/i)).toHaveValue('2026-08-11');
      expect(screen.getByLabelText(/end date/i)).toHaveValue('2026-08-11');
      expect(await screen.findByRole('checkbox', { name: /2026-08-11/ })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not show a Request coverage button in the popover for a past date', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    // Thursday of the week starting Sunday 2026-08-09 - the Tuesday slot
    // below falls two days earlier in that same week, so it's genuinely
    // in the past relative to the component's own today/weekStart state
    // (which is derived from the real pinned clock, not from the mocked
    // week_start field).
    vi.setSystemTime(new Date('2026-08-13T12:00:00Z'));
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
      const popover = await screen.findByRole('list');
      expect(within(popover).getByText('me')).toBeInTheDocument();
      expect(within(popover).queryByRole('button', { name: /request coverage/i })).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows a Reassign control on a normal-status row for an admin, and reassigns on confirm', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.reassignShiftsBatch).mockResolvedValue({
        data: { created: [], skipped: [] },
      } as unknown as AxiosResponse<ReassignShiftsBatchResult>);
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');

      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));
      fireEvent.change(within(popover).getByRole('combobox'), { target: { value: '3' } });
      fireEvent.click(within(popover).getByRole('button', { name: /confirm/i }));

      await waitFor(() => expect(scheduleApi.reassignShiftsBatch).toHaveBeenCalledWith(7, {
        fromUserId: 2,
        toUserId: 3,
        date: '2026-08-11',
        hours: [9],
        notify: true,
      }));
    } finally {
      vi.useRealTimers();
    }
  });

  it('lets the admin uncheck "Notify both volunteers by email" to skip notifications for this reassignment', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.reassignShiftsBatch).mockResolvedValue({
        data: { created: [], skipped: [] },
      } as unknown as AxiosResponse<ReassignShiftsBatchResult>);
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');

      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));
      const notifyCheckbox = within(popover).getByRole('checkbox', { name: /notify both volunteers by email/i });
      expect(notifyCheckbox).toBeChecked();
      fireEvent.click(notifyCheckbox);
      fireEvent.change(within(popover).getByRole('combobox'), { target: { value: '3' } });
      fireEvent.click(within(popover).getByRole('button', { name: /confirm/i }));

      await waitFor(() => expect(scheduleApi.reassignShiftsBatch).toHaveBeenCalledWith(7, {
        fromUserId: 2,
        toUserId: 3,
        date: '2026-08-11',
        hours: [9],
        notify: false,
      }));
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows a checklist of the same person\'s other shifts that day, pre-checked to just the clicked hour', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
          { date: '2026-08-11', day_of_week: 2, hour: 10, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');
      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));

      expect(within(popover).getByRole('checkbox', { name: '9:00 AM' })).toBeChecked();
      expect(within(popover).getByRole('checkbox', { name: '10:00 AM' })).not.toBeChecked();
    } finally {
      vi.useRealTimers();
    }
  });

  it('lets the admin include additional same-day hours, sending all checked hours to the batch endpoint', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.reassignShiftsBatch).mockResolvedValue({
        data: { created: [], skipped: [] },
      } as unknown as AxiosResponse<ReassignShiftsBatchResult>);
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
          { date: '2026-08-11', day_of_week: 2, hour: 10, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');
      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));
      fireEvent.click(within(popover).getByRole('checkbox', { name: '10:00 AM' }));
      fireEvent.change(within(popover).getByRole('combobox'), { target: { value: '3' } });
      fireEvent.click(within(popover).getByRole('button', { name: /confirm/i }));

      await waitFor(() => expect(scheduleApi.reassignShiftsBatch).toHaveBeenCalledWith(7, {
        fromUserId: 2,
        toUserId: 3,
        date: '2026-08-11',
        hours: [9, 10],
        notify: true,
      }));
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not show a checklist when there are no other same-day hours for that person', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');
      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));

      // Only the "Notify both volunteers by email" checkbox should be
      // present - no per-hour checklist checkboxes, since there are no
      // other same-day hours for this person to include.
      expect(within(popover).queryAllByRole('checkbox')).toHaveLength(1);
      expect(within(popover).getByRole('checkbox', { name: /notify both volunteers by email/i })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows a partial-success toast when some reassigned hours are skipped', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.reassignShiftsBatch).mockResolvedValue({
        data: {
          created: [{ id: 1, group_id: 7, requested_by_user_id: 2, date: '2026-08-11', hour: 9, status: 'claimed', priority: 'normal', claimed_by_user_id: 3 }],
          skipped: [{ hour: 10, reason: 'claimant already has a conflicting shift at that time' }],
        },
      } as unknown as AxiosResponse<ReassignShiftsBatchResult>);
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
          { date: '2026-08-11', day_of_week: 2, hour: 10, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');
      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));
      fireEvent.click(within(popover).getByRole('checkbox', { name: '10:00 AM' }));
      fireEvent.change(within(popover).getByRole('combobox'), { target: { value: '3' } });
      fireEvent.click(within(popover).getByRole('button', { name: /confirm/i }));

      await waitFor(() => expect(mockShowSuccess).toHaveBeenCalledWith('Reassigned 1 shift.'));
      expect(mockShowError).toHaveBeenCalledWith(expect.stringContaining('claimant already has a conflicting shift at that time'));
    } finally {
      vi.useRealTimers();
    }
  });

  it('keeps the popover open when every hour in the batch is skipped', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.reassignShiftsBatch).mockResolvedValue({
        data: {
          created: [],
          skipped: [{ hour: 9, reason: 'claimant already has a conflicting shift at that time' }],
        },
      } as unknown as AxiosResponse<ReassignShiftsBatchResult>);
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');
      fireEvent.click(within(popover).getByRole('button', { name: /reassign/i }));
      fireEvent.change(within(popover).getByRole('combobox'), { target: { value: '3' } });
      fireEvent.click(within(popover).getByRole('button', { name: /confirm/i }));

      await waitFor(() => expect(mockShowError).toHaveBeenCalledWith(expect.stringContaining('claimant already has a conflicting shift at that time')));

      // The popover/checklist should still be showing so the admin can retry,
      // rather than having been silently dismissed with nothing reassigned.
      expect(await screen.findByRole('list')).toBeInTheDocument();
      expect(within(screen.getByRole('list')).getByRole('button', { name: /confirm/i })).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not show a Reassign control when the viewer cannot manage members', async () => {
    // Pinned to a date before the slot (rather than relying on the real
    // clock) so this test isolates the canManageMembers gate specifically -
    // otherwise it could pass vacuously once 2026-08-11 is in the past,
    // for the wrong reason (the date gate, not the admin gate).
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={false} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');

      expect(within(popover).queryByRole('button', { name: /reassign/i })).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not show a Reassign control for a past date, even for an admin', async () => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-13T12:00:00Z'));
    try {
      mockOverview({
        week_start: '2026-08-09',
        slots: [
          { date: '2026-08-11', day_of_week: 2, hour: 9, members: [
            { user_id: 2, username: 'vol2', status: 'normal' },
          ] },
        ],
      });
      render(<ScheduleOverview groupId={7} totalMembers={4} currentUserId={1} canManageMembers={true} groupMembers={testMembers} />);

      const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM/i });
      fireEvent.click(cell);
      const popover = await screen.findByRole('list');

      expect(within(popover).queryByRole('button', { name: /reassign/i })).not.toBeInTheDocument();
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
