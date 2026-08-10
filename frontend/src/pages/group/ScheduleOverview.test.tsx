import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ScheduleOverview from './ScheduleOverview';
import { scheduleApi } from '../../api/client';
import type { AxiosResponse } from 'axios';
import type { ScheduleOverviewResponse } from '../../api/client';

vi.mock('../../api/client', () => ({
  scheduleApi: {
    getOverview: vi.fn(),
  },
}));

function mockOverview(data: ScheduleOverviewResponse) {
  vi.mocked(scheduleApi.getOverview).mockResolvedValue({ data } as unknown as AxiosResponse<ScheduleOverviewResponse>);
}

describe('ScheduleOverview', () => {
  beforeEach(() => {
    mockOverview({ slots: [] });
  });

  it('loads the overview for the given group on mount', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} />);
    await waitFor(() => expect(scheduleApi.getOverview).toHaveBeenCalledWith(7, expect.objectContaining({ signal: expect.any(AbortSignal) })));
  });

  it('renders a tier class proportional to how many members are available', async () => {
    mockOverview({
      slots: [
        { day_of_week: 2, hour: 9, members: [{ user_id: 1, username: 'vol1' }, { user_id: 2, username: 'vol2' }] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*2 available/i });
    // 2 of 4 members = 50%, falls in the 26-50% tier.
    expect(cell).toHaveClass('schedule-grid__slot--tier-2');
  });

  it('a cell with nobody available is not clickable and has the zero tier', async () => {
    render(<ScheduleOverview groupId={7} totalMembers={4} />);
    const cell = await screen.findByRole('cell', { name: 'Sun 8:00 AM, 0 available' });
    expect(cell).toHaveClass('schedule-grid__slot--tier-0');
    expect(cell.tagName).not.toBe('BUTTON');
  });

  it('clicking a non-empty cell opens a popover listing member names', async () => {
    mockOverview({
      slots: [
        { day_of_week: 2, hour: 9, members: [{ user_id: 1, username: 'vol1', first_name: 'Jane', last_name: 'Doe' }] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument();
  });

  it('falls back to username when first/last name are blank', async () => {
    mockOverview({
      slots: [{ day_of_week: 2, hour: 9, members: [{ user_id: 1, username: 'vol1' }] }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('vol1')).toBeInTheDocument();
  });

  it('falls back to a data-derived denominator when totalMembers is 0 but slots have members', async () => {
    mockOverview({
      slots: [
        { day_of_week: 2, hour: 9, members: [{ user_id: 1, username: 'vol1' }, { user_id: 2, username: 'vol2' }] },
      ],
    });
    render(<ScheduleOverview groupId={7} totalMembers={0} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*2 available/i });
    // totalMembers is 0 (unreliable), so the fallback denominator is derived
    // from the fetched data: max slot size of 2, so 2/2 = 100% = tier-4.
    expect(cell).not.toHaveClass('schedule-grid__slot--tier-0');
    expect(cell).toHaveClass('schedule-grid__slot--tier-4');
  });

  it('clicking outside the popover closes it', async () => {
    mockOverview({
      slots: [{ day_of_week: 2, hour: 9, members: [{ user_id: 1, username: 'vol1' }] }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={4} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*1 available/i });
    fireEvent.click(cell);
    expect(await screen.findByText('vol1')).toBeInTheDocument();

    fireEvent.mouseDown(document.body);
    await waitFor(() => expect(screen.queryByText('vol1')).not.toBeInTheDocument());
  });

  it('renders every member in a long list even though the popover box is height-capped', async () => {
    // jsdom doesn't compute layout, so this can't verify the CSS
    // max-height/overflow-y clamp actually kicks in visually (that's
    // covered by the Playwright verification instead) - but it does
    // confirm the popover keeps rendering all members in the DOM (for
    // scrolling to reveal) rather than e.g. truncating the list to fit.
    const members = Array.from({ length: 20 }, (_, i) => ({
      user_id: i + 1,
      username: `vol${i + 1}`,
      first_name: `Member`,
      last_name: `${i + 1}`,
    }));
    mockOverview({
      slots: [{ day_of_week: 2, hour: 9, members }],
    });
    render(<ScheduleOverview groupId={7} totalMembers={20} />);

    const cell = await screen.findByRole('cell', { name: /Tue 9:00 AM.*20 available/i });
    fireEvent.click(cell);

    expect(await screen.findByText('Member 1')).toBeInTheDocument();
    expect(screen.getByText('Member 20')).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(20);
  });
});
