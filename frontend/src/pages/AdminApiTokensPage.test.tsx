import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import type { AxiosResponse } from 'axios';
import AdminApiTokensPage from './AdminApiTokensPage';
import { apiTokensApi } from '../api/client';

vi.mock('../api/client', () => ({
  apiTokensApi: {
    list: vi.fn(),
    create: vi.fn(),
    revoke: vi.fn(),
  },
}));

const mockToken = {
  id: 1,
  name: 'Zapier integration',
  token_prefix: 'pat_ab12cd34',
  created_at: '2026-01-01T00:00:00Z',
  expires_at: '2026-12-31T00:00:00Z',
  last_used_at: null,
};

const renderPage = () =>
  render(
    <BrowserRouter>
      <AdminApiTokensPage />
    </BrowserRouter>
  );

describe('AdminApiTokensPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiTokensApi.list).mockResolvedValue({ data: [] } as AxiosResponse);
  });

  it('renders existing tokens', async () => {
    vi.mocked(apiTokensApi.list).mockResolvedValue({ data: [mockToken] } as AxiosResponse);

    renderPage();

    expect(await screen.findByText('Zapier integration')).toBeInTheDocument();
    expect(screen.getByText(/pat_ab12cd34/)).toBeInTheDocument();
  });

  it('shows an empty state when there are no tokens', async () => {
    renderPage();

    expect(await screen.findByText(/no api tokens yet/i)).toBeInTheDocument();
  });

  it('creates a token and shows the one-time secret', async () => {
    const user = userEvent.setup();
    vi.mocked(apiTokensApi.create).mockResolvedValue({
      data: { ...mockToken, token: 'pat_' + 'a'.repeat(64) },
    } as AxiosResponse);

    renderPage();
    await waitFor(() => expect(apiTokensApi.list).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /generate token/i }));
    await user.type(screen.getByLabelText(/name/i), 'Zapier integration');
    await user.click(screen.getByRole('button', { name: /^create$/i }));

    expect(await screen.findByText(/pat_a{64}/)).toBeInTheDocument();
    expect(screen.getByText(/won.t be shown again/i)).toBeInTheDocument();
  });

  it('shows an inline error and keeps the modal open when token creation fails', async () => {
    const user = userEvent.setup();
    vi.mocked(apiTokensApi.create).mockRejectedValue(new Error('network error'));

    renderPage();
    await waitFor(() => expect(apiTokensApi.list).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /generate token/i }));
    await user.type(screen.getByLabelText(/name/i), 'Zapier integration');
    await user.click(screen.getByRole('button', { name: /^create$/i }));

    expect(await screen.findByText(/failed to create api token/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^create$/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
  });

  it('revokes a token after confirmation', async () => {
    const user = userEvent.setup();
    vi.mocked(apiTokensApi.list).mockResolvedValue({ data: [mockToken] } as AxiosResponse);
    vi.mocked(apiTokensApi.revoke).mockResolvedValue({ data: { message: 'ok' } } as AxiosResponse);

    renderPage();
    await screen.findByText('Zapier integration');

    await user.click(screen.getByRole('button', { name: /revoke/i }));
    await user.click(screen.getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(apiTokensApi.revoke).toHaveBeenCalledWith(1));
  });
});
