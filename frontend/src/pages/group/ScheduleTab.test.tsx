import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act, within } from '@testing-library/react';
import ScheduleTab from './ScheduleTab';
import { scheduleApi, groupsApi } from '../../api/client';
import { CanceledError } from 'axios';
import type { AxiosResponse } from 'axios';
import type { ScheduleResponse, GroupMember, ScheduleOverviewResponse, CoverageRequestListItem } from '../../api/client';
import { ToastProvider } from '../../contexts/ToastContext';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getMine: vi.fn(),
    updateMine: vi.fn(),
    getForMember: vi.fn(),
    updateForMember: vi.fn(),
    getOverview: vi.fn(),
    listCoverageRequests: vi.fn(),
  },
  groupsApi: {
    getMembers: vi.fn(),
  },
}));

function renderScheduleTab(canManageMembers: boolean) {
  return render(
    <ToastProvider>
      <ScheduleTab groupId={1} canManageMembers={canManageMembers} currentUserId={1} />
    </ToastProvider>
  );
}

describe('ScheduleTab', () => {
  beforeEach(() => {
    vi.mocked(scheduleApi.getMine).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleResponse>);
    vi.mocked(scheduleApi.updateMine).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleResponse>);
    vi.mocked(groupsApi.getMembers).mockResolvedValue({ data: [] } as unknown as AxiosResponse<GroupMember[]>);
    vi.mocked(scheduleApi.getOverview).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleOverviewResponse>);
    vi.mocked(scheduleApi.listCoverageRequests).mockResolvedValue({ data: [] } as unknown as AxiosResponse<CoverageRequestListItem[]>);
  });

  it('loads and displays the caller\'s own schedule by default', async () => {
    vi.mocked(scheduleApi.getMine).mockResolvedValue({
      data: { slots: [{ day_of_week: 2, hour: 9 }] },
    } as unknown as AxiosResponse<ScheduleResponse>);

    renderScheduleTab(false);

    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalledWith(1, expect.objectContaining({ signal: expect.any(AbortSignal) })));
    const slot = await screen.findByRole('cell', { name: 'Tue 9:00 AM' });
    expect(slot).toHaveAttribute('aria-pressed', 'true');
  });

  it('toggling a cell and saving calls updateMine with the new slot set', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    const slot = await screen.findByRole('cell', { name: 'Wed 10:00 AM' });
    fireEvent.click(slot);
    expect(slot).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(screen.getByRole('button', { name: /save schedule/i }));

    await waitFor(() => {
      expect(scheduleApi.updateMine).toHaveBeenCalledWith(1, [{ day_of_week: 3, hour: 10, cadence: 'weekly' }]);
    });
  });

  it('does not show the volunteer picker for a non-admin member', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());
    expect(screen.queryByLabelText(/viewing schedule for/i)).not.toBeInTheDocument();
  });

  it('a group admin can pick another member and load their schedule', async () => {
    vi.mocked(groupsApi.getMembers).mockResolvedValue({
      data: [{ user_id: 42, username: 'vol2', is_group_admin: false, is_site_admin: false, email: '', skill_tags: [] }],
    } as unknown as AxiosResponse<GroupMember[]>);
    vi.mocked(scheduleApi.getForMember).mockResolvedValue({
      data: { slots: [{ day_of_week: 5, hour: 16 }] },
    } as unknown as AxiosResponse<ScheduleResponse>);

    renderScheduleTab(true);
    await waitFor(() => expect(groupsApi.getMembers).toHaveBeenCalledWith(1));

    fireEvent.change(screen.getByLabelText(/viewing schedule for/i), { target: { value: '42' } });

    await waitFor(() => expect(scheduleApi.getForMember).toHaveBeenCalledWith(1, 42, expect.objectContaining({ signal: expect.any(AbortSignal) })));
    const slot = await screen.findByRole('cell', { name: 'Fri 4:00 PM' });
    expect(slot).toHaveAttribute('aria-pressed', 'true');
  });

  it('does not let a superseded (aborted) request overwrite a newer group\'s data', async () => {
    // Simulates GroupPage's group-switcher changing the groupId prop while
    // the previous group's schedule is still loading (the volunteer picker
    // itself unmounts during loading - see the `loading` early-return above
    // - so this prop-driven race is the realistic path, not picker clicks).
    let resolveFirst!: (value: AxiosResponse<ScheduleResponse>) => void;
    let capturedFirstSignal: AbortSignal | undefined;
    const staleResponse = { data: { slots: [{ day_of_week: 0, hour: 8 }] } } as unknown as AxiosResponse<ScheduleResponse>;
    const freshResponse = { data: { slots: [{ day_of_week: 6, hour: 9 }] } } as unknown as AxiosResponse<ScheduleResponse>;

    // First group's own-schedule fetch: stays pending until resolveFirst is
    // invoked, but rejects immediately - the way a real aborted axios
    // request would - if its signal fires first.
    vi.mocked(scheduleApi.getMine)
      .mockImplementationOnce((_groupId, options) => {
        capturedFirstSignal = options?.signal;
        return new Promise<AxiosResponse<ScheduleResponse>>((resolve, reject) => {
          resolveFirst = resolve;
          options?.signal?.addEventListener('abort', () => reject(new CanceledError()));
        });
      })
      // Second group's fetch (after the prop changes): resolves immediately
      // with different data.
      .mockResolvedValueOnce(freshResponse);

    const { rerender } = render(
      <ToastProvider>
        <ScheduleTab groupId={1} canManageMembers={false} currentUserId={1} />
      </ToastProvider>
    );

    await waitFor(() => expect(capturedFirstSignal).toBeDefined());
    expect(capturedFirstSignal?.aborted).toBe(false);

    // Switching to a new group before the first request resolves must abort it.
    rerender(
      <ToastProvider>
        <ScheduleTab groupId={2} canManageMembers={false} currentUserId={1} />
      </ToastProvider>
    );
    await waitFor(() => expect(capturedFirstSignal?.aborted).toBe(true));

    const freshSlot = await screen.findByRole('cell', { name: 'Sat 9:00 AM' });
    expect(freshSlot).toHaveAttribute('aria-pressed', 'true');

    // The first (now-aborted) request resolving late must not overwrite the
    // fresher data already rendered.
    await act(async () => {
      resolveFirst(staleResponse);
    });

    expect(screen.getByRole('cell', { name: 'Sat 9:00 AM' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('cell', { name: 'Sun 8:00 AM' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('shows the Individual/Overview toggle for non-admin members too', async () => {
    render(
      <ToastProvider>
        <ScheduleTab groupId={7} canManageMembers={false} currentUserId={1} />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByRole('group', { name: /schedule view/i })).toBeInTheDocument());
  });

  it('a group admin can switch to the overview and back to individual view', async () => {
    vi.mocked(groupsApi.getMembers).mockResolvedValue({
      data: [{ user_id: 42, username: 'vol2', is_group_admin: false, is_site_admin: false, email: '', skill_tags: [] }],
    } as unknown as AxiosResponse<GroupMember[]>);

    renderScheduleTab(true);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: /overview/i }));

    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalledWith(1, expect.objectContaining({ signal: expect.any(AbortSignal) })));
    expect(screen.queryByLabelText(/viewing schedule for/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /save schedule/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /individual/i }));
    expect(await screen.findByLabelText(/viewing schedule for/i)).toBeInTheDocument();
  });

  it('applies the active toggle style to the selected view', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    const individualButton = screen.getByRole('button', { name: /individual/i });
    const overviewButton = screen.getByRole('button', { name: /overview/i });
    expect(individualButton).toHaveClass('schedule-tab__view-toggle-btn--active');
    expect(overviewButton).not.toHaveClass('schedule-tab__view-toggle-btn--active');

    fireEvent.click(overviewButton);

    await waitFor(() => expect(overviewButton).toHaveClass('schedule-tab__view-toggle-btn--active'));
    expect(individualButton).not.toHaveClass('schedule-tab__view-toggle-btn--active');
  });

  it('offers a Needs Coverage toggle that loads the open-requests list', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: /needs coverage/i }));

    await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledWith(1, expect.objectContaining({ signal: expect.any(AbortSignal) })));
    expect(await screen.findByText(/no shifts currently need coverage/i)).toBeInTheDocument();
    expect(screen.queryByRole('table', { name: /weekly shift schedule/i })).not.toBeInTheDocument();
  });

  it('opens the bulk request-coverage form when viewing my own schedule, and hides it when viewing another member\'s', async () => {
    vi.mocked(scheduleApi.getMine).mockResolvedValue({
      data: { slots: [{ day_of_week: 2, hour: 9 }] },
    } as unknown as AxiosResponse<ScheduleResponse>);
    vi.mocked(groupsApi.getMembers).mockResolvedValue({
      data: [{ user_id: 2, username: 'vol2', is_group_admin: false, is_site_admin: false, email: '', skill_tags: [] }],
    } as unknown as AxiosResponse<GroupMember[]>);
    vi.mocked(scheduleApi.getForMember).mockResolvedValue({
      data: { slots: [] },
    } as unknown as AxiosResponse<ScheduleResponse>);

    render(
      <ToastProvider>
        <ScheduleTab groupId={7} canManageMembers={true} currentUserId={1} />
      </ToastProvider>
    );
    await waitFor(() => expect(screen.getByRole('button', { name: /request coverage for a date range/i })).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /request coverage for a date range/i }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByLabelText(/start date/i)).toBeInTheDocument();

    // Closing and switching to another member's schedule hides the button.
    fireEvent.click(screen.getByRole('button', { name: /close modal/i }));
    fireEvent.change(screen.getByLabelText(/viewing schedule for/i), { target: { value: '2' } });
    await waitFor(() => expect(screen.queryByRole('button', { name: /request coverage for a date range/i })).not.toBeInTheDocument());
  });

  it('offers the request-coverage form only the persisted schedule, not an unsaved toggle', async () => {
    // Regression test: "toggle a cell, then open Request Coverage without
    // saving" is a natural flow regardless of the button's position on the
    // page. The form's candidate list
    // must reflect only what's actually persisted server-side (savedSlots),
    // not the live, possibly-unsaved grid selection - otherwise the form
    // pre-checks occurrences the backend will just skip (unsaved additions)
    // or silently drops real shifts (unsaved removals).
    //
    // The assertions below depend on 2026-08-11/2026-08-12 being real
    // "today or later" occurrences (computeCandidateOccurrences drops
    // anything before today), so the system clock is pinned here - matching
    // RequestCoverageRangeForm.test.tsx's convention - rather than relying
    // on the real wall clock, which would make this test start failing the
    // day after the pinned date.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-08-10T12:00:00Z'));
    try {
      vi.mocked(scheduleApi.getMine).mockResolvedValue({
        data: { slots: [{ day_of_week: 2, hour: 9 }] }, // Tuesday 9am, persisted
      } as unknown as AxiosResponse<ScheduleResponse>);

      render(
        <ToastProvider>
          <ScheduleTab groupId={7} canManageMembers={false} currentUserId={1} />
        </ToastProvider>
      );
      await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

      // Toggle on an additional, unsaved cell (Wednesday 10am) without saving.
      const unsavedCell = await screen.findByRole('cell', { name: 'Wed 10:00 AM' });
      fireEvent.click(unsavedCell);
      expect(unsavedCell).toHaveAttribute('aria-pressed', 'true');

      fireEvent.click(screen.getByRole('button', { name: /request coverage for a date range/i }));
      await screen.findByRole('dialog');

      // Pick a date range spanning the persisted Tuesday slot's one occurrence
      // (2026-08-11) and the unsaved Wednesday toggle's occurrence the next
      // day (2026-08-12).
      fireEvent.change(screen.getByLabelText(/start date/i), { target: { value: '2026-08-11' } });
      fireEvent.change(screen.getByLabelText(/end date/i), { target: { value: '2026-08-12' } });

      // Only the persisted Tuesday 9am occurrence should be offered - the
      // unsaved Wednesday 10am toggle must not appear as a candidate.
      const candidates = await screen.findAllByRole('checkbox', { name: /2026-08-\d{2}/ });
      expect(candidates).toHaveLength(1);
      expect(screen.getByRole('checkbox', { name: /9:00 AM/ })).toBeInTheDocument();
      expect(screen.queryByRole('checkbox', { name: /10:00 AM/ })).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('cycles a cell through weekly -> biweekly A -> biweekly B -> empty on repeated clicks', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    const slot = await screen.findByRole('cell', { name: 'Wed 10:00 AM' });
    fireEvent.click(slot);
    expect(slot).toHaveAttribute('aria-pressed', 'true');
    expect(slot).toHaveAccessibleName('Wed 10:00 AM');
    expect(slot).toHaveClass('schedule-grid__slot--weekly');

    fireEvent.click(slot);
    expect(slot).toHaveAccessibleName('Wed 10:00 AM (Week A)');
    expect(slot).toHaveClass('schedule-grid__slot--biweekly-a');
    expect(slot).not.toHaveClass('schedule-grid__slot--weekly');

    fireEvent.click(slot);
    expect(slot).toHaveAccessibleName('Wed 10:00 AM (Week B)');
    expect(slot).toHaveClass('schedule-grid__slot--biweekly-b');
    expect(slot).not.toHaveClass('schedule-grid__slot--biweekly-a');

    fireEvent.click(slot);
    expect(slot).toHaveAttribute('aria-pressed', 'false');
    expect(slot).not.toHaveClass('schedule-grid__slot--weekly');
    expect(slot).not.toHaveClass('schedule-grid__slot--biweekly-a');
    expect(slot).not.toHaveClass('schedule-grid__slot--biweekly-b');
  });

  it('disables cells beyond the weekend cap', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    // Disabled cells stay real `role="cell"` elements (marked
    // aria-disabled) rather than `role="presentation"`, so every row in the
    // ARIA table exposes the same number of cells - see the next test for a
    // dedicated check of that ARIA contract.
    expect(screen.queryByRole('cell', { name: /Sun 4:00 PM/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('cell', { name: /Sun 5:00 PM/ })).not.toBeInTheDocument();
    // The weekend's terminal slot (3:00-4:30pm) is still a live, clickable cell.
    expect(screen.getByRole('cell', { name: 'Sun 3:00–4:30 PM' })).toBeInTheDocument();
    // Weekdays keep their full range including the terminal slot.
    expect(screen.getByRole('cell', { name: 'Mon 5:00–6:30 PM' })).toBeInTheDocument();
  });

  it('exposes out-of-range weekend cells as aria-disabled cells, not presentation elements, to keep the ARIA table\'s column count consistent', async () => {
    // Regression test: role="presentation" would drop these cells out of
    // the ARIA table structure entirely, so a weekend row (hours 16-17
    // disabled) would expose fewer role="cell" elements than a weekday row
    // in the same role="table" - a ragged/inconsistent table structure that
    // assistive tech may report confusingly. role="cell" + aria-disabled
    // keeps every row's cell count identical while still communicating
    // "not interactive" to a screen reader.
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    const table = screen.getByRole('table', { name: /weekly shift schedule/i });
    const rows = within(table).getAllByRole('row');
    // First row is the header row; the rest are the 10 hour rows (8am-5pm).
    const hourRows = rows.slice(1);
    expect(hourRows).toHaveLength(10);
    const cellCounts = hourRows.map(row => within(row).getAllByRole('cell').length);
    // Every hour row (weekday or weekend) must expose the same number of
    // role="cell" elements - 7 days - regardless of how many are disabled.
    expect(cellCounts).toEqual(cellCounts.map(() => 7));

    // Spot-check: the disabled Sunday 4pm cell (hour 16, row index 8 of the
    // 10 hour rows: 8,9,10,11,12,13,14,15,16,17) is a real role="cell",
    // marked aria-disabled, not a role="presentation" element.
    const sunday4pmRow = hourRows[8];
    const disabledCells = within(sunday4pmRow).getAllByRole('cell')
      .filter(cell => cell.getAttribute('aria-disabled') === 'true');
    expect(disabledCells.length).toBeGreaterThan(0);
    expect(disabledCells[0]).not.toHaveAttribute('role', 'presentation');
  });

  it('saving includes the cadence for every selected cell', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    const slot = await screen.findByRole('cell', { name: 'Wed 10:00 AM' });
    fireEvent.click(slot); // weekly
    fireEvent.click(slot); // biweekly_a

    fireEvent.click(screen.getByRole('button', { name: /save schedule/i }));

    await waitFor(() => {
      expect(scheduleApi.updateMine).toHaveBeenCalledWith(1, [{ day_of_week: 3, hour: 10, cadence: 'biweekly_a' }]);
    });
  });

  it('shows the A/B legend above the grid', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());
    expect(screen.getByText(/Week A:/)).toBeInTheDocument();
    expect(screen.getByText(/Week B:/)).toBeInTheDocument();
  });

  it('shows a caption explaining the day-of-week hours and 90-min terminal slot, and badges only the terminal cell', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    // The 90-min closing slot previously had no visible UI signal beyond
    // an aria-label and a dashed border - this caption makes both day
    // types' hours and the 90-min terminal slot explicit and visible.
    expect(screen.getByText(/Weekdays: 8am–6:30pm/)).toBeInTheDocument();
    expect(screen.getByText(/Weekends: 8am–4:30pm/)).toBeInTheDocument();
    expect(screen.getByText(/last shift 90 min/)).toBeInTheDocument();

    // The weekday terminal cell (5:00-6:30pm, hour 17) shows a small "90m"
    // badge alongside its dashed-border styling...
    const terminalCell = screen.getByRole('cell', { name: 'Mon 5:00–6:30 PM' });
    expect(within(terminalCell).getByText('90m')).toBeInTheDocument();

    // ...but an ordinary 60-min cell does not.
    const nonTerminalCell = screen.getByRole('cell', { name: 'Mon 4:00 PM' });
    expect(within(nonTerminalCell).queryByText('90m')).not.toBeInTheDocument();
  });

  it('drops a legacy out-of-range slot from the loaded schedule so it is never resubmitted on save', async () => {
    // Saturday 5pm predates the weekend hour cap (weekends now cap at 3pm /
    // hour 15) - a real user could still have this persisted server-side.
    // It must render as a disabled, non-interactive cell (never an active
    // toggle) and must be silently dropped rather than round-tripped back
    // to the backend on save, since the backend itself now rejects it.
    vi.mocked(scheduleApi.getMine).mockResolvedValue({
      data: { slots: [{ day_of_week: 6, hour: 17, cadence: 'weekly' }] },
    } as unknown as AxiosResponse<ScheduleResponse>);

    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());

    // Never rendered as an interactive, checked toggle.
    expect(screen.queryByRole('cell', { name: /Sat 5:00 PM/ })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /save schedule/i }));

    await waitFor(() => {
      expect(scheduleApi.updateMine).toHaveBeenCalledWith(1, []);
    });
  });
});
