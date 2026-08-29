import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import DateRangePicker from './DateRangePicker';

describe('DateRangePicker', () => {
  it('shows a placeholder when no range is set', () => {
    render(<DateRangePicker startDate="" endDate="" onChange={vi.fn()} />);
    expect(screen.getByRole('button', { name: /select date range/i })).toBeInTheDocument();
  });

  it('shows the formatted range on the trigger when both dates are set', () => {
    render(<DateRangePicker startDate="2026-08-29" endDate="2026-08-30" onChange={vi.fn()} />);
    expect(screen.getByRole('button', { name: /Aug 29, 2026.*Aug 30, 2026/ })).toBeInTheDocument();
  });

  it('opens a calendar grid of day buttons when the trigger is clicked', () => {
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={vi.fn()} />);
    expect(screen.queryByRole('button', { name: 'August 15, 2026' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));

    expect(screen.getByRole('button', { name: 'August 15, 2026' })).toBeInTheDocument();
  });

  it('picking a first day sets the start and keeps the popover open, prompting for an end date', () => {
    const onChange = vi.fn();
    render(<DateRangePicker startDate="" endDate="" onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: /select date range/i }));

    fireEvent.click(screen.getByRole('button', { name: 'August 15, 2026' }));

    expect(onChange).toHaveBeenCalledWith('2026-08-15', '');
  });

  it('picking a later day completes the range and closes the popover', () => {
    const onChange = vi.fn();
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));

    fireEvent.click(screen.getByRole('button', { name: 'August 20, 2026' }));

    expect(onChange).toHaveBeenCalledWith('2026-08-15', '2026-08-20');
    expect(screen.queryByRole('button', { name: 'August 20, 2026' })).not.toBeInTheDocument();
  });

  it('picking a day earlier than the current start restarts the selection instead of completing it', () => {
    const onChange = vi.fn();
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));

    fireEvent.click(screen.getByRole('button', { name: 'August 10, 2026' }));

    expect(onChange).toHaveBeenCalledWith('2026-08-10', '');
  });

  it('starting a new selection after a complete range clears the old end date', () => {
    const onChange = vi.fn();
    render(<DateRangePicker startDate="2026-08-15" endDate="2026-08-20" onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026.*Aug 20, 2026/ }));

    fireEvent.click(screen.getByRole('button', { name: 'August 5, 2026' }));

    expect(onChange).toHaveBeenCalledWith('2026-08-05', '');
  });

  it('disables dates before min and does not fire onChange when clicked', () => {
    const onChange = vi.fn();
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={onChange} min="2026-08-10" />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));

    const pastDay = screen.getByRole('button', { name: 'August 5, 2026' });
    expect(pastDay).toBeDisabled();
    fireEvent.click(pastDay);
    expect(onChange).not.toHaveBeenCalled();
  });

  it('navigates to the next month and shows its days', () => {
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));
    expect(screen.queryByRole('button', { name: 'September 5, 2026' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /next month/i }));

    expect(screen.getByRole('button', { name: 'September 5, 2026' })).toBeInTheDocument();
  });

  it('closes the popover on Escape', () => {
    render(<DateRangePicker startDate="2026-08-15" endDate="" onChange={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));
    expect(screen.getByRole('button', { name: 'August 15, 2026' })).toBeInTheDocument();

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(screen.queryByRole('button', { name: 'August 15, 2026' })).not.toBeInTheDocument();
  });

  it('closes the popover when clicking outside', () => {
    render(
      <div>
        <DateRangePicker startDate="2026-08-15" endDate="" onChange={vi.fn()} />
        <button type="button">outside</button>
      </div>
    );
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026/ }));
    expect(screen.getByRole('button', { name: 'August 15, 2026' })).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByRole('button', { name: 'outside' }));

    expect(screen.queryByRole('button', { name: 'August 15, 2026' })).not.toBeInTheDocument();
  });

  it('marks the selected start and end days so they can be styled distinctly', () => {
    render(<DateRangePicker startDate="2026-08-15" endDate="2026-08-20" onChange={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: /Aug 15, 2026.*Aug 20, 2026/ }));

    const grid = screen.getByRole('grid');
    expect(within(grid).getByRole('button', { name: 'August 15, 2026' })).toHaveAttribute('aria-pressed', 'true');
    expect(within(grid).getByRole('button', { name: 'August 20, 2026' })).toHaveAttribute('aria-pressed', 'true');
    expect(within(grid).getByRole('button', { name: 'August 17, 2026' })).toHaveAttribute('aria-pressed', 'false');
  });
});
