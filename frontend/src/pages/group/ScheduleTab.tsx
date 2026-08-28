import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi, groupsApi } from '../../api/client';
import type { GroupMember, ScheduleSlot, ScheduleCadence } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import ScheduleOverview from './ScheduleOverview';
import NeedsCoverageList from './NeedsCoverageList';
import Modal from '../../components/Modal';
import RequestCoverageRangeForm from './RequestCoverageRangeForm';
import CadenceLegend from './CadenceLegend';
import { DAYS, HOURS, slotKey, formatHourLabel, formatSlotRangeLabel, maxHourFor, nextCadence } from './scheduleGrid';
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
  const [selectedSlots, setSelectedSlots] = useState<Map<string, ScheduleCadence>>(new Map());
  const [savedSlots, setSavedSlots] = useState<ScheduleSlot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'individual' | 'overview' | 'needs-coverage'>('individual');
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
        // Legacy slots saved before the weekend hour cap was introduced may
        // exceed maxHourFor(day) (e.g. a Saturday 5pm slot from before
        // weekends capped at 3pm). Such a slot renders as a disabled,
        // non-interactive cell, so the user has no way to remove it via the
        // grid - if it stayed in the Map it would be silently resubmitted
        // (and rejected by the backend's own maxHourFor validation) on every
        // save. Drop it here instead, which self-heals the schedule back to
        // the new valid shape the next time it's loaded and saved.
        setSelectedSlots(new Map(
          res.data.slots
            .filter(s => s.hour <= maxHourFor(s.day_of_week))
            .map(s => [slotKey(s.day_of_week, s.hour), s.cadence ?? 'weekly'])
        ));
        setSavedSlots(res.data.slots);
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
    if (hour > maxHourFor(dayOfWeek)) return;
    setSelectedSlots(prev => {
      const next = new Map(prev);
      const key = slotKey(dayOfWeek, hour);
      const advanced = nextCadence(next.get(key));
      if (advanced === undefined) {
        next.delete(key);
      } else {
        next.set(key, advanced);
      }
      return next;
    });
  };

  const handleSave = () => {
    const slots = Array.from(selectedSlots.entries()).map(([key, cadence]) => {
      const [dayOfWeek, hour] = key.split('-').map(Number);
      return { day_of_week: dayOfWeek, hour, cadence };
    });
    setSaving(true);
    const request = selectedUserId === null
      ? scheduleApi.updateMine(groupId, slots)
      : scheduleApi.updateForMember(groupId, selectedUserId, slots);
    request
      .then(() => {
        toast.showSuccess('Schedule saved.');
        setSavedSlots(slots.map(s => ({ day_of_week: s.day_of_week, hour: s.hour, cadence: s.cadence })));
      })
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
          className={`schedule-tab__view-toggle-btn ${viewMode === 'individual' ? 'schedule-tab__view-toggle-btn--active' : ''}`}
          onClick={() => setViewMode('individual')}
        >
          Individual
        </button>
        <button
          type="button"
          className={`schedule-tab__view-toggle-btn ${viewMode === 'overview' ? 'schedule-tab__view-toggle-btn--active' : ''}`}
          onClick={() => setViewMode('overview')}
        >
          Overview
        </button>
        <button
          type="button"
          className={`schedule-tab__view-toggle-btn ${viewMode === 'needs-coverage' ? 'schedule-tab__view-toggle-btn--active' : ''}`}
          onClick={() => setViewMode('needs-coverage')}
        >
          Needs Coverage
        </button>
      </div>

      {viewMode === 'overview' && (
        <ScheduleOverview groupId={groupId} totalMembers={members.length} currentUserId={currentUserId} />
      )}

      {viewMode === 'needs-coverage' && (
        <NeedsCoverageList groupId={groupId} currentUserId={currentUserId} canManageMembers={canManageMembers} />
      )}

      {viewMode === 'individual' && (
        <>
          {selectedUserId === null && (
            <button
              type="button"
              className="btn-secondary schedule-tab__request-range"
              onClick={() => setShowRequestRangeModal(true)}
            >
              Request Coverage for a Date Range
            </button>
          )}

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

          <CadenceLegend />

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
                  const disabled = hour > maxHourFor(dayOfWeek);
                  if (disabled) {
                    return (
                      <div
                        key={key}
                        role="presentation"
                        className="schedule-grid__slot schedule-grid__slot--disabled"
                      />
                    );
                  }
                  const cadence = selectedSlots.get(key);
                  const isTerminal = hour === maxHourFor(dayOfWeek);
                  const cadenceSuffix = cadence === 'biweekly_a' ? ' (Week A)' : cadence === 'biweekly_b' ? ' (Week B)' : '';
                  const label = `${DAYS[dayOfWeek]} ${formatSlotRangeLabel(dayOfWeek, hour)}${cadenceSuffix}`;
                  const cadenceClass = cadence ? ` schedule-grid__slot--${cadence.replace('_', '-')}` : '';
                  return (
                    <button
                      key={key}
                      type="button"
                      role="cell"
                      className={`schedule-grid__slot${cadenceClass}${isTerminal ? ' schedule-grid__slot--terminal' : ''}`}
                      aria-pressed={!!cadence}
                      aria-label={label}
                      onClick={() => toggleSlot(dayOfWeek, hour)}
                    >
                      {cadence === 'biweekly_a' && <span className="schedule-grid__cadence-tag">A</span>}
                      {cadence === 'biweekly_b' && <span className="schedule-grid__cadence-tag">B</span>}
                    </button>
                  );
                })}
              </div>
            ))}
          </div>

          <button type="button" className="btn-primary schedule-tab__save" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving…' : 'Save Schedule'}
          </button>
        </>
      )}

      <Modal
        isOpen={showRequestRangeModal}
        onClose={() => setShowRequestRangeModal(false)}
        title="Request Coverage for a Date Range"
      >
        <RequestCoverageRangeForm
          groupId={groupId}
          slots={savedSlots}
          onCancel={() => setShowRequestRangeModal(false)}
        />
      </Modal>
    </div>
  );
};

export default ScheduleTab;
