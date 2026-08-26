import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import NeedsCoverageList from './NeedsCoverageList';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { CoverageRequestListItem, CoverageRequestCancelBatchResult } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    listCoverageRequests: vi.fn(),
    claimCoverageRequest: vi.fn(),
    cancelCoverageRequestsBatch: vi.fn(),
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

  describe('bulk cancel', () => {
    function mockCancelBatch(result: CoverageRequestCancelBatchResult) {
      vi.mocked(scheduleApi.cancelCoverageRequestsBatch).mockResolvedValue({ data: result } as unknown as AxiosResponse<CoverageRequestCancelBatchResult>);
    }

    it('shows a checkbox on the viewer\'s own open request', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, claimable: false },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      await screen.findByText('Me');
      expect(screen.getByRole('checkbox', { name: /select.*2026-08-11/i })).toBeInTheDocument();
    });

    it('does not show a checkbox on another member\'s request when the viewer is not an admin', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers={false} />);

      await screen.findByText('Jane Doe');
      expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    });

    it('shows a checkbox on another member\'s request when the viewer is a group admin', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers />);

      await screen.findByText('Jane Doe');
      expect(screen.getByRole('checkbox', { name: /select.*2026-08-11/i })).toBeInTheDocument();
    });

    it('cancelling selected requests calls cancelCoverageRequestsBatch with the selected ids and refetches', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, claimable: false },
        { id: 6, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-13', hour: 10, claimable: false },
      ]);
      mockCancelBatch({ cancelled: [], skipped: [] });
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /2026-08-13/i }));

      const callsBeforeCancel = vi.mocked(scheduleApi.listCoverageRequests).mock.calls.length;
      fireEvent.click(screen.getByRole('button', { name: /cancel selected/i }));

      await waitFor(() => expect(scheduleApi.cancelCoverageRequestsBatch).toHaveBeenCalledWith(7, [6]));
      await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledTimes(callsBeforeCancel + 1));
    });

    it('shows a summary toast reporting cancelled and skipped counts', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, claimable: false },
      ]);
      mockCancelBatch({
        cancelled: [{ id: 5, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 9, status: 'cancelled', claimed_by_user_id: null }],
        skipped: [{ id: 6, reason: 'coverage request has already been claimed' }],
      });
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /2026-08-11/i }));
      fireEvent.click(screen.getByRole('button', { name: /cancel selected/i }));

      await waitFor(() => expect(mockShowSuccess).toHaveBeenCalledWith('Cancelled 1 request.'));
      await waitFor(() => expect(mockShowError).toHaveBeenCalledWith('1 request could not be cancelled.'));
    });

    it('disables the Cancel selected button until at least one request is checked', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, claimable: false },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      const cancelButton = await screen.findByRole('button', { name: /cancel selected/i });
      expect(cancelButton).toBeDisabled();

      fireEvent.click(screen.getByRole('checkbox', { name: /2026-08-11/i }));
      expect(cancelButton).toBeEnabled();
    });
  });
});
