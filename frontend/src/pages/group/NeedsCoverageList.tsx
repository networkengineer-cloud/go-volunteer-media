import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi } from '../../api/client';
import type { CoverageRequestListItem, CoverageRequestPriority } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { formatSlotRangeLabel } from './scheduleGrid';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './NeedsCoverageList.css';

export interface NeedsCoverageListProps {
  groupId: number;
  currentUserId: number;
  canManageMembers?: boolean;
}

function formatDateLabel(isoDate: string): string {
  const opts: Intl.DateTimeFormatOptions = { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' };
  return new Date(`${isoDate}T00:00:00Z`).toLocaleDateString(undefined, opts);
}

// CoverageRequestListItem only carries a `date` (not a day_of_week), so the
// day-of-week needed by formatSlotRangeLabel (to know whether this item's
// hour is a day's terminal 90-min slot) is derived here from that date.
function dayOfWeekFromIso(isoDate: string): number {
  return new Date(`${isoDate}T00:00:00Z`).getUTCDay();
}

const NeedsCoverageList: React.FC<NeedsCoverageListProps> = ({ groupId, currentUserId, canManageMembers = false }) => {
  const toast = useToast();
  const [items, setItems] = useState<CoverageRequestListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyRequestId, setBusyRequestId] = useState<number | null>(null);
  const [checkedIds, setCheckedIds] = useState<Set<number>>(new Set());
  const [cancelling, setCancelling] = useState(false);
  const [claiming, setClaiming] = useState(false);
  const [busyPriorityId, setBusyPriorityId] = useState<number | null>(null);

  // Cancels any in-flight request before starting a new one, matching the
  // AbortController pattern already used by ScheduleTab/ScheduleOverview's
  // own loaders, so a superseded fetch (e.g. a rapid claim-then-refetch)
  // can't land after a newer one and overwrite it.
  const loadAbortControllerRef = useRef<AbortController | null>(null);
  const loadRequests = useCallback(() => {
    loadAbortControllerRef.current?.abort();
    const controller = new AbortController();
    loadAbortControllerRef.current = controller;

    setLoading(true);
    setError(null);
    scheduleApi.listCoverageRequests(groupId, { signal: controller.signal })
      .then(res => setItems(res.data))
      .catch(err => {
        if (axios.isCancel(err)) return;
        setError('Unable to load shifts that need coverage.');
      })
      .finally(() => {
        if (loadAbortControllerRef.current === controller) {
          setLoading(false);
        }
      });
  }, [groupId]);

  useEffect(() => {
    loadRequests();
  }, [loadRequests]);

  useEffect(() => {
    return () => {
      loadAbortControllerRef.current?.abort();
    };
  }, []);

  // Drop any checked id that no longer appears in the freshly loaded list
  // (e.g. someone else claimed it between page loads), so a stale id never
  // rides along in the next cancel-selected submission.
  useEffect(() => {
    setCheckedIds(prev => {
      const validIds = new Set(items.map(i => i.id));
      const next = new Set([...prev].filter(id => validIds.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [items]);

  const handleClaim = (requestId: number) => {
    setBusyRequestId(requestId);
    scheduleApi.claimCoverageRequest(groupId, requestId)
      .then(() => {
        toast.showSuccess('Shift claimed.');
        loadRequests();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to claim shift.');
        loadRequests();
      })
      .finally(() => setBusyRequestId(null));
  };

  const handleTogglePriority = (item: CoverageRequestListItem) => {
    const nextPriority: CoverageRequestPriority = item.priority === 'optional' ? 'normal' : 'optional';
    setBusyPriorityId(item.id);
    scheduleApi.updateCoverageRequestPriority(groupId, item.id, nextPriority)
      .then(() => {
        toast.showSuccess(nextPriority === 'optional' ? 'Marked optional.' : 'Marked normal (must-fill).');
        loadRequests();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to update priority.');
      })
      .finally(() => setBusyPriorityId(null));
  };

  const isCancellable = (item: CoverageRequestListItem) => item.requested_by_user_id === currentUserId || canManageMembers;
  const isClaimable = (item: CoverageRequestListItem) => item.requested_by_user_id !== currentUserId && item.claimable;

  const toggleChecked = (id: number) => {
    setCheckedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const cancellableIds = items.filter(isCancellable).map(i => i.id);
  const claimableIds = items.filter(isClaimable).map(i => i.id);
  const selectableIds = Array.from(new Set([...cancellableIds, ...claimableIds]));
  const allChecked = selectableIds.length > 0 && selectableIds.every(id => checkedIds.has(id));
  const toggleAll = () => {
    setCheckedIds(allChecked ? new Set() : new Set(selectableIds));
  };

  const checkedCancellableIds = cancellableIds.filter(id => checkedIds.has(id));
  const checkedClaimableIds = claimableIds.filter(id => checkedIds.has(id));

  const handleCancelSelected = () => {
    if (checkedCancellableIds.length === 0) return;
    setCancelling(true);
    scheduleApi.cancelCoverageRequestsBatch(groupId, checkedCancellableIds)
      .then(res => {
        const { cancelled, skipped } = res.data;
        if (cancelled.length > 0) {
          toast.showSuccess(`Cancelled ${cancelled.length} request${cancelled.length === 1 ? '' : 's'}.`);
        }
        if (skipped.length > 0) {
          toast.showError(`${skipped.length} request${skipped.length === 1 ? '' : 's'} could not be cancelled.`);
        }
        setCheckedIds(new Set());
        loadRequests();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to cancel requests.');
      })
      .finally(() => setCancelling(false));
  };

  const handleClaimSelected = () => {
    if (checkedClaimableIds.length === 0) return;
    setClaiming(true);
    scheduleApi.claimCoverageRequestsBatch(groupId, checkedClaimableIds)
      .then(res => {
        const { claimed, skipped } = res.data;
        if (claimed.length > 0) {
          toast.showSuccess(`Claimed ${claimed.length} request${claimed.length === 1 ? '' : 's'}.`);
        }
        if (skipped.length > 0) {
          toast.showError(`${skipped.length} request${skipped.length === 1 ? '' : 's'} could not be claimed.`);
        }
        setCheckedIds(new Set());
        loadRequests();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to claim requests.');
      })
      .finally(() => setClaiming(false));
  };

  if (loading) {
    return <SkeletonLoader variant="card" count={1} />;
  }

  if (error) {
    return <ErrorState message={error} onRetry={loadRequests} />;
  }

  if (items.length === 0) {
    return <p className="needs-coverage-list__empty">No shifts currently need coverage.</p>;
  }

  return (
    <div className="needs-coverage-list-wrapper">
      {selectableIds.length > 0 && (
        <div className="needs-coverage-list__bulk-bar">
          <label>
            <input type="checkbox" checked={allChecked} onChange={toggleAll} aria-label="Select all requests" />
            Select all
          </label>
          <button
            type="button"
            className="btn-secondary needs-coverage-list__claim-selected"
            disabled={checkedClaimableIds.length === 0 || claiming}
            onClick={handleClaimSelected}
          >
            Claim selected ({checkedClaimableIds.length})
          </button>
          <button
            type="button"
            className="btn-secondary needs-coverage-list__cancel-selected"
            disabled={checkedCancellableIds.length === 0 || cancelling}
            onClick={handleCancelSelected}
          >
            Cancel selected ({checkedCancellableIds.length})
          </button>
        </div>
      )}
      <ul className="needs-coverage-list">
        {items.map(item => (
          <li key={item.id} className="needs-coverage-list__row">
            <span className="needs-coverage-list__details">
              {(isCancellable(item) || isClaimable(item)) && (
                <input
                  type="checkbox"
                  checked={checkedIds.has(item.id)}
                  onChange={() => toggleChecked(item.id)}
                  aria-label={`Select coverage request for ${item.date} at ${formatSlotRangeLabel(dayOfWeekFromIso(item.date), item.hour)}`}
                />
              )}
              <span className="needs-coverage-list__date">{formatDateLabel(item.date)}</span>
              {' at '}
              <span className="needs-coverage-list__hour">{formatSlotRangeLabel(dayOfWeekFromIso(item.date), item.hour)}</span>
              {' — '}
              <span className="needs-coverage-list__name">{item.requested_by_name}</span>
              {item.priority === 'optional' && (
                <span className="needs-coverage-list__priority-badge"> Optional</span>
              )}
            </span>
            {canManageMembers && (
              <button
                type="button"
                className="btn-secondary needs-coverage-list__priority-toggle"
                disabled={busyPriorityId === item.id}
                onClick={() => handleTogglePriority(item)}
              >
                {item.priority === 'optional' ? 'Mark normal' : 'Mark optional'}
              </button>
            )}
            {item.requested_by_user_id !== currentUserId && (
              <button
                type="button"
                className="btn-secondary needs-coverage-list__claim"
                disabled={!item.claimable || busyRequestId === item.id}
                title={!item.claimable ? 'You already have a conflicting shift at this time' : undefined}
                onClick={() => handleClaim(item.id)}
              >
                Claim
              </button>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
};

export default NeedsCoverageList;
