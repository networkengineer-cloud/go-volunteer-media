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

  describe('popover positioning (escaping a clipping ancestor like Modal)', () => {
    function stubViewport(width: number, height: number) {
      const widthSpy = vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(width);
      const heightSpy = vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(height);
      return () => {
        widthSpy.mockRestore();
        heightSpy.mockRestore();
      };
    }

    function stubTriggerRect(trigger: HTMLElement, rect: Partial<DOMRect>) {
      vi.spyOn(trigger, 'getBoundingClientRect').mockReturnValue({
        top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => '',
        ...rect,
      } as DOMRect);
    }

    it('positions the popover below the trigger via inline styles, not the old CSS-only offset', () => {
      const restoreViewport = stubViewport(400, 800);
      try {
        render(<DateRangePicker startDate="" endDate="" onChange={vi.fn()} />);
        const trigger = screen.getByRole('button', { name: /select date range/i });
        stubTriggerRect(trigger, { top: 100, bottom: 130, left: 20, right: 280, width: 260 });

        fireEvent.click(trigger);

        const popover = document.querySelector('.date-range-picker__popover') as HTMLElement;
        // Reverting to the old position:absolute/top:calc(100% + 4px) CSS
        // (no inline style at all) would leave these both empty - this only
        // passes because the fix computes and applies them explicitly.
        expect(popover.style.top).toBe('134px'); // rect.bottom + 4
        expect(popover.style.left).toBe('20px'); // rect.left, plenty of room to the right
      } finally {
        restoreViewport();
      }
    });

    it('flips the popover above the trigger when there is not enough room below', () => {
      const restoreViewport = stubViewport(400, 700);
      try {
        render(<DateRangePicker startDate="" endDate="" onChange={vi.fn()} />);
        const trigger = screen.getByRole('button', { name: /select date range/i });
        // Only 66px of space below (700 - 630 - 4) - the 340px-tall calendar can't fit.
        stubTriggerRect(trigger, { top: 600, bottom: 630, left: 20, right: 280, width: 260 });

        fireEvent.click(trigger);

        const popover = document.querySelector('.date-range-picker__popover') as HTMLElement;
        expect(popover.style.top).toBe('256px'); // rect.top - 4 - 340
      } finally {
        restoreViewport();
      }
    });

    it('clamps the popover horizontally so it never overflows the right edge of a narrow viewport', () => {
      const restoreViewport = stubViewport(390, 800);
      try {
        render(<DateRangePicker startDate="" endDate="" onChange={vi.fn()} />);
        const trigger = screen.getByRole('button', { name: /select date range/i });
        // Trigger sits near the right edge; left-aligning the 260px popover
        // there would push it off-screen.
        stubTriggerRect(trigger, { top: 100, bottom: 130, left: 300, right: 400, width: 100 });

        fireEvent.click(trigger);

        const popover = document.querySelector('.date-range-picker__popover') as HTMLElement;
        expect(popover.style.left).toBe('122px'); // 390 - 8 (margin) - 260 (popover width)
      } finally {
        restoreViewport();
      }
    });
  });
});
