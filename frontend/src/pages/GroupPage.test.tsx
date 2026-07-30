import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import GroupPage from './GroupPage';
import { groupsApi, animalsApi, authApi } from '../api/client';
import type { Animal, Group, GroupMembership } from '../api/client';
import type { AxiosResponse } from 'axios';
import { AuthProvider } from '../contexts/AuthContext';
import { ToastProvider } from '../contexts/ToastContext';

// Mock the API client. GroupPage's 'animals' view only needs group/membership/animal
// data plus the site-wide group switcher list and the length-of-stay preference; the
// activity/members/documents view APIs are intentionally left unmocked since this test
// never navigates to those tabs.
vi.mock('../api/client', () => ({
  authApi: {
    getCurrentUser: vi.fn(),
    getEmailPreferences: vi.fn(),
  },
  groupsApi: {
    getById: vi.fn(),
    getMembership: vi.fn(),
    getAll: vi.fn(),
  },
  animalsApi: {
    getAll: vi.fn(),
  },
}));

// Mock useParams/useSearchParams so the page loads group id=1 directly into the
// 'animals' view (its default view is 'activity', which pulls in a much larger set
// of APIs this test doesn't care about).
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: '1' }),
    useSearchParams: () => [new URLSearchParams('view=animals'), vi.fn()],
  };
});

const mockGroup: Group = {
  id: 1,
  name: 'Test Group',
  description: 'A test group',
  image_url: '',
  hero_image_url: '',
  has_protocols: false,
  groupme_enabled: false,
};

const mockMembership: GroupMembership = {
  user_id: 1,
  group_id: 1,
  is_member: true,
  is_group_admin: false,
  is_site_admin: false,
};

const quarantinedAnimal: Animal = {
  id: 1,
  group_id: 1,
  name: 'Rex',
  species: 'Dog',
  breed: 'Mixed',
  age: 3,
  description: '',
  image_url: '',
  status: 'bite_quarantine',
  quarantine_start_date: '2026-06-22T00:00:00Z',
  quarantine_end_date: '2026-07-15T12:00:00Z', // manually overridden by staff (noon UTC avoids timezone-shift to previous day)
  is_returned: false,
};

describe('GroupPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(authApi.getCurrentUser).mockResolvedValue({
      data: {
        id: 1,
        username: 'testuser',
        email: 'test@example.com',
        phone_number: '',
        hide_email: false,
        hide_phone_number: false,
        is_admin: false,
      },
    } as AxiosResponse);

    vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
      data: { show_length_of_stay: false },
    } as AxiosResponse);

    vi.mocked(groupsApi.getById).mockResolvedValue({
      data: mockGroup,
    } as AxiosResponse<Group>);

    vi.mocked(groupsApi.getMembership).mockResolvedValue({
      data: mockMembership,
    } as AxiosResponse<GroupMembership>);

    vi.mocked(groupsApi.getAll).mockResolvedValue({
      data: [mockGroup],
    } as AxiosResponse<Group[]>);

    vi.mocked(animalsApi.getAll).mockResolvedValue({
      data: [quarantinedAnimal],
    } as AxiosResponse<Animal[]>);
  });

  const renderGroupPage = () => {
    return render(
      <BrowserRouter>
        <AuthProvider>
          <ToastProvider>
            <GroupPage />
          </ToastProvider>
        </AuthProvider>
      </BrowserRouter>
    );
  };

  it('shows the stored quarantine end date, not a recomputed one, on the animals tab', async () => {
    renderGroupPage();

    // The computed default (10 days after the 2026-06-22 start, rolled forward past any
    // weekend) lands in early July — different from the stored override below, so this
    // assertion would fail if the code still computed the fallback instead of reading
    // the stored field.
    expect(await screen.findByText(/Ends: Jul 15, 2026/)).toBeInTheDocument();
    expect(screen.queryByText(/Ends: Jul 6, 2026/)).not.toBeInTheDocument();
  });

  // Regression coverage for a refetch-storm bug: the name-search box used to
  // be undebounced and feed a useCallback whose identity change re-ran an
  // effect that also (redundantly) fetched email preferences, group data,
  // and the full groups list on every keystroke - plus a second, separate
  // effect meant only to react to filter changes duplicated the animals
  // fetch itself. Fixed by debouncing the search value and splitting the
  // once-per-visit fetches (preferences/group data/groups list, keyed only
  // on the group id) from the one place that triggers loadAnimals.
  describe('debounced search and one-time-per-visit data loading', () => {
    const DEBOUNCE_MS = 400; // must match GroupPage.tsx's NAME_SEARCH_DEBOUNCE_MS

    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    // Flushes pending microtasks (mocked-promise resolution and the
    // resulting state updates/effects) without relying on real wall-clock
    // time - findByText/waitFor poll via real setTimeout internally, which
    // never advances under fake timers, so this is the fake-timers
    // replacement for "wait for the initial load to settle".
    const flushInitialLoad = async () => {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
    };

    it('does not refetch email preferences, group data, or the groups list while typing in the name-search box', async () => {
      renderGroupPage();
      await flushInitialLoad();
      expect(screen.getByText('Rex')).toBeInTheDocument();

      // Each of these fires exactly once on initial mount - confirming that
      // up front makes the "still 1" assertions below meaningful (not just
      // "never called").
      expect(authApi.getEmailPreferences).toHaveBeenCalledTimes(1);
      expect(groupsApi.getById).toHaveBeenCalledTimes(1);
      expect(groupsApi.getMembership).toHaveBeenCalledTimes(1);
      expect(groupsApi.getAll).toHaveBeenCalledTimes(1);

      const input = screen.getByLabelText('Search animals by name');
      fireEvent.change(input, { target: { value: 'r' } });
      fireEvent.change(input, { target: { value: 're' } });
      fireEvent.change(input, { target: { value: 'rex' } });

      // Let the debounce settle (which does reload the animals list - see
      // the next test) and confirm none of the other three endpoints fire
      // again, since none of them depend on the search value anymore.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
      });

      expect(authApi.getEmailPreferences).toHaveBeenCalledTimes(1);
      expect(groupsApi.getById).toHaveBeenCalledTimes(1);
      expect(groupsApi.getMembership).toHaveBeenCalledTimes(1);
      expect(groupsApi.getAll).toHaveBeenCalledTimes(1);
    });

    it('reloads the animals list exactly once, with the final value, after typing settles', async () => {
      renderGroupPage();
      await flushInitialLoad();
      expect(screen.getByText('Rex')).toBeInTheDocument();
      vi.mocked(animalsApi.getAll).mockClear();

      const input = screen.getByLabelText('Search animals by name');
      fireEvent.change(input, { target: { value: 'r' } });
      fireEvent.change(input, { target: { value: 're' } });
      fireEvent.change(input, { target: { value: 'rex' } });

      // Not settled yet - none of the keystrokes above should have queued
      // an animals fetch on their own.
      expect(animalsApi.getAll).not.toHaveBeenCalled();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
      });

      // Exactly once - not twice (the old double-effect bug) - and with the
      // final debounced value, not an intermediate keystroke.
      expect(animalsApi.getAll).toHaveBeenCalledTimes(1);
      expect(animalsApi.getAll).toHaveBeenLastCalledWith(1, '', 'rex');
    });
  });
});
