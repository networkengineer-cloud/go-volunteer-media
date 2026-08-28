import React from 'react';
import { weekParity, upcomingDatesForParity } from './scheduleGrid';
import './ScheduleTab.css';

export interface CadenceLegendProps {
  // The ISO Sunday (YYYY-MM-DD) of the week currently being viewed, if any.
  // When given, that parity's entry is visually highlighted as "current".
  referenceWeekStart?: string;
}

function formatShortDate(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' });
}

const UPCOMING_COUNT = 3;

const CadenceLegend: React.FC<CadenceLegendProps> = ({ referenceWeekStart }) => {
  const currentParity = referenceWeekStart ? weekParity(referenceWeekStart) : null;
  const datesA = upcomingDatesForParity('a', UPCOMING_COUNT);
  const datesB = upcomingDatesForParity('b', UPCOMING_COUNT);

  return (
    <div className="cadence-legend">
      <span className={`cadence-legend__entry ${currentParity === 'a' ? 'cadence-legend__entry--current' : ''}`}>
        <strong>Week A:</strong> {datesA.map(formatShortDate).join(', ')}
      </span>
      <span className={`cadence-legend__entry ${currentParity === 'b' ? 'cadence-legend__entry--current' : ''}`}>
        <strong>Week B:</strong> {datesB.map(formatShortDate).join(', ')}
      </span>
    </div>
  );
};

export default CadenceLegend;
