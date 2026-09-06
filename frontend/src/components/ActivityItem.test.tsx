import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ActivityItem from './ActivityItem';
import type { ActivityItem as ActivityItemType } from '../api/client';

const baseItem: ActivityItemType = {
  id: 1,
  type: 'coverage_request',
  created_at: '2026-09-01T12:00:00Z',
  user_id: 2,
  user: { id: 2, username: 'jane', email: 'jane@example.com', phone_number: '', hide_email: false, hide_phone_number: false, is_admin: false },
  content: '',
  date: '2026-09-12',
  hour: 9,
  status: 'open',
};

const renderItem = (item: ActivityItemType) =>
  render(
    <BrowserRouter>
      <ActivityItem item={item} groupId={1} />
    </BrowserRouter>
  );

describe('ActivityItem coverage_request rendering', () => {
  it('shows the shift date/time and an open status for an open request', () => {
    renderItem(baseItem);

    expect(screen.getByText(/sat, sep 12/i)).toBeInTheDocument();
    expect(screen.getByText(/9:00 am/i)).toBeInTheDocument();
    expect(screen.getByText(/needs coverage/i)).toBeInTheDocument();
  });

  it('shows who claimed the shift for a claimed request', () => {
    renderItem({
      ...baseItem,
      status: 'claimed',
      claimed_by_user: { id: 3, username: 'bob', email: 'bob@example.com', phone_number: '', hide_email: false, hide_phone_number: false, is_admin: false },
    });

    expect(screen.getByText(/claimed by bob/i)).toBeInTheDocument();
    expect(screen.queryByText(/needs coverage/i)).not.toBeInTheDocument();
  });

  it('links to the group\'s schedule tab', () => {
    renderItem(baseItem);

    const link = screen.getByRole('link', { name: /view in schedule/i });
    expect(link).toHaveAttribute('href', '/groups/1?view=schedule');
  });
});
