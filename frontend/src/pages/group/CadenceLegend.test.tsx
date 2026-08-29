import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import CadenceLegend from './CadenceLegend';

describe('CadenceLegend', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2024-01-10T12:00:00Z')); // within the "a" week (2024-01-07..13)
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows three upcoming dates for each of Week A and Week B', () => {
    render(<CadenceLegend />);
    expect(screen.getByText(/Week A:/)).toBeInTheDocument();
    expect(screen.getByText(/Week B:/)).toBeInTheDocument();
    // Week A's first upcoming date from 2024-01-10 (an A week) is Jan 7.
    expect(screen.getByText(/Jan 7/)).toBeInTheDocument();
    // Week B's first upcoming date from 2024-01-10 is Jan 14.
    expect(screen.getByText(/Jan 14/)).toBeInTheDocument();
  });

  it('frames the dates as week-start Sundays, not the volunteer\'s actual shift day', () => {
    render(<CadenceLegend />);
    // "weeks of ..." disambiguates these from the volunteer's real shift
    // day (e.g. a Tuesday shift), which would otherwise be easy to
    // mistake bare dates like "Jan 7" for.
    const entries = screen.getAllByText(/weeks of/i);
    expect(entries.length).toBe(2);
  });

  it('highlights Week A with aria-current and a visible text cue when referenceWeekStart falls on an A week', () => {
    render(<CadenceLegend referenceWeekStart="2024-01-07" />);
    const currentEntry = screen.getByText(/Week A/).closest('.cadence-legend__entry');
    const otherEntry = screen.getByText(/Week B/).closest('.cadence-legend__entry');

    expect(currentEntry).toHaveClass('cadence-legend__entry--current');
    expect(currentEntry).toHaveAttribute('aria-current', 'true');
    // Must be an actual text cue, not just a class/style change, so it's
    // perceivable without relying on color or font-weight.
    expect(currentEntry).toHaveTextContent(/currently viewing/i);

    expect(otherEntry).not.toHaveClass('cadence-legend__entry--current');
    expect(otherEntry).not.toHaveAttribute('aria-current');
    expect(otherEntry).not.toHaveTextContent(/currently viewing/i);
  });

  it('highlights Week B with aria-current and a visible text cue when referenceWeekStart falls on a B week', () => {
    render(<CadenceLegend referenceWeekStart="2024-01-14" />);
    const currentEntry = screen.getByText(/Week B/).closest('.cadence-legend__entry');
    const otherEntry = screen.getByText(/Week A/).closest('.cadence-legend__entry');

    expect(currentEntry).toHaveClass('cadence-legend__entry--current');
    expect(currentEntry).toHaveAttribute('aria-current', 'true');
    expect(currentEntry).toHaveTextContent(/currently viewing/i);

    expect(otherEntry).not.toHaveClass('cadence-legend__entry--current');
    expect(otherEntry).not.toHaveAttribute('aria-current');
  });

  it('highlights neither when no referenceWeekStart is given', () => {
    render(<CadenceLegend />);
    const entryA = screen.getByText(/Week A/).closest('.cadence-legend__entry');
    const entryB = screen.getByText(/Week B/).closest('.cadence-legend__entry');
    expect(entryA).not.toHaveClass('cadence-legend__entry--current');
    expect(entryA).not.toHaveAttribute('aria-current');
    expect(entryB).not.toHaveClass('cadence-legend__entry--current');
    expect(entryB).not.toHaveAttribute('aria-current');
  });
});
