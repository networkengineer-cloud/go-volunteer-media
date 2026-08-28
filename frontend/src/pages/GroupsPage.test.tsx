import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import GroupsPage from './GroupsPage';
import { groupsApi, animalsApi, statisticsApi } from '../api/client';
import type { Group } from '../api/client';
import type { AxiosResponse } from 'axios';
import { ToastProvider } from '../contexts/ToastContext';

// Only the scheduling checkbox behavior added to the Edit Group modal is
// covered here - it's new logic (edit-only rendering, and diffing against
// the group's current value before firing a second API call) that isn't
// exercised anywhere else.
vi.mock('../api/client', () => ({
  groupsApi: {
    getAll: vi.fn(),
    update: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    updateScheduling: vi.fn(),
    uploadImage: vi.fn(),
  },
  animalsApi: {
    getAll: vi.fn(),
  },
  statisticsApi: {
    getGroupStatistics: vi.fn(),
  },
}));

const schedulingOffGroup: Group = {
  id: 1,
  name: 'Dogs',
  description: 'Dog volunteers',
  image_url: '',
  hero_image_url: '',
  has_protocols: false,
  scheduling_enabled: false,
  groupme_enabled: false,
};

const schedulingOnGroup: Group = {
  ...schedulingOffGroup,
  id: 2,
  name: 'Cats',
  scheduling_enabled: true,
};

const renderGroupsPage = () =>
  render(
    <BrowserRouter>
      <ToastProvider>
        <GroupsPage />
      </ToastProvider>
    </BrowserRouter>,
  );

describe('GroupsPage scheduling setting', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(groupsApi.getAll).mockResolvedValue({
      data: [schedulingOffGroup, schedulingOnGroup],
    } as AxiosResponse<Group[]>);
    vi.mocked(statisticsApi.getGroupStatistics).mockResolvedValue({
      data: { data: [] },
    } as AxiosResponse<{ data: [] }>);
    vi.mocked(animalsApi.getAll).mockResolvedValue({ data: [] } as AxiosResponse<[]>);
    vi.mocked(groupsApi.update).mockResolvedValue({ data: schedulingOffGroup } as AxiosResponse<Group>);
    vi.mocked(groupsApi.updateScheduling).mockResolvedValue({
      data: { scheduling_enabled: true },
    } as AxiosResponse<{ scheduling_enabled: boolean }>);
  });

  it('does not show a scheduling checkbox when creating a new group', async () => {
    renderGroupsPage();
    fireEvent.click(await screen.findByRole('button', { name: /add group/i }));

    expect(await screen.findByLabelText(/name/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/enable scheduling/i)).not.toBeInTheDocument();
  });

  it('reflects the group\'s current scheduling_enabled value when editing', async () => {
    renderGroupsPage();
    // Default sort is by name ascending, so "Cats" (index 0) sorts before "Dogs" (index 1).
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' });
    fireEvent.click(editButtons[0]); // Cats - scheduling already enabled

    const checkbox = await screen.findByLabelText(/enable scheduling/i);
    expect(checkbox).toBeChecked();
  });

  it('does not call updateScheduling when the checkbox is left unchanged', async () => {
    renderGroupsPage();
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' });
    fireEvent.click(editButtons[1]); // Dogs - scheduling off

    await screen.findByLabelText(/enable scheduling/i);
    fireEvent.click(screen.getByRole('button', { name: /update/i }));

    await waitFor(() => expect(groupsApi.update).toHaveBeenCalled());
    expect(groupsApi.updateScheduling).not.toHaveBeenCalled();
  });

  it('calls updateScheduling with the new value when the checkbox is toggled', async () => {
    renderGroupsPage();
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' });
    fireEvent.click(editButtons[1]); // Dogs (id 1) - scheduling off

    const checkbox = await screen.findByLabelText(/enable scheduling/i);
    fireEvent.click(checkbox);
    fireEvent.click(screen.getByRole('button', { name: /update/i }));

    await waitFor(() => expect(groupsApi.updateScheduling).toHaveBeenCalledWith(1, true));
  });
});
