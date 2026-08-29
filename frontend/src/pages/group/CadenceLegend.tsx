import React from 'react';
import { weekParity, upcomingDatesForParity } from './scheduleGrid';
import './ScheduleTab.css';

export interface CadenceLegendProps {
  // The ISO Sunday (YYYY-MM-DD) of the week currently being viewed, if any.
  // When given, that parity's entry is marked "currently viewing" (both
  // visually and via aria-current, so it isn't a color/weight-only signal).
  referenceWeekStart?: string;
}

function formatShortDate(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' });
}

const UPCOMING_COUNT = 3;

interface LegendEntryProps {
  letter: 'A' | 'B';
  dates: string[];
  isCurrent: boolean;
}

// A dedicated entry component (rather than inlining this twice) keeps the
// aria-current/visible-text-marker logic in exactly one place for both
// Week A and Week B. The "(currently viewing)" marker is appended after the
// dates (rather than folded into the "Week A:"/"Week B:" label itself) so
// that label stays a stable, literal substring in both states - other
// components (e.g. ScheduleOverview) match on it directly.
const LegendEntry: React.FC<LegendEntryProps> = ({ letter, dates, isCurrent }) => (
  <span
    className={`cadence-legend__entry ${isCurrent ? 'cadence-legend__entry--current' : ''}`}
    aria-current={isCurrent ? 'true' : undefined}
  >
    <strong>Week {letter}:</strong> weeks of {dates.map(formatShortDate).join(', ')}
    {isCurrent && ' (currently viewing)'}
  </span>
);

const CadenceLegend: React.FC<CadenceLegendProps> = ({ referenceWeekStart }) => {
  const currentParity = referenceWeekStart ? weekParity(referenceWeekStart) : null;
  const datesA = upcomingDatesForParity('a', UPCOMING_COUNT);
  const datesB = upcomingDatesForParity('b', UPCOMING_COUNT);

  return (
    <div className="cadence-legend">
      <LegendEntry letter="A" dates={datesA} isCurrent={currentParity === 'a'} />
      <LegendEntry letter="B" dates={datesB} isCurrent={currentParity === 'b'} />
    </div>
  );
};

export default CadenceLegend;
