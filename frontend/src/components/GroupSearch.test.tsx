import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import GroupSearch from './GroupSearch';
import { searchApi } from '../api/client';
import type { SearchAnimalResult, SearchCommentResult, SearchResponse } from '../api/client';
import type { AxiosResponse } from 'axios';

vi.mock('../api/client', () => ({
  searchApi: {
    search: vi.fn(),
  },
}));

const renderSearch = (groupId = 1) =>
  render(
    <BrowserRouter>
      <GroupSearch groupId={groupId} />
    </BrowserRouter>
  );

const mockSearchResponse = (data: SearchResponse) => {
  vi.mocked(searchApi.search).mockResolvedValue({ data } as AxiosResponse<SearchResponse>);
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

describe('GroupSearch', () => {
  beforeEach(() => {
    vi.mocked(searchApi.search).mockReset();
  });

  it('shows no results panel before a query is typed', () => {
    renderSearch();
    expect(searchApi.search).not.toHaveBeenCalled();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('searches with q/type/limit/offset and displays animal and comment matches', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1, comments: [rexComment], total_comments: 1 });
    renderSearch(7);

    fireEvent.change(screen.getByLabelText('Search animals and comments'), {
      target: { value: 'resource guarding' },
    });

    expect(await screen.findByText('Rex')).toBeInTheDocument();
    expect(screen.getByText('on Rex')).toBeInTheDocument();
    expect(screen.getByText(/Rex had a great playgroup session today\./)).toBeInTheDocument();

    expect(searchApi.search).toHaveBeenCalledWith(7, {
      q: 'resource guarding',
      type: 'all',
      limit: 10,
      offset: 0,
    });
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

    expect(await screen.findAllByRole('status')).not.toHaveLength(0);

    resolveRequest({ data: { animals: [rex], total_animals: 1 } } as AxiosResponse<SearchResponse>);
    expect(await screen.findByText('Rex')).toBeInTheDocument();
  });

  it('shows an empty state when a query matches nothing', async () => {
    mockSearchResponse({ animals: [], total_animals: 0, comments: [], total_comments: 0 });
    renderSearch();

    fireEvent.change(screen.getByLabelText('Search animals and comments'), { target: { value: 'nonexistentterm' } });

    expect(await screen.findByText('No matches found')).toBeInTheDocument();
    expect(screen.getByText(/nonexistentterm/)).toBeInTheDocument();
  });

  it('shows an error state with a working retry on failure', async () => {
    vi.mocked(searchApi.search).mockRejectedValueOnce(new Error('network error'));
    renderSearch();

    fireEvent.change(screen.getByLabelText('Search animals and comments'), { target: { value: 'guarding' } });

    expect(await screen.findByRole('alert')).toBeInTheDocument();

    mockSearchResponse({ animals: [rex], total_animals: 1 });
    fireEvent.click(screen.getByRole('button', { name: /try again/i }));

    expect(await screen.findByText('Rex')).toBeInTheDocument();
  });

  it('omits the comments section entirely when type=animals, and vice versa', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1 });
    renderSearch();

    fireEvent.change(screen.getByLabelText('Limit search to'), { target: { value: 'animals' } });
    fireEvent.change(screen.getByLabelText('Search animals and comments'), { target: { value: 'guarding' } });

    expect(await screen.findByText('Rex')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Comments/ })).not.toBeInTheDocument();
    expect(searchApi.search).toHaveBeenCalledWith(1, { q: 'guarding', type: 'animals', limit: 10, offset: 0 });
  });

  it('loads more results on click, appending to the existing list and advancing the offset', async () => {
    const page1: SearchAnimalResult = { ...rex, id: 1, name: 'Rex' };
    const page2: SearchAnimalResult = { ...rex, id: 2, name: 'Fido' };

    vi.mocked(searchApi.search)
      .mockResolvedValueOnce({ data: { animals: [page1], total_animals: 2 } } as AxiosResponse<SearchResponse>)
      .mockResolvedValueOnce({ data: { animals: [page2], total_animals: 2 } } as AxiosResponse<SearchResponse>);

    renderSearch();
    fireEvent.change(screen.getByLabelText('Search animals and comments'), { target: { value: 'dog' } });

    expect(await screen.findByText('Rex')).toBeInTheDocument();
    const loadMore = screen.getByRole('button', { name: /load more results/i });

    fireEvent.click(loadMore);

    expect(await screen.findByText('Fido')).toBeInTheDocument();
    expect(screen.getByText('Rex')).toBeInTheDocument();

    expect(searchApi.search).toHaveBeenLastCalledWith(1, { q: 'dog', type: 'all', limit: 10, offset: 10 });
  });

  it('clears results when the query is cleared back to empty', async () => {
    mockSearchResponse({ animals: [rex], total_animals: 1 });
    renderSearch();

    const input = screen.getByLabelText('Search animals and comments');
    fireEvent.change(input, { target: { value: 'guarding' } });
    expect(await screen.findByText('Rex')).toBeInTheDocument();

    fireEvent.change(input, { target: { value: '' } });
    await waitFor(() => expect(screen.queryByText('Rex')).not.toBeInTheDocument());
  });
});
