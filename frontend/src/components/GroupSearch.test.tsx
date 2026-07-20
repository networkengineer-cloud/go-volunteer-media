import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import GroupSearch from './GroupSearch';
import { searchApi } from '../api/client';
import type { SearchAnimalResult, SearchCommentResult, SearchUpdateResult, SearchResponse } from '../api/client';
import type { AxiosResponse } from 'axios';

vi.mock('../api/client', () => ({
  searchApi: {
    search: vi.fn(),
  },
}));

// Must match GroupSearch.tsx's DEBOUNCE_MS.
const DEBOUNCE_MS = 400;

const renderSearch = (groupId = 1) =>
  render(
    <BrowserRouter>
      <GroupSearch groupId={groupId} />
    </BrowserRouter>
  );

const mockSearchResponse = (data: SearchResponse) => {
  vi.mocked(searchApi.search).mockResolvedValue({ data } as AxiosResponse<SearchResponse>);
};

// Flushes pending microtasks (mocked-promise resolution and the resulting
// state updates/effects) without relying on real wall-clock time — the
// fake-timers replacement for RTL's real-timer-based findBy/waitFor.
const flush = async () => {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
};

// Types into the search box and advances fake time past the debounce delay,
// flushing the resulting search request/state updates along the way.
const typeQuery = async (input: HTMLElement, value: string) => {
  fireEvent.change(input, { target: { value } });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
  });
};

const rex: SearchAnimalResult = {
  id: 1,
  group_id: 1,
  name: 'Rex',
  species: 'Dog',
  breed: 'Shepherd',
  age: 3,
  description: 'Shows resource guarding around food bowls.',
  image_url: '',
  status: 'available',
  is_returned: false,
  rank: 0.5,
};

const rexComment: SearchCommentResult = {
  id: 10,
  animal_id: 1,
  user_id: 1,
  content: 'Rex had a great playgroup session today.',
  image_url: '',
  is_edited: false,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  animal_name: 'Rex',
  rank: 0.3,
};

const playgroupUpdate: SearchUpdateResult = {
  id: 20,
  group_id: 1,
  user_id: 1,
  title: 'Playgroup Saturday',
  content: 'Pairings: Rex+Fido, Bella+Max. 10am at the field.',
  image_url: '',
  send_groupme: false,
  created_at: '2026-07-01T00:00:00Z',
  rank: 0.4,
};

describe('GroupSearch', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(searchApi.search).mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows no results panel before a query is typed', () => {
    renderSearch();
    expect(searchApi.search).not.toHaveBeenCalled();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('searches with q/type/limit/offset and displays animal and comment matches', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1, comments: [rexComment], total_comments: 1 });
    renderSearch(7);

    await typeQuery(screen.getByLabelText('Search animals and comments'), 'resource guarding');

    expect(screen.getByText('Rex')).toBeInTheDocument();
    expect(screen.getByText('on Rex')).toBeInTheDocument();
    expect(screen.getByText(/Rex had a great playgroup session today\./)).toBeInTheDocument();
    expect(screen.getByRole('status')).toBeInTheDocument();

    expect(searchApi.search).toHaveBeenCalledWith(
      7,
      { q: 'resource guarding', type: 'all', limit: 10, offset: 0 },
      { signal: expect.any(AbortSignal) }
    );
  });

  it('shows a loading state while the request is in flight', async () => {
    let resolveRequest: (value: AxiosResponse<SearchResponse>) => void = () => {};
    vi.mocked(searchApi.search).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve;
      })
    );
    renderSearch();

    fireEvent.change(screen.getByLabelText('Search animals and comments'), { target: { value: 'guarding' } });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
    });

    expect(screen.getAllByRole('status').length).toBeGreaterThan(0);

    await act(async () => {
      resolveRequest({ data: { animals: [rex], total_animals: 1 } } as AxiosResponse<SearchResponse>);
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByText('Rex')).toBeInTheDocument();
  });

  it('shows an empty state when a query matches nothing', async () => {
    mockSearchResponse({ animals: [], total_animals: 0, comments: [], total_comments: 0 });
    renderSearch();

    await typeQuery(screen.getByLabelText('Search animals and comments'), 'nonexistentterm');

    expect(screen.getByText('No matches found')).toBeInTheDocument();
    expect(screen.getByText(/nonexistentterm/)).toBeInTheDocument();
  });

  it('shows an error state, logs the failure, and retries successfully', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const networkError = new Error('network error');
    vi.mocked(searchApi.search).mockRejectedValueOnce(networkError);
    renderSearch();

    await typeQuery(screen.getByLabelText('Search animals and comments'), 'guarding');

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(consoleErrorSpy).toHaveBeenCalledWith('Failed to search:', networkError);

    mockSearchResponse({ animals: [rex], total_animals: 1 });
    fireEvent.click(screen.getByRole('button', { name: /try again/i }));
    await flush();

    expect(screen.getByText('Rex')).toBeInTheDocument();
    consoleErrorSpy.mockRestore();
  });

  it('does not surface an error for a cancelled/superseded request', async () => {
    // Simulate what an aborted request looks like: axios rejects with a
    // cancellation error, which axios.isCancel() must recognize so it's
    // swallowed instead of shown as "Failed to search".
    const { CanceledError } = await import('axios');
    vi.mocked(searchApi.search).mockRejectedValueOnce(new CanceledError());
    renderSearch();

    await typeQuery(screen.getByLabelText('Search animals and comments'), 'guarding');

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('hides the comments section entirely when type=animals', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1 });
    renderSearch();

    fireEvent.change(screen.getByLabelText('Limit search to'), { target: { value: 'animals' } });
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'guarding');

    expect(screen.getByText('Rex')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Comments/ })).not.toBeInTheDocument();
    expect(searchApi.search).toHaveBeenCalledWith(
      1,
      { q: 'guarding', type: 'animals', limit: 10, offset: 0 },
      { signal: expect.any(AbortSignal) }
    );
  });

  it('shows update matches and hides them when type=comments', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1, updates: [playgroupUpdate], total_updates: 1 });
    renderSearch();

    await typeQuery(screen.getByLabelText('Search animals and comments'), 'pairings');

    expect(screen.getByText('Playgroup Saturday')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Limit search to'), { target: { value: 'comments' } });
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'pairings');
    expect(screen.queryByText('Playgroup Saturday')).not.toBeInTheDocument();
  });

  it('hides the animals section entirely when type=comments', async () => {
    mockSearchResponse({ comments: [rexComment], total_comments: 1 });
    renderSearch();

    fireEvent.change(screen.getByLabelText('Limit search to'), { target: { value: 'comments' } });
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'guarding');

    expect(screen.getByText('on Rex')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Animals/ })).not.toBeInTheDocument();
    expect(searchApi.search).toHaveBeenCalledWith(
      1,
      { q: 'guarding', type: 'comments', limit: 10, offset: 0 },
      { signal: expect.any(AbortSignal) }
    );
  });

  it('hides the animals and comments sections entirely when type=updates', async () => {
    mockSearchResponse({ updates: [playgroupUpdate], total_updates: 1 });
    renderSearch();

    fireEvent.change(screen.getByLabelText('Limit search to'), { target: { value: 'updates' } });
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'pairings');

    expect(screen.getByText('Playgroup Saturday')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Animals/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Comments/ })).not.toBeInTheDocument();
    expect(searchApi.search).toHaveBeenCalledWith(
      1,
      { q: 'pairings', type: 'updates', limit: 10, offset: 0 },
      { signal: expect.any(AbortSignal) }
    );
  });

  it('loads more results on click, appending to the existing list and advancing the offset', async () => {
    const page1: SearchAnimalResult = { ...rex, id: 1, name: 'Rex' };
    const page2: SearchAnimalResult = { ...rex, id: 2, name: 'Fido' };

    vi.mocked(searchApi.search)
      .mockResolvedValueOnce({ data: { animals: [page1], total_animals: 2 } } as AxiosResponse<SearchResponse>)
      .mockResolvedValueOnce({ data: { animals: [page2], total_animals: 2 } } as AxiosResponse<SearchResponse>);

    renderSearch();
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'dog');

    expect(screen.getByText('Rex')).toBeInTheDocument();
    const loadMore = screen.getByRole('button', { name: /load more results/i });

    fireEvent.click(loadMore);
    await flush();

    expect(screen.getByText('Fido')).toBeInTheDocument();
    expect(screen.getByText('Rex')).toBeInTheDocument();

    expect(searchApi.search).toHaveBeenLastCalledWith(
      1,
      { q: 'dog', type: 'all', limit: 10, offset: 10 },
      { signal: expect.any(AbortSignal) }
    );
  });

  it('clears results when the query is cleared back to empty', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1 });
    renderSearch();

    const input = screen.getByLabelText('Search animals and comments');
    await typeQuery(input, 'guarding');
    expect(screen.getByText('Rex')).toBeInTheDocument();

    fireEvent.change(input, { target: { value: '' } });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(DEBOUNCE_MS);
    });
    expect(screen.queryByText('Rex')).not.toBeInTheDocument();
  });

  it('clears stale results immediately when groupId changes, and searches the new group', async () => {
    const groupAAnimal: SearchAnimalResult = { ...rex, id: 1, name: 'Rex' };
    const groupBAnimal: SearchAnimalResult = { ...rex, id: 99, name: 'Buddy' };

    vi.mocked(searchApi.search)
      .mockResolvedValueOnce({ data: { animals: [groupAAnimal], total_animals: 1 } } as AxiosResponse<SearchResponse>)
      .mockResolvedValueOnce({ data: { animals: [groupBAnimal], total_animals: 1 } } as AxiosResponse<SearchResponse>);

    const { rerender } = renderSearch(1);
    await typeQuery(screen.getByLabelText('Search animals and comments'), 'dog');
    expect(screen.getByText('Rex')).toBeInTheDocument();

    const rexLink = screen.getByText('Rex').closest('a');
    expect(rexLink).toHaveAttribute('href', '/groups/1/animals/1/view');

    await act(async () => {
      rerender(
        <BrowserRouter>
          <GroupSearch groupId={2} />
        </BrowserRouter>
      );
    });

    // Group A's result must be gone immediately, before group B's search
    // resolves — otherwise it would briefly render under a /groups/2/... link.
    expect(screen.queryByText('Rex')).not.toBeInTheDocument();

    await flush();

    expect(screen.getByText('Buddy')).toBeInTheDocument();
    const buddyLink = screen.getByText('Buddy').closest('a');
    expect(buddyLink).toHaveAttribute('href', '/groups/2/animals/99/view');

    expect(searchApi.search).toHaveBeenLastCalledWith(
      2,
      { q: 'dog', type: 'all', limit: 10, offset: 0 },
      { signal: expect.any(AbortSignal) }
    );
  });
});
