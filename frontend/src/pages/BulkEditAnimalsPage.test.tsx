import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import BulkEditAnimalsPage from './BulkEditAnimalsPage';
import { animalsApi, groupsApi } from '../api/client';
import type { Animal, Group } from '../api/client';
import type { AxiosResponse } from 'axios';
import { ToastProvider } from '../contexts/ToastContext';

// Mock the API client
vi.mock('../api/client', () => ({
  animalsApi: {
    getAllForAdmin: vi.fn(),
  },
  groupsApi: {
    getAll: vi.fn(),
  },
}));

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

const mockGroup: Group = {
  id: 1,
  name: 'Test Group',
  description: '',
  image_url: '',
  hero_image_url: '',
  has_protocols: false,
  groupme_enabled: false,
};

describe('BulkEditAnimalsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(animalsApi.getAllForAdmin).mockResolvedValue({
      data: [quarantinedAnimal],
    } as AxiosResponse<Animal[]>);

    vi.mocked(groupsApi.getAll).mockResolvedValue({
      data: [mockGroup],
    } as AxiosResponse<Group[]>);
  });

  const renderPage = () => {
    return render(
      <BrowserRouter>
        <ToastProvider>
          <BulkEditAnimalsPage />
        </ToastProvider>
      </BrowserRouter>
    );
  };

  it('shows the stored quarantine end date, not a recomputed one, in card view', async () => {
    renderPage();

    // Computed default from a 2026-06-22 Monday start would be 2026-07-06 (10 days,
    // no weekend roll needed) — very different from the stored override below, so this
    // assertion would fail if the code still computed the fallback instead of reading
    // the stored field.
    expect(await screen.findByText('Jul 15, 2026')).toBeInTheDocument();
    expect(screen.queryByText('Jul 6, 2026')).not.toBeInTheDocument();
  });

  it('shows the stored quarantine end date, not a recomputed one, in table view', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('Rex');

    await user.click(screen.getByRole('button', { name: 'Table view' }));

    await waitFor(() => {
      expect(screen.getByText('Jul 15, 2026')).toBeInTheDocument();
    });
    expect(screen.queryByText('Jul 6, 2026')).not.toBeInTheDocument();
  });
});
