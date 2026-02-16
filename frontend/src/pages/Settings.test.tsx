import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import Settings from './Settings';
import { authApi, settingsApi } from '../api/client';
import { AxiosResponse } from 'axios';
import { ToastProvider } from '../contexts/ToastContext';
import { SiteSettingsProvider } from '../contexts/SiteSettingsContext';

// Mock the API client
vi.mock('../api/client', () => ({
  authApi: {
    getCurrentUser: vi.fn(),
    getEmailPreferences: vi.fn(),
    updateEmailPreferences: vi.fn(),
    updateCurrentUserProfile: vi.fn(),
  },
  settingsApi: {
    getAll: vi.fn(),
    update: vi.fn(),
    uploadHeroImage: vi.fn(),
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

    // Default happy-path mocks so the page can render.
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
      data: {
        email_notifications_enabled: false,
        show_length_of_stay: false,
      },
    } as AxiosResponse);

    // Mock site settings API
    vi.mocked(settingsApi.getAll).mockResolvedValue({
      data: {
        site_name: 'MyHAWS',
        site_short_name: 'MyHAWS',
        site_description: 'MyHAWS Volunteer Portal',
        hero_image_url: '',
      },
    } as AxiosResponse);
  });

  const renderSettings = () => {
    return render(
      <BrowserRouter>
        <SiteSettingsProvider>
          <ToastProvider>
            <Settings />
          </ToastProvider>
        </SiteSettingsProvider>
      </BrowserRouter>
    );
  };

  describe('Loading state', () => {
    it('should display loading indicator while fetching settings', () => {
      vi.mocked(authApi.getEmailPreferences).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      renderSettings();
      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });
  });

  describe('Successful preference loading', () => {
    it('should load and display email notifications when enabled', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i) as HTMLInputElement;
      expect(checkbox.checked).toBe(true);
    });

    it('should load and display email notifications when disabled', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i) as HTMLInputElement;
      expect(checkbox.checked).toBe(false);
    });

    it('should call the correct API endpoint', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(authApi.getCurrentUser).toHaveBeenCalledTimes(1);
        expect(authApi.getEmailPreferences).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('Error handling', () => {
    it('should display error message when loading settings fails', async () => {
      vi.mocked(authApi.getEmailPreferences).mockRejectedValue(
        new Error('Network error')
      );

      renderSettings();

      await waitFor(() => {
        expect(screen.getByText(/failed to load settings/i)).toBeInTheDocument();
      });
    });
  });

  describe('Saving preferences', () => {
    it('should save email notification preference when save button is clicked', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockResolvedValue({
        data: { message: 'Success', email_notifications_enabled: true, show_length_of_stay: false },
      } as AxiosResponse<{ message: string; email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i);
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save email preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(authApi.updateEmailPreferences).toHaveBeenCalledWith(true, false);
        expect(screen.getByText(/email preferences saved successfully/i)).toBeInTheDocument();
      });
    });

    it('should show error message when save fails', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockRejectedValue({
        response: { data: { error: 'Failed to update' } },
      });

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i);
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save email preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText(/failed to update/i)).toBeInTheDocument();
      });
    });

    it('should disable save button while saving', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: false, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);
      vi.mocked(authApi.updateEmailPreferences).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const user = userEvent.setup();
      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i);
      await user.click(checkbox);

      const saveButton = screen.getByRole('button', { name: /save email preferences/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /^saving\.\.\.$/i })).toBeDisabled();
      });
    });
  });

  describe('Navigation', () => {
    it('should navigate to dashboard when back button is clicked', async () => {
      vi.mocked(authApi.getEmailPreferences).mockResolvedValue({
        data: { email_notifications_enabled: true, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

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
        data: { email_notifications_enabled: true, show_length_of_stay: false },
      } as AxiosResponse<{ email_notifications_enabled: boolean; show_length_of_stay: boolean }>);

      renderSettings();

      await waitFor(() => {
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
      });

      const checkbox = screen.getByLabelText(/receive announcement emails/i);
      expect(checkbox).toHaveAttribute('id', 'email-notifications');
    });
  });
});
