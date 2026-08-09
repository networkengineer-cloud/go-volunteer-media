import React, { useCallback, useEffect, useState } from 'react';
import { scheduleApi, groupsApi } from '../../api/client';
import type { GroupMember } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './ScheduleTab.css';

export interface ScheduleTabProps {
  groupId: number;
  canManageMembers: boolean;
}

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const HOURS = [8, 9, 10, 11, 12, 13, 14, 15, 16, 17];

function slotKey(dayOfWeek: number, hour: number): string {
  return `${dayOfWeek}-${hour}`;
}

function formatHourLabel(hour: number): string {
  const period = hour < 12 ? 'AM' : 'PM';
  const displayHour = hour % 12 === 0 ? 12 : hour % 12;
  return `${displayHour}:00 ${period}`;
}

const ScheduleTab: React.FC<ScheduleTabProps> = ({ groupId, canManageMembers }) => {
  const toast = useToast();
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
  const [selectedSlots, setSelectedSlots] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const loadSchedule = useCallback(() => {
    setLoading(true);
    setError(null);
    const request = selectedUserId === null
      ? scheduleApi.getMine(groupId)
      : scheduleApi.getForMember(groupId, selectedUserId);
    request
      .then(res => {
        setSelectedSlots(new Set(res.data.slots.map(s => slotKey(s.day_of_week, s.hour))));
      })
      .catch(() => setError('Unable to load schedule.'))
      .finally(() => setLoading(false));
  }, [groupId, selectedUserId]);

  useEffect(() => {
    loadSchedule();
  }, [loadSchedule]);

  useEffect(() => {
    if (!canManageMembers) return;
    groupsApi.getMembers(groupId)
      .then(res => setMembers(res.data))
      .catch(() => { /* picker just won't populate; own-schedule view still works */ });
  }, [groupId, canManageMembers]);

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
    </div>
  );
};

export default ScheduleTab;
