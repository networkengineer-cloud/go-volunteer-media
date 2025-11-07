import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import Settings from './Settings';
import { authApi } from '../api/client';
import { AxiosResponse } from 'axios';

// Mock the API client
vi.mock('../api/client', () => ({
  authApi: {
    getEmailPreferences: vi.fn(),
    updateEmailPreferences: vi.fn(),
  },
}));

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('Settings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderSettings = () => {
    return render(
      <BrowserRouter>
        <Settings />
      </BrowserRouter>
    );
  };

  describe('Loading state', () => {
    it('should display loading indicator while fetching preferences', () => {
      vi.mocked(authApi.getEmailPreferences).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      renderSettings();
      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });
  });

  describe('Successful preference loading', () => {
    it('should load and display email preferences when enabled', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox') as HTMLInputElement;
      expect(checkbox.checked).toBe(true);
    });

    it('should load and display email preferences when disabled', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox') as HTMLInputElement;
      expect(checkbox.checked).toBe(false);
    });

    it('should call the correct API endpoint', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(authApi.getEmailPreferences).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('Error handling', () => {
    it('should display error message when loading preferences fails', async () => {
      vi.mocked(authApi.getEmailPreferences).mockRejectedValue(
        new Error('Network error')
      );

      renderSettings();

      await waitFor(() => {
        expect(screen.getByText(/failed to load preferences/i)).toBeInTheDocument();
      });
    });
  });

  describe('Saving preferences', () => {
    it('should save preferences when save button is clicked', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockResolvedValue({
        data: { message: 'Success', email_notifications_enabled: true },
      } as AxiosResponse<{ message: string; email_notifications_enabled: boolean }>);

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox');
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(authApi.updateEmailPreferences).toHaveBeenCalledWith(true);
        expect(screen.getByText(/preferences saved successfully/i)).toBeInTheDocument();
      });
    });

    it('should show error message when save fails', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockRejectedValue({
        response: { data: { error: 'Failed to update' } },
      });

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox');
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText(/failed to update/i)).toBeInTheDocument();
      });
    });

    it('should disable save button while saving', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox');
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled();
      });
    });
  });

  describe('Navigation', () => {
    it('should navigate to dashboard when back button is clicked', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const backButton = screen.getByRole('button', { name: /back to dashboard/i });
      await user.click(backButton);

      expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
    });
  });

  describe('Accessibility', () => {
    it('should have proper labels for checkbox', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true },
      } as AxiosResponse<{ email_notifications_enabled: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByRole('checkbox');
      expect(checkbox).toHaveAttribute('id', 'email-notifications');
      
      const label = screen.getByLabelText(/receive announcement emails/i);
      expect(label).toBeInTheDocument();
    });
  });
});
