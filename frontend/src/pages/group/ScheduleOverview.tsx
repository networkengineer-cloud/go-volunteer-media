import React, { useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi } from '../../api/client';
import type { ScheduleOverviewMember } from '../../api/client';
import { DAYS, HOURS, slotKey, formatHourLabel } from './scheduleGrid';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './ScheduleOverview.css';

export interface ScheduleOverviewProps {
  groupId: number;
  totalMembers: number;
}

function memberDisplayName(member: ScheduleOverviewMember): string {
  return [member.first_name, member.last_name].filter(Boolean).join(' ') || member.username;
}

// Buckets a slot's availability into one of 5 shading tiers based on what
// fraction of the group is free, so the heatmap reads consistently
// regardless of group size: 0%, 1-25%, 26-50%, 51-75%, 76-100%.
function tierFor(count: number, totalMembers: number): number {
  if (count === 0 || totalMembers === 0) return 0;
  const fraction = count / totalMembers;
  if (fraction <= 0.25) return 1;
  if (fraction <= 0.5) return 2;
  if (fraction <= 0.75) return 3;
  return 4;
}

const ScheduleOverview: React.FC<ScheduleOverviewProps> = ({ groupId, totalMembers }) => {
  const [membersBySlot, setMembersBySlot] = useState<Map<string, ScheduleOverviewMember[]>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeCellKey, setActiveCellKey] = useState<string | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);

  const loadOverview = () => {
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    scheduleApi.getOverview(groupId, { signal: controller.signal })
      .then(res => {
        const map = new Map<string, ScheduleOverviewMember[]>();
        res.data.slots.forEach(slot => {
          map.set(slotKey(slot.day_of_week, slot.hour), slot.members);
        });
        setMembersBySlot(map);
      })
      .catch(err => {
        if (axios.isCancel(err)) return;
        setError('Unable to load schedule overview.');
      })
      .finally(() => setLoading(false));
    return controller;
  };

  useEffect(() => {
    const controller = loadOverview();
    return () => controller.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupId]);

  useEffect(() => {
    if (activeCellKey === null) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        setActiveCellKey(null);
      }
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setActiveCellKey(null);
    };
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [activeCellKey]);

  if (loading) {
    return <SkeletonLoader variant="card" count={1} />;
  }

  if (error) {
    return <ErrorState message={error} onRetry={loadOverview} />;
  }

  return (
    <div className="schedule-overview">
      <div className="schedule-grid" role="table" aria-label="Weekly availability overview">
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
              const members = membersBySlot.get(key) ?? [];
              const tier = tierFor(members.length, totalMembers);
              const label = `${DAYS[dayOfWeek]} ${formatHourLabel(hour)}, ${members.length} available`;
              const isActive = activeCellKey === key;

              if (members.length === 0) {
                return (
                  <div
                    key={key}
                    role="cell"
                    aria-label={label}
                    className={`schedule-grid__slot schedule-grid__slot--tier-${tier}`}
                  />
                );
              }

              return (
                <div key={key} className="schedule-overview__cell-wrapper">
                  <button
                    type="button"
                    role="cell"
                    aria-label={label}
                    className={`schedule-grid__slot schedule-grid__slot--tier-${tier}`}
                    onClick={() => setActiveCellKey(isActive ? null : key)}
                  />
                  {isActive && (
                    <div className="schedule-overview__popover" ref={popoverRef}>
                      <ul>
                        {members.map(member => (
                          <li key={member.user_id}>{memberDisplayName(member)}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
};

export default ScheduleOverview;
