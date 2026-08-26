import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import NeedsCoverageList from './NeedsCoverageList';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { CoverageRequestListItem } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    listCoverageRequests: vi.fn(),
    claimCoverageRequest: vi.fn(),
  },
}));

const mockShowSuccess = vi.fn();
const mockShowError = vi.fn();
vi.mock('../../hooks/useToast', () => ({
  useToast: () => ({ showSuccess: mockShowSuccess, showError: mockShowError }),
}));

function mockList(items: CoverageRequestListItem[]) {
  vi.mocked(scheduleApi.listCoverageRequests).mockResolvedValue({ data: items } as unknown as AxiosResponse<CoverageRequestListItem[]>);
}

describe('NeedsCoverageList', () => {
  beforeEach(() => {
    mockList([]);
    mockShowSuccess.mockClear();
    mockShowError.mockClear();
  });

  it('loads open coverage requests for the group on mount', async () => {
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);
    await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledWith(7, expect.objectContaining({ signal: expect.any(AbortSignal) })));
  });

  it('shows an empty state when nothing needs coverage', async () => {
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);
    expect(await screen.findByText(/no shifts currently need coverage/i)).toBeInTheDocument();
  });

  it('lists an open request with requester name, date and time', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: true },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument();
    expect(screen.getByText(/9:00 AM/)).toBeInTheDocument();
  });

  it('claiming a shift calls claimCoverageRequest and refetches the list', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: true },
    ]);
    vi.mocked(scheduleApi.claimCoverageRequest).mockResolvedValue({} as unknown as AxiosResponse);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    await screen.findByRole('button', { name: /claim/i });
    const callsBeforeClaim = vi.mocked(scheduleApi.listCoverageRequests).mock.calls.length;
    fireEvent.click(screen.getByRole('button', { name: /claim/i }));

    await waitFor(() => expect(scheduleApi.claimCoverageRequest).toHaveBeenCalledWith(7, 5));
    await waitFor(() => expect(mockShowSuccess).toHaveBeenCalled());
    // Claiming triggers exactly one refetch so the list reflects the new state.
    await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledTimes(callsBeforeClaim + 1));
  });

  it('shows a failure toast and does not crash when claiming fails', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: true },
    ]);
    vi.mocked(scheduleApi.claimCoverageRequest).mockRejectedValue({ response: { data: { error: 'Already claimed' } } });
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    fireEvent.click(await screen.findByRole('button', { name: /claim/i }));

    await waitFor(() => expect(mockShowError).toHaveBeenCalledWith('Already claimed'));
  });

  it('disables the Claim button and explains why when the viewer has a conflicting shift', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: false },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    const claimButton = await screen.findByRole('button', { name: /claim/i });
    expect(claimButton).toBeDisabled();
    expect(claimButton).toHaveAttribute('title', expect.stringMatching(/conflicting shift/i));
  });

  it('does not show a Claim button for the viewer\'s own request', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, claimable: false },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    await screen.findByText('Me');
    expect(screen.queryByRole('button', { name: /claim/i })).not.toBeInTheDocument();
  });
});
