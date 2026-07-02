import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
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

    // Computed default from a 2026-06-22 Monday start would be Jul 6, 2026 (10 days,
    // no weekend roll needed) — very different from the stored override below, so this
    // assertion would fail if the code still computed the fallback instead of reading
    // the stored field.
    expect(await screen.findByText(/Ends: Jul 15, 2026/)).toBeInTheDocument();
    expect(screen.queryByText(/Ends: Jul 6, 2026/)).not.toBeInTheDocument();
  });
});
