import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import GroupPage from './GroupPage';
import { groupsApi, animalsApi, authApi } from '../api/client';
import type { Animal, Group, GroupMembership } from '../api/client';
import { CanceledError } from 'axios';
import type { AxiosResponse } from 'axios';
import { AuthProvider } from '../contexts/AuthContext';
import { ToastProvider } from '../contexts/ToastContext';

// GroupPage isn't wrapped in a real LDProvider here (that only happens in App.tsx),
// so useFlags() would otherwise return {} and hide LD-gated UI like the Schedule
// tab. Routed through a controllable mock (mockUseFlags) defaulting to flag-on,
// so most tests exercise the same group.scheduling_enabled/membership logic
// they did before the flag existed, while a dedicated test below can flip it off.
const mockUseFlags = vi.fn(() => ({ scheduleTabAccess: true }));
vi.mock('launchdarkly-react-client-sdk', () => ({
  useFlags: () => mockUseFlags(),
}));

// Mock the API client. GroupPage's 'animals' view only needs group/membership/animal
// data plus the site-wide group switcher list and the length-of-stay preference; the
// members/documents view APIs are intentionally left unmocked since this test never
// navigates to those tabs. getActivityFeed is mocked too, for the 'activity' view
// tests further below.
vi.mock('../api/client', () => ({
  authApi: {
    getCurrentUser: vi.fn(),
    getEmailPreferences: vi.fn(),
  },
  groupsApi: {
    getById: vi.fn(),
    getMembership: vi.fn(),
    getAll: vi.fn(),
    getActivityFeed: vi.fn(),
  },
  animalsApi: {
    getAll: vi.fn(),
  },
}));

// useSearchParams is routed through a controllable mock (mockUseSearchParams -
// the "mock" prefix is required for vitest to allow referencing it from inside
// this hoisted factory) so most tests can load group id=1 directly into the
// 'animals' view, while the 'activity' view tests further below can switch it
// to 'view=activity' without needing a real navigation.
const mockUseSearchParams = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: '1' }),
    useSearchParams: () => mockUseSearchParams(),
  };
});

const mockGroup: Group = {
  id: 1,
  name: 'Test Group',
  description: 'A test group',
  image_url: '',
  hero_image_url: '',
  has_protocols: false,
  scheduling_enabled: false,
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

    // Default view for most tests below; the 'activity' view tests further
    // down override this in their own beforeEach.
    mockUseSearchParams.mockReturnValue([new URLSearchParams('view=animals'), vi.fn()]);

    // Default flag-on so most tests exercise the same scheduling logic they
    // did before the flag existed; the dedicated flag test overrides this.
    mockUseFlags.mockReturnValue({ scheduleTabAccess: true });
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
      expect(animalsApi.getAll).toHaveBeenLastCalledWith(1, '', 'rex', { signal: expect.any(AbortSignal) });
    });
  });

  // Regression coverage for two further bugs found reviewing the fix above:
  // (1) "Clear Filters" reset statusFilter and nameSearch together, but
  // loadAnimals depended on the *debounced* search value, which took up to
  // NAME_SEARCH_DEBOUNCE_MS to catch up - producing one fetch combining the
  // new (empty) status with the old (stale) search term, before a second,
  // corrected fetch landed ~400ms later. (2) loadAnimals had no request
  // cancellation, unlike GroupSearch.tsx's AbortController pattern, so a
  // superseded request could resolve after a newer one and overwrite it
  // with stale data.
  describe('Clear Filters and request cancellation', () => {
    const DEBOUNCE_MS = 400; // must match GroupPage.tsx's NAME_SEARCH_DEBOUNCE_MS

    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const flush = async () => {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
    };

    it('produces exactly one fetch with the cleared params when Clear Filters is clicked, not a stale-then-corrected pair', async () => {
      renderGroupPage();
      await flush();
      expect(screen.getByText('Rex')).toBeInTheDocument();

      // Empty results so the "Clear Filters" button (only rendered in the
      // empty-results state) actually appears once filters are applied.
      vi.mocked(animalsApi.getAll).mockResolvedValue({ data: [] } as unknown as AxiosResponse<Animal[]>);

      fireEvent.change(screen.getByLabelText('Search animals by name'), { target: { value: 'rex' } });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
      });
      fireEvent.change(screen.getByLabelText('Filter animals by status'), { target: { value: 'available' } });
      await flush();

      const clearButton = screen.getByRole('button', { name: 'Clear Filters' });
      vi.mocked(animalsApi.getAll).mockClear();

      fireEvent.click(clearButton);

      // Immediately correct - a single fetch with both filters cleared
      // together, not a stale-search-term fetch followed by a correction.
      expect(animalsApi.getAll).toHaveBeenCalledTimes(1);
      expect(animalsApi.getAll).toHaveBeenLastCalledWith(1, '', '', { signal: expect.any(AbortSignal) });

      // Wait out the full debounce window to confirm nothing extra fires
      // once it would have caught up - proving the second, corrected fetch
      // the bug produced is actually gone, not just that the first fetch
      // happens to look right.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
      });
      expect(animalsApi.getAll).toHaveBeenCalledTimes(1);
    });

    it('does not let a superseded (aborted) request overwrite fresher results', async () => {
      let capturedFirstSignal: AbortSignal | undefined;
      let resolveFirst: (value: AxiosResponse<Animal[]>) => void = () => {};
      const fido: Animal = { ...quarantinedAnimal, id: 2, name: 'Fido', status: 'available' };

      // First call (the initial mount's loadAnimals): stays pending until
      // resolveFirst is invoked, but rejects immediately - the way a real
      // aborted axios request would - if its signal fires first.
      vi.mocked(animalsApi.getAll)
        .mockImplementationOnce((_groupId, _status, _name, options) => {
          capturedFirstSignal = options?.signal;
          return new Promise<AxiosResponse<Animal[]>>((resolve, reject) => {
            resolveFirst = resolve;
            options?.signal?.addEventListener('abort', () => reject(new CanceledError()));
          });
        })
        // Second call (triggered by the status-filter change below):
        // resolves normally with different data.
        .mockResolvedValueOnce({ data: [fido] } as AxiosResponse<Animal[]>);

      renderGroupPage();
      await flush();
      expect(capturedFirstSignal).toBeDefined();
      expect(capturedFirstSignal?.aborted).toBe(false);

      // Fires the second loadAnimals call before the first has resolved -
      // this must abort the first.
      fireEvent.change(screen.getByLabelText('Filter animals by status'), { target: { value: 'available' } });
      await flush();

      expect(capturedFirstSignal?.aborted).toBe(true);
      expect(screen.getByText('Fido')).toBeInTheDocument();

      // The first (now-aborted) request resolving late must not overwrite
      // the fresher data already rendered.
      await act(async () => {
        resolveFirst({ data: [quarantinedAnimal] } as AxiosResponse<Animal[]>);
        await vi.advanceTimersByTimeAsync(0);
      });

      expect(screen.getByText('Fido')).toBeInTheDocument();
      expect(screen.queryByText('Rex')).not.toBeInTheDocument();
    });
  });

  // Regression coverage for a timezone bug found reviewing the activity-feed
  // date-range filter: the backend interprets a bare YYYY-MM-DD as UTC
  // midnight, but every real user is in some other timezone - so sending the
  // date-picker's raw value produced a day boundary off by the user's UTC
  // offset (see localDayStartISO/localDayEndISO in dateUtils.ts). Fixed by
  // converting each bound to a full UTC-instant ISO timestamp, computed from
  // the *local* start/end of that calendar day, before it ever leaves the
  // browser.
  describe('activity feed date-range filter (timezone-aware)', () => {
    beforeEach(() => {
      // Deliberately non-UTC (standard time, no DST edge) - if GroupPage ever
      // regressed to sending the bare date-picker value, or swapped which
      // helper backs which field, this is what would catch it: the whole
      // point of this test is that "from"/"to" must NOT equal the bare
      // '2024-01-15'/'2024-01-20' the user typed.
      vi.stubEnv('TZ', 'America/Chicago');
      mockUseSearchParams.mockReturnValue([new URLSearchParams('view=activity'), vi.fn()]);
      vi.mocked(groupsApi.getActivityFeed).mockResolvedValue({
        data: { items: [], total: 0, limit: 20, offset: 0, hasMore: false, summary: {} },
      } as unknown as AxiosResponse);
    });

    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it('sends full local-day-boundary UTC instants for from/to, not the bare date-picker value', async () => {
      renderGroupPage();
      await screen.findByLabelText('Filter from date');

      fireEvent.change(screen.getByLabelText('Filter from date'), { target: { value: '2024-01-15' } });
      fireEvent.change(screen.getByLabelText('Filter to date'), { target: { value: '2024-01-20' } });

      await waitFor(() => {
        expect(groupsApi.getActivityFeed).toHaveBeenLastCalledWith(1, expect.objectContaining({
          // Local midnight of 2024-01-15 in America/Chicago (UTC-6, standard
          // time) is 2024-01-15T06:00:00.000Z.
          from: '2024-01-15T06:00:00.000Z',
          // Local end-of-day (23:59:59.999) of 2024-01-20 in America/Chicago
          // is 2024-01-21T05:59:59.999Z - one day later in UTC, and NOT the
          // bare '2024-01-20' a UTC-anchored interpretation would produce.
          to: '2024-01-21T05:59:59.999Z',
        }));
      });
    });

    it('omits from/to entirely when the date filters are empty, unlike the tags/rating/animal filters', async () => {
      renderGroupPage();
      await screen.findByLabelText('Filter from date');

      await waitFor(() => {
        expect(groupsApi.getActivityFeed).toHaveBeenCalled();
      });

      const lastCall = vi.mocked(groupsApi.getActivityFeed).mock.calls.at(-1);
      expect(lastCall?.[1]?.from).toBeUndefined();
      expect(lastCall?.[1]?.to).toBeUndefined();
    });
  });

  // Regression coverage for a request-cancellation gap found while verifying
  // the timezone fix above: loadActivityFeed had no AbortController, unlike
  // GroupSearch.tsx's/loadAnimals' pattern - so a superseded request (e.g.
  // changing the type filter while an earlier load was still in flight)
  // could resolve after a newer one and overwrite it with stale activity
  // items.
  describe('activity feed request cancellation', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      mockUseSearchParams.mockReturnValue([new URLSearchParams('view=activity'), vi.fn()]);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const flush = async () => {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
    };

    it('does not let a superseded (aborted) request overwrite fresher activity items', async () => {
      let capturedFirstSignal: AbortSignal | undefined;
      let resolveFirst: (value: AxiosResponse) => void = () => {};
      const staleResponse = {
        data: {
          items: [{ id: 1, type: 'comment', content: 'stale comment', created_at: '2026-01-01T00:00:00Z' }],
          total: 1, limit: 20, offset: 0, hasMore: false, summary: {},
        },
      } as unknown as AxiosResponse;
      const freshResponse = {
        data: {
          items: [{ id: 2, type: 'comment', content: 'fresh comment', created_at: '2026-01-02T00:00:00Z' }],
          total: 1, limit: 20, offset: 0, hasMore: false, summary: {},
        },
      } as unknown as AxiosResponse;

      // First call (the initial mount's loadActivityFeed): stays pending
      // until resolveFirst is invoked, but rejects immediately - the way a
      // real aborted axios request would - if its signal fires first.
      vi.mocked(groupsApi.getActivityFeed)
        .mockImplementationOnce((_groupId, options) => {
          capturedFirstSignal = options?.signal;
          return new Promise<AxiosResponse>((resolve, reject) => {
            resolveFirst = resolve;
            options?.signal?.addEventListener('abort', () => reject(new CanceledError()));
          });
        })
        // Second call (triggered by the type-filter change below): resolves
        // normally with different data.
        .mockResolvedValueOnce(freshResponse);

      renderGroupPage();
      await flush();
      expect(capturedFirstSignal).toBeDefined();
      expect(capturedFirstSignal?.aborted).toBe(false);

      // Fires the second loadActivityFeed call before the first has
      // resolved - this must abort the first.
      fireEvent.change(screen.getByLabelText('Filter activity by type'), { target: { value: 'comments' } });
      await flush();

      expect(capturedFirstSignal?.aborted).toBe(true);
      expect(screen.getByText('fresh comment')).toBeInTheDocument();

      // The first (now-aborted) request resolving late must not overwrite
      // the fresher data already rendered.
      await act(async () => {
        resolveFirst(staleResponse);
        await vi.advanceTimersByTimeAsync(0);
      });

      expect(screen.getByText('fresh comment')).toBeInTheDocument();
      expect(screen.queryByText('stale comment')).not.toBeInTheDocument();
    });
  });

  describe('scheduling toggle', () => {
    it('does not show the Schedule tab when scheduling is disabled for the group', async () => {
      renderGroupPage();
      await screen.findByRole('tab', { name: /animals/i });
      expect(screen.queryByRole('tab', { name: /schedule/i })).not.toBeInTheDocument();
    });

    it('shows the Schedule tab to a member when scheduling is enabled for the group', async () => {
      vi.mocked(groupsApi.getById).mockResolvedValue({
        data: { ...mockGroup, scheduling_enabled: true },
      } as AxiosResponse<Group>);

      renderGroupPage();
      expect(await screen.findByRole('tab', { name: /schedule/i })).toBeInTheDocument();
    });

    it('does not show the Schedule tab when the LaunchDarkly flag is off, even with scheduling enabled', async () => {
      mockUseFlags.mockReturnValue({ scheduleTabAccess: false });
      vi.mocked(groupsApi.getById).mockResolvedValue({
        data: { ...mockGroup, scheduling_enabled: true },
      } as AxiosResponse<Group>);

      renderGroupPage();
      await screen.findByRole('tab', { name: /animals/i });
      expect(screen.queryByRole('tab', { name: /schedule/i })).not.toBeInTheDocument();
    });

    it('does not show a scheduling toggle button to a regular member', async () => {
      renderGroupPage();
      await screen.findByRole('tab', { name: /animals/i });
      expect(screen.queryByRole('button', { name: /enable scheduling|disable scheduling/i })).not.toBeInTheDocument();
    });

    it('does not show a scheduling toggle button to a group admin either - it now lives in group settings', async () => {
      vi.mocked(groupsApi.getMembership).mockResolvedValue({
        data: { ...mockMembership, is_group_admin: true },
      } as AxiosResponse<GroupMembership>);

      renderGroupPage();
      await screen.findByRole('tab', { name: /animals/i });
      expect(screen.queryByRole('button', { name: /enable scheduling|disable scheduling/i })).not.toBeInTheDocument();
    });
  });
});
