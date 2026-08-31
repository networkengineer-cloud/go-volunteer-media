import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import NeedsCoverageList from './NeedsCoverageList';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { CoverageRequestListItem, CoverageRequestCancelBatchResult, CoverageRequestClaimBatchResult } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    listCoverageRequests: vi.fn(),
    claimCoverageRequest: vi.fn(),
    cancelCoverageRequestsBatch: vi.fn(),
    claimCoverageRequestsBatch: vi.fn(),
    updateCoverageRequestPriority: vi.fn(),
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
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument();
    expect(screen.getByText(/9:00 AM/)).toBeInTheDocument();
  });

  it('shows the 90-min start-end range for a terminal-hour shift, not just the start time', async () => {
    // 2026-08-11 is a Tuesday (a weekday); hour 17 is that day's terminal
    // (maxHourFor) slot, a 90-min shift ending 6:30 PM. Someone deciding
    // whether to claim it needs to see the actual end time.
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 17, priority: 'normal', claimable: true },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    expect(await screen.findByText(/5:00–6:30 PM/)).toBeInTheDocument();
  });

  it('claiming a shift calls claimCoverageRequest and refetches the list', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
    ]);
    vi.mocked(scheduleApi.claimCoverageRequest).mockResolvedValue({} as unknown as AxiosResponse);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    await screen.findByRole('button', { name: 'Claim' });
    const callsBeforeClaim = vi.mocked(scheduleApi.listCoverageRequests).mock.calls.length;
    fireEvent.click(screen.getByRole('button', { name: 'Claim' }));

    await waitFor(() => expect(scheduleApi.claimCoverageRequest).toHaveBeenCalledWith(7, 5));
    await waitFor(() => expect(mockShowSuccess).toHaveBeenCalled());
    // Claiming triggers exactly one refetch so the list reflects the new state.
    await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledTimes(callsBeforeClaim + 1));
  });

  it('shows a failure toast and does not crash when claiming fails', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
    ]);
    vi.mocked(scheduleApi.claimCoverageRequest).mockRejectedValue({ response: { data: { error: 'Already claimed' } } });
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    fireEvent.click(await screen.findByRole('button', { name: 'Claim' }));

    await waitFor(() => expect(mockShowError).toHaveBeenCalledWith('Already claimed'));
  });

  it('disables the Claim button and explains why when the viewer has a conflicting shift', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    const claimButton = await screen.findByRole('button', { name: /claim/i });
    expect(claimButton).toBeDisabled();
    expect(claimButton).toHaveAttribute('title', expect.stringMatching(/conflicting shift/i));
  });

  it('does not show a Claim button for the viewer\'s own request', async () => {
    mockList([
      { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
    ]);
    render(<NeedsCoverageList groupId={7} currentUserId={1} />);

    await screen.findByText('Me');
    expect(screen.queryByRole('button', { name: 'Claim' })).not.toBeInTheDocument();
  });

  describe('bulk cancel', () => {
    function mockCancelBatch(result: CoverageRequestCancelBatchResult) {
      vi.mocked(scheduleApi.cancelCoverageRequestsBatch).mockResolvedValue({ data: result } as unknown as AxiosResponse<CoverageRequestCancelBatchResult>);
    }

    it('shows a checkbox on the viewer\'s own open request', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      await screen.findByText('Me');
      expect(screen.getByRole('checkbox', { name: /select.*2026-08-11/i })).toBeInTheDocument();
    });

    it('does not show a checkbox on a request the viewer can neither cancel nor claim', async () => {
      // Not the viewer's own request (so not cancellable, viewer isn't admin)
      // and not claimable (e.g. a conflicting shift) - there is no bulk
      // action this row could join, so no checkbox.
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers={false} />);

      await screen.findByText('Jane Doe');
      expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    });

    it('shows a checkbox on another member\'s request when the viewer is a group admin', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers />);

      await screen.findByText('Jane Doe');
      expect(screen.getByRole('checkbox', { name: /select.*2026-08-11/i })).toBeInTheDocument();
    });

    it('cancelling selected requests calls cancelCoverageRequestsBatch with the selected ids and refetches', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
        { id: 6, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-13', hour: 10, priority: 'normal', claimable: false },
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
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
      ]);
      mockCancelBatch({
        cancelled: [{ id: 5, group_id: 7, requested_by_user_id: 1, date: '2026-08-11', hour: 9, status: 'cancelled', priority: 'normal', claimed_by_user_id: null }],
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
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      const cancelButton = await screen.findByRole('button', { name: /cancel selected/i });
      expect(cancelButton).toBeDisabled();

      fireEvent.click(screen.getByRole('checkbox', { name: /2026-08-11/i }));
      expect(cancelButton).toBeEnabled();
    });
  });

  describe('bulk claim', () => {
    function mockClaimBatch(result: CoverageRequestClaimBatchResult) {
      vi.mocked(scheduleApi.claimCoverageRequestsBatch).mockResolvedValue({ data: result } as unknown as AxiosResponse<CoverageRequestClaimBatchResult>);
    }

    it('shows a checkbox on another member\'s claimable request even for a non-admin viewer', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers={false} />);

      await screen.findByText('Jane Doe');
      expect(screen.getByRole('checkbox', { name: /select.*2026-08-11/i })).toBeInTheDocument();
    });

    it('claiming selected requests calls claimCoverageRequestsBatch with the selected ids and refetches', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
        { id: 6, group_id: 7, requested_by_user_id: 3, requested_by_name: 'John Roe', date: '2026-08-13', hour: 10, priority: 'normal', claimable: true },
      ]);
      mockClaimBatch({ claimed: [], skipped: [] });
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /2026-08-13/i }));

      const callsBeforeClaim = vi.mocked(scheduleApi.listCoverageRequests).mock.calls.length;
      fireEvent.click(screen.getByRole('button', { name: /claim selected/i }));

      await waitFor(() => expect(scheduleApi.claimCoverageRequestsBatch).toHaveBeenCalledWith(7, [6]));
      await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledTimes(callsBeforeClaim + 1));
    });

    it('shows a summary toast reporting claimed and skipped counts', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      mockClaimBatch({
        claimed: [{ id: 5, group_id: 7, requested_by_user_id: 2, date: '2026-08-11', hour: 9, status: 'claimed', priority: 'normal', claimed_by_user_id: 1 }],
        skipped: [{ id: 6, reason: 'coverage request is no longer open' }],
      });
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /2026-08-11/i }));
      fireEvent.click(screen.getByRole('button', { name: /claim selected/i }));

      await waitFor(() => expect(mockShowSuccess).toHaveBeenCalledWith('Claimed 1 request.'));
      await waitFor(() => expect(mockShowError).toHaveBeenCalledWith('1 request could not be claimed.'));
    });

    it('disables the Claim selected button until at least one claimable request is checked', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      const claimSelectedButton = await screen.findByRole('button', { name: /claim selected/i });
      expect(claimSelectedButton).toBeDisabled();

      fireEvent.click(screen.getByRole('checkbox', { name: /2026-08-11/i }));
      expect(claimSelectedButton).toBeEnabled();
    });

    it('checking a mix of the viewer\'s own request and another\'s claimable request lets each bulk button act on only its own subset', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
        { id: 6, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-13', hour: 10, priority: 'normal', claimable: true },
      ]);
      mockClaimBatch({ claimed: [], skipped: [] });
      vi.mocked(scheduleApi.cancelCoverageRequestsBatch).mockResolvedValue({ data: { cancelled: [], skipped: [] } } as unknown as AxiosResponse<CoverageRequestCancelBatchResult>);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /2026-08-11/i }));
      fireEvent.click(screen.getByRole('checkbox', { name: /2026-08-13/i }));

      fireEvent.click(screen.getByRole('button', { name: /claim selected/i }));
      await waitFor(() => expect(scheduleApi.claimCoverageRequestsBatch).toHaveBeenCalledWith(7, [6]));

      fireEvent.click(screen.getByRole('checkbox', { name: /2026-08-11/i }));
      fireEvent.click(screen.getByRole('button', { name: /cancel selected/i }));
      await waitFor(() => expect(scheduleApi.cancelCoverageRequestsBatch).toHaveBeenCalledWith(7, [5]));
    });

    it('"Select all" selects both cancellable and claimable rows', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 1, requested_by_name: 'Me', date: '2026-08-11', hour: 9, priority: 'normal', claimable: false },
        { id: 6, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-13', hour: 10, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      fireEvent.click(await screen.findByRole('checkbox', { name: /select all/i }));

      expect(screen.getByRole('checkbox', { name: /2026-08-11/i })).toBeChecked();
      expect(screen.getByRole('checkbox', { name: /2026-08-13/i })).toBeChecked();
      expect(screen.getByRole('button', { name: /cancel selected/i })).toBeEnabled();
      expect(screen.getByRole('button', { name: /claim selected/i })).toBeEnabled();
    });
  });

  describe('priority', () => {
    it('shows an Optional badge on a request with priority "optional"', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'optional', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      await screen.findByText('Jane Doe');
      expect(screen.getByText(/optional/i)).toBeInTheDocument();
    });

    it('does not show an Optional badge on a normal-priority request', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} />);

      await screen.findByText('Jane Doe');
      expect(screen.queryByText(/optional/i)).not.toBeInTheDocument();
    });

    it('does not show a priority override control for a non-admin', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers={false} />);

      await screen.findByText('Jane Doe');
      expect(screen.queryByRole('button', { name: /mark optional/i })).not.toBeInTheDocument();
    });

    it('a group admin can mark a normal request optional, which refetches the list', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'normal', claimable: true },
      ]);
      vi.mocked(scheduleApi.updateCoverageRequestPriority).mockResolvedValue({} as unknown as AxiosResponse);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers />);

      await screen.findByText('Jane Doe');
      const callsBeforeUpdate = vi.mocked(scheduleApi.listCoverageRequests).mock.calls.length;
      fireEvent.click(screen.getByRole('button', { name: /mark optional/i }));

      await waitFor(() => expect(scheduleApi.updateCoverageRequestPriority).toHaveBeenCalledWith(7, 5, 'optional'));
      await waitFor(() => expect(scheduleApi.listCoverageRequests).toHaveBeenCalledTimes(callsBeforeUpdate + 1));
    });

    it('a group admin can mark an optional request normal (must-fill)', async () => {
      mockList([
        { id: 5, group_id: 7, requested_by_user_id: 2, requested_by_name: 'Jane Doe', date: '2026-08-11', hour: 9, priority: 'optional', claimable: true },
      ]);
      vi.mocked(scheduleApi.updateCoverageRequestPriority).mockResolvedValue({} as unknown as AxiosResponse);
      render(<NeedsCoverageList groupId={7} currentUserId={1} canManageMembers />);

      await screen.findByText('Jane Doe');
      fireEvent.click(screen.getByRole('button', { name: /mark normal/i }));

      await waitFor(() => expect(scheduleApi.updateCoverageRequestPriority).toHaveBeenCalledWith(7, 5, 'normal'));
    });
  });
});
