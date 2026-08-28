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

  it('highlights Week A when referenceWeekStart falls on an A week', () => {
    render(<CadenceLegend referenceWeekStart="2024-01-07" />);
    expect(screen.getByText(/Week A:/).closest('.cadence-legend__entry')).toHaveClass('cadence-legend__entry--current');
    expect(screen.getByText(/Week B:/).closest('.cadence-legend__entry')).not.toHaveClass('cadence-legend__entry--current');
  });

  it('highlights Week B when referenceWeekStart falls on a B week', () => {
    render(<CadenceLegend referenceWeekStart="2024-01-14" />);
    expect(screen.getByText(/Week B:/).closest('.cadence-legend__entry')).toHaveClass('cadence-legend__entry--current');
  });

  it('highlights neither when no referenceWeekStart is given', () => {
    render(<CadenceLegend />);
    expect(screen.getByText(/Week A:/).closest('.cadence-legend__entry')).not.toHaveClass('cadence-legend__entry--current');
    expect(screen.getByText(/Week B:/).closest('.cadence-legend__entry')).not.toHaveClass('cadence-legend__entry--current');
  });
});
