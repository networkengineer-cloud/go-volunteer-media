import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import ScheduleTab from './ScheduleTab';
import { scheduleApi, groupsApi } from '../../api/client';
import { CanceledError } from 'axios';
import type { AxiosResponse } from 'axios';
import type { ScheduleResponse, GroupMember, ScheduleOverviewResponse } from '../../api/client';
import { ToastProvider } from '../../contexts/ToastContext';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getMine: vi.fn(),
    updateMine: vi.fn(),
    getForMember: vi.fn(),
    updateForMember: vi.fn(),
    getOverview: vi.fn(),
  },
  groupsApi: {
    getMembers: vi.fn(),
  },
}));

function renderScheduleTab(canManageMembers: boolean) {
  return render(
    <ToastProvider>
      <ScheduleTab groupId={1} canManageMembers={canManageMembers} />
    </ToastProvider>
  );
}

describe('ScheduleTab', () => {
  beforeEach(() => {
    vi.mocked(scheduleApi.getMine).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleResponse>);
    vi.mocked(scheduleApi.updateMine).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleResponse>);
    vi.mocked(groupsApi.getMembers).mockResolvedValue({ data: [] } as unknown as AxiosResponse<GroupMember[]>);
    vi.mocked(scheduleApi.getOverview).mockResolvedValue({ data: { slots: [] } } as unknown as AxiosResponse<ScheduleOverviewResponse>);
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
      expect(scheduleApi.updateMine).toHaveBeenCalledWith(1, [{ day_of_week: 3, hour: 10 }]);
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
    const freshResponse = { data: { slots: [{ day_of_week: 6, hour: 17 }] } } as unknown as AxiosResponse<ScheduleResponse>;

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
        <ScheduleTab groupId={1} canManageMembers={false} />
      </ToastProvider>
    );

    await waitFor(() => expect(capturedFirstSignal).toBeDefined());
    expect(capturedFirstSignal?.aborted).toBe(false);

    // Switching to a new group before the first request resolves must abort it.
    rerender(
      <ToastProvider>
        <ScheduleTab groupId={2} canManageMembers={false} />
      </ToastProvider>
    );
    await waitFor(() => expect(capturedFirstSignal?.aborted).toBe(true));

    const freshSlot = await screen.findByRole('cell', { name: 'Sat 5:00 PM' });
    expect(freshSlot).toHaveAttribute('aria-pressed', 'true');

    // The first (now-aborted) request resolving late must not overwrite the
    // fresher data already rendered.
    await act(async () => {
      resolveFirst(staleResponse);
    });

    expect(screen.getByRole('cell', { name: 'Sat 5:00 PM' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('cell', { name: 'Sun 8:00 AM' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('does not show the individual/overview toggle for a non-admin member', async () => {
    renderScheduleTab(false);
    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalled());
    expect(screen.queryByRole('button', { name: /overview/i })).not.toBeInTheDocument();
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
});
