import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import AnimalDetailPage from './AnimalDetailPage';
import { animalsApi, animalCommentsApi, commentTagsApi, groupsApi, authApi } from '../api/client';
import type { Animal, AnimalComment } from '../api/client';
import type { AxiosResponse } from 'axios';
import { AuthProvider } from '../contexts/AuthContext';
import { ToastProvider } from '../contexts/ToastContext';

let mockRouteParams = { groupId: '1', id: '1' };

// Mock the API client
vi.mock('../api/client', () => ({
  authApi: {
    getCurrentUser: vi.fn(),
  },
  animalsApi: {
    getById: vi.fn(),
  },
  animalCommentsApi: {
    getAll: vi.fn(),
    getDeleted: vi.fn(),
    getPosition: vi.fn(),
  },
  commentTagsApi: {
    getAll: vi.fn(),
  },
  groupsApi: {
    getById: vi.fn(),
    getMembership: vi.fn(),
  },
}));

// Mock useParams to provide a fixed groupId/id, while keeping the rest of react-router-dom real.
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => mockRouteParams,
  };
});

const baseAnimal: Animal = {
  id: 1,
  group_id: 1,
  name: 'Rex',
  species: 'Dog',
  breed: 'Labrador',
  age: 3,
  description: '',
  image_url: '',
  status: 'available',
  is_returned: false,
};

const mockAnimal = (overrides: Partial<Animal>) => {
  vi.mocked(animalsApi.getById).mockResolvedValue({
    data: { ...baseAnimal, ...overrides },
  } as AxiosResponse<Animal>);
};

const mockComment = (overrides: Partial<AnimalComment>): AnimalComment => ({
  id: 1,
  animal_id: 1,
  user_id: 1,
  content: 'a comment',
  image_url: '',
  is_edited: false,
  created_at: '2026-06-22T00:00:00Z',
  updated_at: '2026-06-22T00:00:00Z',
  ...overrides,
});

const mockDeletedComment = (overrides: Partial<AnimalComment>): AnimalComment => ({
  id: 1,
  animal_id: 1,
  user_id: 1,
  content: 'deleted comment',
  image_url: '',
  is_edited: false,
  created_at: '2026-06-22T00:00:00Z',
  updated_at: '2026-06-22T00:00:00Z',
  deleted_at: '2026-06-23T00:00:00Z',
  ...overrides,
});

describe('AnimalDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockRouteParams = { groupId: '1', id: '1' };

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

    vi.mocked(groupsApi.getById).mockResolvedValue({
      data: {},
    } as AxiosResponse);

    vi.mocked(groupsApi.getMembership).mockResolvedValue({
      data: {},
    } as AxiosResponse);

    vi.mocked(commentTagsApi.getAll).mockResolvedValue({
      data: [],
    } as AxiosResponse);

    vi.mocked(animalCommentsApi.getAll).mockResolvedValue({
      data: { comments: [], total: 0, limit: 10, offset: 0, hasMore: false },
    } as AxiosResponse);

    vi.mocked(animalCommentsApi.getDeleted).mockResolvedValue({
      data: [],
    } as AxiosResponse);
  });

  const renderDetailPage = () => {
    return render(
      <BrowserRouter>
        <AuthProvider>
          <ToastProvider>
            <AnimalDetailPage />
          </ToastProvider>
        </AuthProvider>
      </BrowserRouter>
    );
  };

  it('shows the bite quarantine incident details while in quarantine', async () => {
    mockAnimal({
      status: 'bite_quarantine',
      quarantine_start_date: '2026-06-22T00:00:00Z',
      quarantine_incident_details: 'Bit a volunteer on the hand.',
    });

    renderDetailPage();

    expect(await screen.findByText('Bit a volunteer on the hand.')).toBeInTheDocument();
  });

  it('does not show incident details when not in quarantine', async () => {
    mockAnimal({ status: 'available', quarantine_incident_details: '' });

    renderDetailPage();

    await screen.findByRole('heading', { name: 'Rex' });
    expect(screen.queryByText(/Incident Details/i)).not.toBeInTheDocument();
  });

  it('shows the stored quarantine end date rather than a recomputed one', async () => {
    mockAnimal({
      status: 'bite_quarantine',
      quarantine_start_date: '2026-06-22T00:00:00Z',
      quarantine_end_date: '2026-07-15T12:00:00Z', // manually overridden by staff (noon UTC avoids timezone-shift to previous day)
    });

    renderDetailPage();

    expect(await screen.findByText(/End: July 15, 2026/)).toBeInTheDocument();
  });

  it('falls back to a computed end date when no stored end date is present yet', async () => {
    mockAnimal({
      status: 'bite_quarantine',
      quarantine_start_date: '2024-06-03T12:00:00Z', // Monday
    });

    renderDetailPage();

    expect(await screen.findByText(/End: June 13, 2024/)).toBeInTheDocument();
  });

  describe('comment deep-linking from search (?comment=<id>)', () => {
    beforeEach(() => {
      // jsdom doesn't implement scrollIntoView.
      Element.prototype.scrollIntoView = vi.fn();
      mockAnimal({ status: 'available' });
      // Only tests that actually exercise the "not on the loaded page(s)"
      // path override this; default to "not found" so an accidental call
      // elsewhere doesn't hang on an unresolved mock.
      vi.mocked(animalCommentsApi.getPosition).mockResolvedValue({
        data: { found: false },
      } as AxiosResponse);
    });

    afterEach(() => {
      window.history.pushState({}, '', '/');
    });

    it('scrolls to and highlights the comment named in the ?comment= query param', async () => {
      vi.mocked(animalCommentsApi.getAll).mockResolvedValue({
        data: {
          comments: [
            mockComment({ id: 1, content: 'first comment' }),
            mockComment({ id: 2, content: 'second comment' }),
          ],
          total: 2,
          limit: 10,
          offset: 0,
          hasMore: false,
        },
      } as AxiosResponse);

      window.history.pushState({}, '', '/groups/1/animals/1/view?comment=2');
      renderDetailPage();

      await screen.findByText('second comment');
      // The highlight class is applied by a useEffect that runs after the
      // comments state update commits — a separate render pass from the one
      // findByText observes — so wait for it rather than asserting inline.
      await waitFor(() => {
        expect(document.getElementById('comment-2')).toHaveClass('comment-card--highlighted');
      });
      expect(Element.prototype.scrollIntoView).toHaveBeenCalled();

      const notHighlighted = document.getElementById('comment-1');
      expect(notHighlighted).not.toHaveClass('comment-card--highlighted');
      // Already on the loaded page — no need to ask the backend for it.
      expect(animalCommentsApi.getPosition).not.toHaveBeenCalled();
    });

    it('jumps straight to the page containing a deep-linked comment that is not already loaded', async () => {
      const earlierComments = Array.from({ length: 10 }, (_, i) => mockComment({ id: 100 + i, content: `comment ${i}` }));

      vi.mocked(animalCommentsApi.getAll)
        .mockResolvedValueOnce({
          data: { comments: earlierComments, total: 21, limit: 10, offset: 0, hasMore: true },
        } as AxiosResponse)
        .mockResolvedValueOnce({
          data: {
            comments: [...earlierComments, ...Array.from({ length: 5 }, (_, i) => mockComment({ id: 200 + i, content: `later ${i}` })), mockComment({ id: 2, content: 'deep comment' })],
            total: 21,
            limit: 20,
            offset: 0,
            hasMore: true,
          },
        } as AxiosResponse);

      // Position 15 (0-based) puts the target on the second page (index 1)
      // under COMMENTS_PER_PAGE=10, so fetching it in one shot needs limit=20.
      vi.mocked(animalCommentsApi.getPosition).mockResolvedValue({
        data: { found: true, offset: 15 },
      } as AxiosResponse);

      window.history.pushState({}, '', '/groups/1/animals/1/view?comment=2');
      renderDetailPage();

      await screen.findByText('deep comment');
      await waitFor(() => {
        expect(document.getElementById('comment-2')).toHaveClass('comment-card--highlighted');
      });

      expect(animalCommentsApi.getPosition).toHaveBeenCalledWith(1, 1, 2, { order: 'desc' });
      expect(animalCommentsApi.getAll).toHaveBeenCalledTimes(2);
      expect(animalCommentsApi.getAll).toHaveBeenLastCalledWith(1, 1, expect.objectContaining({ limit: 20, offset: 0 }));
    });

    it("shows a message when the deep-linked comment can't be located", async () => {
      vi.mocked(animalCommentsApi.getAll).mockResolvedValue({
        data: { comments: [], total: 0, limit: 10, offset: 0, hasMore: false },
      } as AxiosResponse);
      vi.mocked(animalCommentsApi.getPosition).mockResolvedValue({
        data: { found: false },
      } as AxiosResponse);

      window.history.pushState({}, '', '/groups/1/animals/1/view?comment=999');
      renderDetailPage();

      expect(await screen.findByText(/couldn.t be found/i)).toBeInTheDocument();
    });

    it('only clears the tag filter once per deep-linked id, so a filter picked afterward sticks', async () => {
      vi.mocked(animalCommentsApi.getAll).mockResolvedValue({
        data: {
          comments: [mockComment({ id: 2, content: 'unfiltered comment' })],
          total: 1,
          limit: 10,
          offset: 0,
          hasMore: false,
        },
      } as AxiosResponse);
      vi.mocked(commentTagsApi.getAll).mockResolvedValue({
        data: [{ id: 1, group_id: 1, name: 'urgent', color: '#FF0000' }],
      } as AxiosResponse);

      window.history.pushState({}, '', '/groups/1/animals/1/view?comment=2');
      renderDetailPage();

      await screen.findByText('unfiltered comment');
      await waitFor(() => {
        expect(document.getElementById('comment-2')).toHaveClass('comment-card--highlighted');
      });

      // Deep-link resolution is done for id 2 (locatedCommentIdRef is set) —
      // a filter the user picks afterward should stick, not get silently
      // cleared again by the same effect on its next run.
      const filterButton = await screen.findByLabelText('Filter by urgent');
      fireEvent.click(filterButton);

      await waitFor(() => {
        expect(filterButton).toHaveAttribute('aria-pressed', 'true');
      });
    });
  });

  describe('admin deleted comments scoping', () => {
    beforeEach(() => {
      localStorage.setItem('token', 'test-token');
      vi.mocked(authApi.getCurrentUser).mockResolvedValue({
        data: {
          id: 1,
          username: 'admin',
          email: 'admin@example.com',
          phone_number: '',
          hide_email: false,
          hide_phone_number: false,
          is_admin: true,
        },
      } as AxiosResponse);
      mockAnimal({ status: 'available' });
    });

    it('does not show the deleted-comments toggle when only a different animal in the group has deleted comments', async () => {
      vi.mocked(animalCommentsApi.getDeleted).mockResolvedValue({
        data: [mockDeletedComment({ animal_id: 2 })],
      } as AxiosResponse);

      renderDetailPage();

      await screen.findByRole('heading', { name: 'Rex' });
      expect(screen.queryByText(/Show Deleted Comments/)).not.toBeInTheDocument();
    });

    it('resets the show-deleted toggle when navigating to a different animal', async () => {
      vi.mocked(animalCommentsApi.getDeleted).mockResolvedValue({
        data: [
          mockDeletedComment({ id: 1, animal_id: 1 }),
          mockDeletedComment({ id: 2, animal_id: 2 }),
        ],
      } as AxiosResponse);

      const { rerender } = renderDetailPage();

      const toggle = await screen.findByLabelText(/Show Deleted Comments/);
      fireEvent.click(toggle);
      expect(await screen.findByText('🗑️ Deleted Comments (Admin Review)')).toBeInTheDocument();

      mockRouteParams = { groupId: '1', id: '2' };
      rerender(
        <BrowserRouter>
          <AuthProvider>
            <ToastProvider>
              <AnimalDetailPage />
            </ToastProvider>
          </AuthProvider>
        </BrowserRouter>
      );

      await screen.findByText(/Show Deleted Comments \(1\)/);
      expect(screen.queryByText('🗑️ Deleted Comments (Admin Review)')).not.toBeInTheDocument();
    });
  });
});
