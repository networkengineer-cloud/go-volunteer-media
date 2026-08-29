import React, { useEffect, useRef, useState } from 'react';
import './DateRangePicker.css';

export interface DateRangePickerProps {
  startDate: string;
  endDate: string;
  onChange: (startDate: string, endDate: string) => void;
  min?: string;
}

const WEEKDAY_HEADERS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];

function parseIso(iso: string): { y: number; m: number; d: number } {
  const [y, m, d] = iso.split('-').map(Number);
  return { y, m: m - 1, d };
}

function isoOf(y: number, m: number, d: number): string {
  return `${y}-${String(m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
}

function startOfMonthIso(iso: string): string {
  const { y, m } = parseIso(iso);
  return isoOf(y, m, 1);
}

function addMonths(iso: string, delta: number): string {
  const { y, m } = parseIso(iso);
  const date = new Date(Date.UTC(y, m + delta, 1));
  return isoOf(date.getUTCFullYear(), date.getUTCMonth(), 1);
}

function daysInMonth(iso: string): number {
  const { y, m } = parseIso(iso);
  return new Date(Date.UTC(y, m + 1, 0)).getUTCDate();
}

function firstWeekdayOfMonth(iso: string): number {
  const { y, m } = parseIso(iso);
  return new Date(Date.UTC(y, m, 1)).getUTCDay();
}

function monthLabel(iso: string): string {
  const { y, m } = parseIso(iso);
  return new Date(Date.UTC(y, m, 1)).toLocaleDateString(undefined, { month: 'long', year: 'numeric', timeZone: 'UTC' });
}

function formatDisplayDate(iso: string): string {
  const { y, m, d } = parseIso(iso);
  return new Date(Date.UTC(y, m, d)).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' });
}

function formatDayLabel(iso: string): string {
  const { y, m, d } = parseIso(iso);
  return new Date(Date.UTC(y, m, d)).toLocaleDateString(undefined, { month: 'long', day: 'numeric', year: 'numeric', timeZone: 'UTC' });
}

function todayIsoFallback(): string {
  return new Date().toISOString().slice(0, 10);
}

interface PopoverPosition {
  top: number;
  left: number;
}

const VIEWPORT_MARGIN = 8;
const POPOVER_WIDTH = 260;
// The popover's own CSS caps it at this height (with overflow-y: auto), so
// this doubles as the guaranteed maximum for the flip-above decision below,
// not just an estimate - same technique ScheduleOverview.tsx's own popover
// positioning uses for the same reason (see its ESTIMATED_POPOVER_HEIGHT).
const POPOVER_MAX_HEIGHT = 340;

// Anchors the popover to the trigger button's live viewport position rather
// than relying on `position: absolute` inside a `position: relative`
// ancestor. This component can be rendered inside Modal, whose
// `.modal__content` is `overflow-y: auto` - an absolutely-positioned
// popover there doesn't expand the modal's own (content-sized) height, so
// it was rendering squeezed into a few pixels of scrollable sliver on
// mobile instead of appearing below the field. `position: fixed` computed
// from getBoundingClientRect() escapes that clipping ancestor entirely,
// the same fix ScheduleOverview.tsx's member popover already uses for an
// identical class of problem.
function popoverPositionFor(rect: DOMRect): PopoverPosition {
  const spaceBelow = window.innerHeight - rect.bottom - 4;
  const top = spaceBelow >= POPOVER_MAX_HEIGHT
    ? rect.bottom + 4
    : Math.max(VIEWPORT_MARGIN, rect.top - 4 - POPOVER_MAX_HEIGHT);

  const maxLeft = window.innerWidth - VIEWPORT_MARGIN - POPOVER_WIDTH;
  const left = Math.min(Math.max(rect.left, VIEWPORT_MARGIN), Math.max(VIEWPORT_MARGIN, maxLeft));
  return { top, left };
}

// DateRangePicker lets both ends of a date range be chosen from one popover
// (two clicks on a calendar) instead of two separate native date inputs.
// The first click sets the start and clears any existing end, so a click
// after a complete range always begins a fresh selection rather than
// silently editing the old one. A click before the current start restarts
// the selection at that earlier day instead of trying to "complete" a
// backwards range.
const DateRangePicker: React.FC<DateRangePickerProps> = ({ startDate, endDate, onChange, min }) => {
  const [open, setOpen] = useState(false);
  const [viewMonth, setViewMonth] = useState(() => startOfMonthIso(startDate || min || todayIsoFallback()));
  const [popoverPosition, setPopoverPosition] = useState<PopoverPosition | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [open]);

  const toggleOpen = () => {
    if (!open) {
      setViewMonth(startOfMonthIso(startDate || min || todayIsoFallback()));
      if (triggerRef.current) {
        setPopoverPosition(popoverPositionFor(triggerRef.current.getBoundingClientRect()));
      }
    }
    setOpen(prev => !prev);
  };

  const handleDayClick = (iso: string) => {
    if (!startDate || endDate) {
      onChange(iso, '');
      return;
    }
    if (iso < startDate) {
      onChange(iso, '');
      return;
    }
    onChange(startDate, iso);
    setOpen(false);
  };

  const triggerLabel = startDate && endDate
    ? `${formatDisplayDate(startDate)} – ${formatDisplayDate(endDate)}`
    : startDate
      ? `${formatDisplayDate(startDate)} – Select end date`
      : 'Select date range';

  const leadingBlanks = firstWeekdayOfMonth(viewMonth);
  const totalDays = daysInMonth(viewMonth);
  const { y, m } = parseIso(viewMonth);

  return (
    <div className="date-range-picker" ref={containerRef}>
      <button type="button" className="date-range-picker__trigger" onClick={toggleOpen} ref={triggerRef}>
        {triggerLabel}
      </button>
      {open && popoverPosition && (
        <div
          className="date-range-picker__popover"
          style={{ top: popoverPosition.top, left: popoverPosition.left }}
        >
          <div className="date-range-picker__nav">
            <button
              type="button"
              className="date-range-picker__nav-btn"
              aria-label="Previous month"
              onClick={() => setViewMonth(prev => addMonths(prev, -1))}
            >
              ‹
            </button>
            <span className="date-range-picker__month-label">{monthLabel(viewMonth)}</span>
            <button
              type="button"
              className="date-range-picker__nav-btn"
              aria-label="Next month"
              onClick={() => setViewMonth(prev => addMonths(prev, 1))}
            >
              ›
            </button>
          </div>
          <div className="date-range-picker__weekdays" aria-hidden="true">
            {WEEKDAY_HEADERS.map(w => (
              <span key={w} className="date-range-picker__weekday">{w}</span>
            ))}
          </div>
          <div className="date-range-picker__grid" role="grid" aria-label={monthLabel(viewMonth)}>
            {Array.from({ length: leadingBlanks }).map((_, i) => (
              <span key={`blank-${i}`} className="date-range-picker__blank" aria-hidden="true" />
            ))}
            {Array.from({ length: totalDays }, (_, i) => i + 1).map(day => {
              const iso = isoOf(y, m, day);
              const disabled = !!min && iso < min;
              const isSelected = iso === startDate || iso === endDate;
              const inRange = !!startDate && !!endDate && iso > startDate && iso < endDate;
              return (
                <button
                  key={iso}
                  type="button"
                  className={`date-range-picker__day${isSelected ? ' date-range-picker__day--selected' : ''}${inRange ? ' date-range-picker__day--in-range' : ''}`}
                  aria-label={formatDayLabel(iso)}
                  aria-pressed={isSelected}
                  disabled={disabled}
                  onClick={() => handleDayClick(iso)}
                >
                  {day}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

export default DateRangePicker;
