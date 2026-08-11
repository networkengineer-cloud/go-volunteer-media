import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi, groupsApi } from '../../api/client';
import type { GroupMember } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import ScheduleOverview from './ScheduleOverview';
import Modal from '../../components/Modal';
import RequestCoverageRangeForm from './RequestCoverageRangeForm';
import { DAYS, HOURS, slotKey, formatHourLabel } from './scheduleGrid';
import './ScheduleTab.css';

export interface ScheduleTabProps {
  groupId: number;
  canManageMembers: boolean;
  currentUserId: number;
}

const ScheduleTab: React.FC<ScheduleTabProps> = ({ groupId, canManageMembers, currentUserId }) => {
  const toast = useToast();
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
  const [selectedSlots, setSelectedSlots] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'individual' | 'overview'>('individual');
  const [showRequestRangeModal, setShowRequestRangeModal] = useState(false);

  // Cancels any in-flight request before starting a new one - matching
  // GroupPage.tsx's loadAnimals/loadActivityFeed AbortController pattern -
  // so switching the volunteer picker (or a group/prop change) in quick
  // succession can't let a slower, older response land after a newer one
  // and overwrite it with the wrong volunteer's schedule.
  const loadAbortControllerRef = useRef<AbortController | null>(null);
  const loadSchedule = useCallback(() => {
    loadAbortControllerRef.current?.abort();
    const controller = new AbortController();
    loadAbortControllerRef.current = controller;

    setLoading(true);
    setError(null);
    const request = selectedUserId === null
      ? scheduleApi.getMine(groupId, { signal: controller.signal })
      : scheduleApi.getForMember(groupId, selectedUserId, { signal: controller.signal });
    request
      .then(res => {
        setSelectedSlots(new Set(res.data.slots.map(s => slotKey(s.day_of_week, s.hour))));
      })
      .catch(err => {
        if (axios.isCancel(err)) return;
        setError('Unable to load schedule.');
      })
      .finally(() => {
        // A superseded (now-aborted) request's `finally` must not flip
        // loading back off after a newer request already set it - only the
        // request that's still current should clear it.
        if (loadAbortControllerRef.current === controller) {
          setLoading(false);
        }
      });
  }, [groupId, selectedUserId]);

  useEffect(() => {
    loadSchedule();
  }, [loadSchedule]);

  // Cancel any in-flight schedule request on unmount so it can't set state
  // on an unmounted component.
  useEffect(() => {
    return () => {
      loadAbortControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    groupsApi.getMembers(groupId)
      .then(res => setMembers(res.data))
      .catch(() => { /* picker/overview member count just won't populate; own-schedule view still works */ });
  }, [groupId]);

  const toggleSlot = (dayOfWeek: number, hour: number) => {
    setSelectedSlots(prev => {
      const next = new Set(prev);
      const key = slotKey(dayOfWeek, hour);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const handleSave = () => {
    const slots = Array.from(selectedSlots).map(key => {
      const [dayOfWeek, hour] = key.split('-').map(Number);
      return { day_of_week: dayOfWeek, hour };
    });
    setSaving(true);
    const request = selectedUserId === null
      ? scheduleApi.updateMine(groupId, slots)
      : scheduleApi.updateForMember(groupId, selectedUserId, slots);
    request
      .then(() => toast.showSuccess('Schedule saved.'))
      .catch(() => toast.showError('Failed to save schedule.'))
      .finally(() => setSaving(false));
  };

  if (loading) {
    return <SkeletonLoader variant="card" count={1} />;
  }

  if (error) {
    return <ErrorState message={error} onRetry={loadSchedule} />;
  }

  return (
    <div className="schedule-tab">
      <div className="schedule-tab__view-toggle" role="group" aria-label="Schedule view">
        <button
          type="button"
          className={viewMode === 'individual' ? 'btn-primary' : 'btn-secondary'}
          onClick={() => setViewMode('individual')}
        >
          Individual
        </button>
        <button
          type="button"
          className={viewMode === 'overview' ? 'btn-primary' : 'btn-secondary'}
          onClick={() => setViewMode('overview')}
        >
          Overview
        </button>
      </div>

      {viewMode === 'overview' ? (
        <ScheduleOverview groupId={groupId} totalMembers={members.length} currentUserId={currentUserId} />
      ) : (
        <>
          {canManageMembers && (
            <div className="schedule-tab__picker">
              <label htmlFor="schedule-member-select">Viewing schedule for</label>
              <select
                id="schedule-member-select"
                value={selectedUserId ?? ''}
                onChange={e => setSelectedUserId(e.target.value === '' ? null : Number(e.target.value))}
              >
                <option value="">My Schedule</option>
                {members.map(m => (
                  <option key={m.user_id} value={m.user_id}>
                    {[m.first_name, m.last_name].filter(Boolean).join(' ') || m.username}
                  </option>
                ))}
              </select>
            </div>
          )}

          <div className="schedule-grid" role="table" aria-label="Weekly shift schedule">
            <div className="schedule-grid__row schedule-grid__row--header" role="row">
              <div className="schedule-grid__cell schedule-grid__cell--corner" role="columnheader" />
              {DAYS.map(day => (
                <div key={day} className="schedule-grid__cell schedule-grid__cell--header" role="columnheader">
                  {day}
                </div>
              ))}
            </div>
            {HOURS.map(hour => (
              <div key={hour} className="schedule-grid__row" role="row">
                <div className="schedule-grid__cell schedule-grid__cell--header" role="rowheader">
                  {formatHourLabel(hour)}
                </div>
                {DAYS.map((_, dayOfWeek) => {
                  const key = slotKey(dayOfWeek, hour);
                  const active = selectedSlots.has(key);
                  return (
                    <button
                      key={key}
                      type="button"
                      role="cell"
                      className={`schedule-grid__slot ${active ? 'schedule-grid__slot--active' : ''}`}
                      aria-pressed={active}
                      aria-label={`${DAYS[dayOfWeek]} ${formatHourLabel(hour)}`}
                      onClick={() => toggleSlot(dayOfWeek, hour)}
                    />
                  );
                })}
              </div>
            ))}
          </div>

          <button type="button" className="btn-primary schedule-tab__save" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : 'Save Schedule'}
          </button>

          {selectedUserId === null && (
            <button
              type="button"
              className="btn-secondary schedule-tab__request-range"
              onClick={() => setShowRequestRangeModal(true)}
            >
              Request Coverage for a Date Range
            </button>
          )}
        </>
      )}

      <Modal
        isOpen={showRequestRangeModal}
        onClose={() => setShowRequestRangeModal(false)}
        title="Request Coverage for a Date Range"
      >
        <RequestCoverageRangeForm
          groupId={groupId}
          slots={Array.from(selectedSlots).map(key => {
            const [dayOfWeek, hour] = key.split('-').map(Number);
            return { day_of_week: dayOfWeek, hour };
          })}
          onSuccess={() => setShowRequestRangeModal(false)}
          onCancel={() => setShowRequestRangeModal(false)}
        />
      </Modal>
    </div>
  );
};

export default ScheduleTab;
