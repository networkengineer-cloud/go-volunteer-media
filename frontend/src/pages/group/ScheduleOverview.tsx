import React, { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { scheduleApi } from '../../api/client';
import type { ScheduleOverviewMember } from '../../api/client';
import { useToast } from '../../hooks/useToast';
import { DAYS, HOURS, slotKey, formatHourLabel, currentWeekStart } from './scheduleGrid';
import CadenceLegend from './CadenceLegend';
import SkeletonLoader from '../../components/SkeletonLoader';
import ErrorState from '../../components/ErrorState';
import './ScheduleTab.css';
import './ScheduleOverview.css';

export interface ScheduleOverviewProps {
  groupId: number;
  totalMembers: number;
  currentUserId: number;
}

function memberDisplayName(member: ScheduleOverviewMember): string {
  return [member.first_name, member.last_name].filter(Boolean).join(' ') || member.username;
}

// Compact name shown directly in a cell (e.g. "Jane D.") so who's scheduled
// is visible at a glance, without the click the full popover requires.
// Deliberately distinct from memberDisplayName's full "First Last" - keeps
// cells narrow and avoids duplicate-text ambiguity with the popover.
function shortDisplayName(member: ScheduleOverviewMember): string {
  if (!member.first_name) return member.username;
  return member.last_name ? `${member.first_name} ${member.last_name.charAt(0)}.` : member.first_name;
}

// Caps how many names render inline per cell before collapsing the rest
// into a "+N more" indicator, so a heavily-staffed slot can't blow out the
// row height for the whole week.
const MAX_VISIBLE_NAMES = 3;

interface PopoverPosition {
  top: number;
  left: number;
}

const VIEWPORT_MARGIN = 8;

// The popover is horizontally centered on the anchor point via the CSS
// `transform: translateX(-50%)`, so its rendered left/right edges sit
// `POPOVER_WIDTH / 2` away from whatever `left` we compute here. We clamp
// against the *widest* it can ever render - `max-width` from
// ScheduleOverview.css (140px min-width / 220px max-width; content-sized
// in between) - so the clamp holds even for popovers with long member
// lists that hit the max-width.
const POPOVER_WIDTH = 220;
const POPOVER_HALF_WIDTH = POPOVER_WIDTH / 2;

// The popover's member list is unbounded in length, but
// `.schedule-overview__popover` in ScheduleOverview.css caps the rendered
// box at `max-height: 220px` (with `overflow-y: auto` so long lists scroll
// internally instead of growing the box). Because of that cap, this is no
// longer an estimate of a variable height - it's the guaranteed maximum,
// so the flip decision below is correct for every member count, not just
// typical ones.
//
// MUST match `.schedule-overview__popover`'s `max-height` in
// ScheduleOverview.css exactly. `* { box-sizing: border-box }` (index.css)
// applies globally, so that 220px already includes the popover's own
// padding/border - nothing to add here. Nothing enforces this link
// automatically; if the CSS value changes, update this constant too.
const ESTIMATED_POPOVER_HEIGHT = 220;

// Computes a fixed-position anchor point for the popover, derived from the
// clicked cell's live bounding rect (so it's correct regardless of scroll
// position within any clipping ancestor). Clamps both axes against the
// viewport:
//  - Vertically, flips the popover above the cell when there isn't enough
//    room below for the estimated popover height.
//  - Horizontally, accounts for the popover's own (max) rendered width when
//    clamping, not just the raw anchor point, since the anchor is the
//    horizontal *center* of the popover, not its edge.
function popoverPositionFor(rect: DOMRect): PopoverPosition {
  const spaceBelow = window.innerHeight - rect.bottom - 4;
  const top = spaceBelow >= ESTIMATED_POPOVER_HEIGHT
    ? rect.bottom + 4
    : Math.max(VIEWPORT_MARGIN, rect.top - 4 - ESTIMATED_POPOVER_HEIGHT);

  const rawLeft = rect.left + rect.width / 2;
  const minLeft = VIEWPORT_MARGIN + POPOVER_HALF_WIDTH;
  const maxLeft = window.innerWidth - VIEWPORT_MARGIN - POPOVER_HALF_WIDTH;
  const left = Math.min(Math.max(rawLeft, minLeft), maxLeft);
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

// dateForWeekStart returns the ISO date (YYYY-MM-DD) for the given offset
// (0-6) within the week starting at weekStart. Computed via UTC-anchored
// Date arithmetic (rather than local-time Date parsing) so DST transitions
// in the viewer's timezone can never shift the result by a day.
function dateForWeekStart(weekStart: string, dayOfWeek: number): string {
  const [y, m, d] = weekStart.split('-').map(Number);
  const start = new Date(Date.UTC(y, m - 1, d));
  start.setUTCDate(start.getUTCDate() + dayOfWeek);
  return start.toISOString().slice(0, 10);
}

function addDays(isoDate: string, days: number): string {
  const [y, m, d] = isoDate.split('-').map(Number);
  const date = new Date(Date.UTC(y, m - 1, d));
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function formatWeekLabel(weekStart: string): string {
  const start = new Date(`${weekStart}T00:00:00Z`);
  const end = new Date(`${addDays(weekStart, 6)}T00:00:00Z`);
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric', timeZone: 'UTC' };
  return `Week of ${start.toLocaleDateString(undefined, opts)} – ${end.toLocaleDateString(undefined, opts)}`;
}

// todayIso returns "today" (UTC calendar date, matching the backend's
// same-day-or-later check in CreateCoverageRequest) as an ISO YYYY-MM-DD
// string, for comparing against slot dates.
function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

const ScheduleOverview: React.FC<ScheduleOverviewProps> = ({ groupId, totalMembers, currentUserId }) => {
  const toast = useToast();
  const [weekStart, setWeekStart] = useState<string>(currentWeekStart());
  const [membersBySlot, setMembersBySlot] = useState<Map<string, ScheduleOverviewMember[]>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeCellKey, setActiveCellKey] = useState<string | null>(null);
  const [popoverPosition, setPopoverPosition] = useState<PopoverPosition | null>(null);
  const [busyRequestId, setBusyRequestId] = useState<number | null>(null);
  // Guards handleRequestCoverage the same way busyRequestId guards
  // handleClaim/handleCancelRequest: keyed by the (date, hour) slot key so
  // the "Request coverage" button for the in-flight cell disables itself
  // mid-request, preventing a double-click from firing two concurrent
  // POSTs (which the DB-level unique index would otherwise let race).
  const [busyRequestSlotKey, setBusyRequestSlotKey] = useState<string | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);

  // Cancels any in-flight request before starting a new one - matching
  // ScheduleTab.tsx's loadSchedule AbortController pattern - so both the
  // initial mount fetch and a user-triggered retry (or week navigation) go
  // through the same ref-guarded controller: whichever request is
  // superseded gets aborted, and only the request still current when it
  // finishes is allowed to clear `loading`.
  const loadAbortControllerRef = useRef<AbortController | null>(null);
  const loadOverview = useCallback(() => {
    loadAbortControllerRef.current?.abort();
    const controller = new AbortController();
    loadAbortControllerRef.current = controller;

    setLoading(true);
    setError(null);
    scheduleApi.getOverview(groupId, { weekStart, signal: controller.signal })
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
  }, [groupId, weekStart]);

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

  // Changing weeks invalidates whatever cell was open (its data belongs to
  // the previous week), so close the popover to avoid showing stale
  // members/actions against the wrong date.
  useEffect(() => {
    setActiveCellKey(null);
    setPopoverPosition(null);
  }, [weekStart]);

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

  const handleClaim = (requestId: number) => {
    setBusyRequestId(requestId);
    scheduleApi.claimCoverageRequest(groupId, requestId)
      .then(() => {
        toast.showSuccess('Shift claimed.');
        setActiveCellKey(null);
        setPopoverPosition(null);
        loadOverview();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to claim shift.');
        loadOverview();
      })
      .finally(() => setBusyRequestId(null));
  };

  const handleCancelRequest = (requestId: number) => {
    setBusyRequestId(requestId);
    scheduleApi.cancelCoverageRequest(groupId, requestId)
      .then(() => {
        toast.showSuccess('Coverage request cancelled.');
        setActiveCellKey(null);
        setPopoverPosition(null);
        loadOverview();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to cancel coverage request.');
        loadOverview();
      })
      .finally(() => setBusyRequestId(null));
  };

  const handleRequestCoverage = (slotKeyValue: string, date: string, hour: number) => {
    setBusyRequestSlotKey(slotKeyValue);
    scheduleApi.createCoverageRequest(groupId, { date, hour })
      .then(() => {
        toast.showSuccess('Coverage requested.');
        setActiveCellKey(null);
        setPopoverPosition(null);
        loadOverview();
      })
      .catch(err => {
        toast.showError(err.response?.data?.error || 'Failed to request coverage.');
        loadOverview();
      })
      .finally(() => setBusyRequestSlotKey(null));
  };

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

  const today = todayIso();

  return (
    <div className="schedule-overview">
      <CadenceLegend referenceWeekStart={weekStart} />

      <div className="schedule-overview__week-nav">
        <button type="button" className="schedule-overview__week-nav-btn" onClick={() => setWeekStart(addDays(weekStart, -7))} aria-label="Previous week">
          ◀
        </button>
        <span>{formatWeekLabel(weekStart)}</span>
        <button type="button" className="schedule-overview__week-nav-btn" onClick={() => setWeekStart(addDays(weekStart, 7))} aria-label="Next week">
          ▶
        </button>
      </div>

      <div className="schedule-grid" role="table" aria-label="Weekly availability overview">
        <div className="schedule-grid__row schedule-grid__row--header" role="row">
          <div className="schedule-grid__cell schedule-grid__cell--corner" role="columnheader" />
          {DAYS.map((day, dayOfWeek) => (
            <div key={day} className="schedule-grid__cell schedule-grid__cell--header" role="columnheader">
              {day} {new Date(`${dateForWeekStart(weekStart, dayOfWeek)}T00:00:00Z`).getUTCDate()}
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
              const date = dateForWeekStart(weekStart, dayOfWeek);
              const needsCoverage = members.some(m => m.status === 'needs_coverage');
              const label = `${DAYS[dayOfWeek]} ${formatHourLabel(hour)}, ${members.length} available${needsCoverage ? ', needs coverage' : ''}`;
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

              const visibleMembers = members.slice(0, MAX_VISIBLE_NAMES);
              const overflowCount = members.length - visibleMembers.length;

              return (
                <div key={key} className="schedule-overview__cell-wrapper">
                  <button
                    type="button"
                    role="cell"
                    aria-label={label}
                    className={`schedule-grid__slot schedule-grid__slot--tier-${tier}${needsCoverage ? ' schedule-grid__slot--needs-coverage' : ''}`}
                    onClick={(event) => {
                      if (isActive) {
                        setActiveCellKey(null);
                        setPopoverPosition(null);
                      } else {
                        setPopoverPosition(popoverPositionFor(event.currentTarget.getBoundingClientRect()));
                        setActiveCellKey(key);
                      }
                    }}
                  >
                    <span className="schedule-overview__names">
                      {visibleMembers.map(member => (
                        <span key={member.user_id} className="schedule-overview__name">
                          {shortDisplayName(member)}
                          {member.cadence === 'biweekly_a' && <span className="schedule-grid__cadence-tag"> A</span>}
                          {member.cadence === 'biweekly_b' && <span className="schedule-grid__cadence-tag"> B</span>}
                          {member.status === 'needs_coverage' && (
                            <span className="schedule-overview__tag" aria-hidden="true"> ⚠</span>
                          )}
                        </span>
                      ))}
                      {overflowCount > 0 && (
                        <span className="schedule-overview__name schedule-overview__name--more">
                          +{overflowCount} more
                        </span>
                      )}
                    </span>
                  </button>
                  {isActive && popoverPosition && (
                    <div
                      className="schedule-overview__popover"
                      ref={popoverRef}
                      style={{ top: popoverPosition.top, left: popoverPosition.left }}
                    >
                      <ul>
                        {members.map(member => (
                          <li key={member.user_id} className="schedule-overview__popover-row">
                            <span>
                              {memberDisplayName(member)}
                              {member.cadence === 'biweekly_a' && <span className="schedule-grid__cadence-tag"> A</span>}
                              {member.cadence === 'biweekly_b' && <span className="schedule-grid__cadence-tag"> B</span>}
                              {member.status === 'needs_coverage' && <span className="schedule-overview__tag"> needs coverage</span>}
                              {member.status === 'covering' && <span className="schedule-overview__tag"> covering</span>}
                            </span>
                            {member.status === 'needs_coverage' && member.coverage_request_id !== undefined && member.user_id !== currentUserId && (
                              <button
                                type="button"
                                className="btn-secondary schedule-overview__action"
                                disabled={busyRequestId !== null || member.conflict}
                                title={member.conflict ? 'You already have a conflicting shift at this time' : undefined}
                                onClick={() => handleClaim(member.coverage_request_id as number)}
                              >
                                Claim
                              </button>
                            )}
                            {member.user_id === currentUserId && member.status === 'normal' && date >= today && (
                              <button
                                type="button"
                                className="btn-secondary schedule-overview__action"
                                disabled={busyRequestSlotKey === key}
                                onClick={() => handleRequestCoverage(key, date, hour)}
                              >
                                Request coverage
                              </button>
                            )}
                            {member.user_id === currentUserId && member.status === 'needs_coverage' && member.coverage_request_id !== undefined && (
                              <button
                                type="button"
                                className="btn-secondary schedule-overview__action"
                                disabled={busyRequestId === member.coverage_request_id}
                                onClick={() => handleCancelRequest(member.coverage_request_id as number)}
                              >
                                Cancel request
                              </button>
                            )}
                          </li>
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
