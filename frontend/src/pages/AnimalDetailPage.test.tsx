import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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
