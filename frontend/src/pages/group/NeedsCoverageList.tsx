import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi } from '../../api/client';
import type { CoverageRequestListItem } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { formatHourLabel } from './scheduleGrid';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './NeedsCoverageList.css';

export interface NeedsCoverageListProps {
  groupId: number;
  currentUserId: number;
}

function formatDateLabel(isoDate: string): string {
  const opts: Intl.DateTimeFormatOptions = { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' };
  return new Date(`${isoDate}T00:00:00Z`).toLocaleDateString(undefined, opts);
}

const NeedsCoverageList: React.FC<NeedsCoverageListProps> = ({ groupId, currentUserId }) => {
  const toast = useToast();
  const [items, setItems] = useState<CoverageRequestListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyRequestId, setBusyRequestId] = useState<number | null>(null);

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
    <ul className="needs-coverage-list">
      {items.map(item => (
        <li key={item.id} className="needs-coverage-list__row">
          <span className="needs-coverage-list__details">
            <span className="needs-coverage-list__date">{formatDateLabel(item.date)}</span>
            {' at '}
            <span className="needs-coverage-list__hour">{formatHourLabel(item.hour)}</span>
            {' — '}
            <span className="needs-coverage-list__name">{item.requested_by_name}</span>
          </span>
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
  );
};

export default NeedsCoverageList;
