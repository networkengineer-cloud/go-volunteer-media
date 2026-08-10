import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi } from '../../api/client';
import type { ScheduleOverviewMember } from '../../api/client';
import { DAYS, HOURS, slotKey, formatHourLabel } from './scheduleGrid';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './ScheduleTab.css';
import './ScheduleOverview.css';

export interface ScheduleOverviewProps {
  groupId: number;
  totalMembers: number;
}

function memberDisplayName(member: ScheduleOverviewMember): string {
  return [member.first_name, member.last_name].filter(Boolean).join(' ') || member.username;
}

interface PopoverPosition {
  top: number;
  left: number;
}

// The popover's own rendered width isn't known before it renders (it's
// content-sized up to the 220px max-width in ScheduleOverview.css), so we
// don't try to compute an exact clamp for its edges. Instead we clamp the
// raw anchor point itself to a small margin from the viewport edges before
// the CSS `transform: translateX(-50%)` centers the popover under it. This
// isn't pixel-perfect at extreme edges (the popover can still peek past the
// margin by roughly half its width there) but keeps it from running
// substantially off-screen, which is all this bug requires.
const VIEWPORT_MARGIN = 8;

// Computes a fixed-position anchor point (bottom-center of the clicked
// cell), clamped so it won't sit at the very left/right edge of the
// viewport. Using the button's live bounding rect means this is correct
// regardless of scroll position within any clipping ancestor.
function popoverPositionFor(rect: DOMRect): PopoverPosition {
  const top = rect.bottom + 4;
  const rawLeft = rect.left + rect.width / 2;
  const left = Math.min(Math.max(rawLeft, VIEWPORT_MARGIN), window.innerWidth - VIEWPORT_MARGIN);
  return { top, left };
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
  const [popoverPosition, setPopoverPosition] = useState<PopoverPosition | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);

  // Cancels any in-flight request before starting a new one - matching
  // ScheduleTab.tsx's loadSchedule AbortController pattern - so both the
  // initial mount fetch and a user-triggered retry go through the same
  // ref-guarded controller: whichever request is superseded gets aborted,
  // and only the request still current when it finishes is allowed to
  // clear `loading`.
  const loadAbortControllerRef = useRef<AbortController | null>(null);
  const loadOverview = useCallback(() => {
    loadAbortControllerRef.current?.abort();
    const controller = new AbortController();
    loadAbortControllerRef.current = controller;

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
      .finally(() => {
        if (loadAbortControllerRef.current === controller) {
          setLoading(false);
        }
      });
  }, [groupId]);

  useEffect(() => {
    loadOverview();
  }, [loadOverview]);

  // Cancel any in-flight overview request on unmount so it can't set state
  // on an unmounted component.
  useEffect(() => {
    return () => {
      loadAbortControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (activeCellKey === null) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        setActiveCellKey(null);
        setPopoverPosition(null);
      }
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setActiveCellKey(null);
        setPopoverPosition(null);
      }
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

  // If the caller couldn't supply a member count (e.g. its own member-list
  // fetch failed or hadn't resolved yet), fall back to a denominator derived
  // from the fetched slot data itself rather than trusting `totalMembers`
  // blindly - otherwise every cell renders as tier-0 ("nobody available")
  // even when real availability data exists.
  const effectiveTotal = totalMembers > 0
    ? totalMembers
    : Math.max(0, ...Array.from(membersBySlot.values()).map(m => m.length));

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
              const tier = tierFor(members.length, effectiveTotal);
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
                    onClick={(event) => {
                      if (isActive) {
                        setActiveCellKey(null);
                        setPopoverPosition(null);
                      } else {
                        setPopoverPosition(popoverPositionFor(event.currentTarget.getBoundingClientRect()));
                        setActiveCellKey(key);
                      }
                    }}
                  />
                  {isActive && popoverPosition && (
                    <div
                      className="schedule-overview__popover"
                      ref={popoverRef}
                      style={{ top: popoverPosition.top, left: popoverPosition.left }}
                    >
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
