import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ScheduleTab from './ScheduleTab';
import { scheduleApi, groupsApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { ScheduleResponse, GroupMember } from '../../api/client';
import { ToastProvider } from '../../contexts/ToastContext';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getMine: vi.fn(),
    updateMine: vi.fn(),
    getForMember: vi.fn(),
    updateForMember: vi.fn(),
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
  });

  it('loads and displays the caller\'s own schedule by default', async () => {
    vi.mocked(scheduleApi.getMine).mockResolvedValue({
      data: { slots: [{ day_of_week: 2, hour: 9 }] },
    } as unknown as AxiosResponse<ScheduleResponse>);

    renderScheduleTab(false);

    await waitFor(() => expect(scheduleApi.getMine).toHaveBeenCalledWith(1));
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

    await waitFor(() => expect(scheduleApi.getForMember).toHaveBeenCalledWith(1, 42));
    const slot = await screen.findByRole('cell', { name: 'Fri 4:00 PM' });
    expect(slot).toHaveAttribute('aria-pressed', 'true');
  });
});
