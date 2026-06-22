import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import AnimalDetailPage from './AnimalDetailPage';
import { animalsApi, animalCommentsApi, commentTagsApi, groupsApi, authApi } from '../api/client';
import type { Animal } from '../api/client';
import type { AxiosResponse } from 'axios';
import { AuthProvider } from '../contexts/AuthContext';
import { ToastProvider } from '../contexts/ToastContext';

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
    useParams: () => ({ groupId: '1', id: '1' }),
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

describe('AnimalDetailPage', () => {
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
});
