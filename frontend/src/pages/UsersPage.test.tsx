import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import type { AxiosResponse } from 'axios';
import UsersPage from './UsersPage';
import { usersApi, groupsApi, statisticsApi } from '../api/client';
import { useAuth } from '../hooks/useAuth';

// Mock the API client
vi.mock('../api/client', () => ({
  usersApi: {
    getAll: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    promote: vi.fn(),
    demote: vi.fn(),
    getDeleted: vi.fn(),
    restore: vi.fn(),
    resetPassword: vi.fn(),
    resendInvitation: vi.fn(),
    assignGroup: vi.fn(),
    removeGroup: vi.fn(),
    unlock: vi.fn(),
  },
  groupsApi: {
    getAll: vi.fn(),
    getMembers: vi.fn(),
  },
  groupAdminApi: {
    createUser: vi.fn(),
    promoteToGroupAdmin: vi.fn(),
    demoteFromGroupAdmin: vi.fn(),
    addMemberToGroup: vi.fn(),
    removeMemberFromGroup: vi.fn(),
  },
  statisticsApi: {
    getUserStatistics: vi.fn(),
  },
}));

// Mock the auth hook
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

const mockGroups = [
  { id: 1, name: 'Dogs', description: 'Dog group', image_url: '', hero_image_url: '', has_protocols: false, groupme_bot_id: '', groupme_enabled: false },
  { id: 2, name: 'Cats', description: 'Cat group', image_url: '', hero_image_url: '', has_protocols: false, groupme_bot_id: '', groupme_enabled: false },
];

const mockUsers = [
  { id: 1, username: 'admin_user', email: 'admin@example.com', is_admin: true, groups: [], first_name: 'Admin', last_name: 'User' },
  { id: 2, username: 'john_doe', email: 'john@example.com', is_admin: false, groups: [mockGroups[0]], first_name: 'John', last_name: 'Doe' },
  { id: 3, username: 'jane_smith', email: 'jane@example.com', is_admin: false, groups: [mockGroups[1]], first_name: '', last_name: '' },
];

const mockAdminAuth = {
  user: { id: 1, username: 'admin_user', email: 'admin@example.com', is_admin: true, groups: [] },
  isAdmin: true,
  isGroupAdmin: false,
  isAuthenticated: true,
  isLoading: false,
  token: 'admin-token',
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
};

const mockRegularAuth = {
  user: { id: 2, username: 'john_doe', email: 'john@example.com', is_admin: false, groups: [] },
  isAdmin: false,
  isGroupAdmin: false,
  isAuthenticated: true,
  isLoading: false,
  token: 'user-token',
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
};

describe('UsersPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Default: admin user
    vi.mocked(useAuth).mockReturnValue(mockAdminAuth);

    // Default happy-path API mocks
    vi.mocked(usersApi.getAll).mockResolvedValue({
      data: { data: mockUsers, total: 3, limit: 100, offset: 0, hasMore: false },
    } as AxiosResponse);

    vi.mocked(statisticsApi.getUserStatistics).mockResolvedValue({
      data: { data: [], total: 0, limit: 100, offset: 0, hasMore: false },
    } as AxiosResponse);

    vi.mocked(groupsApi.getAll).mockResolvedValue({
      data: mockGroups,
    } as AxiosResponse);

    vi.mocked(groupsApi.getMembers).mockResolvedValue({
      data: [],
    } as AxiosResponse);
  });

  const renderUsersPage = () =>
    render(
      <BrowserRouter>
        <UsersPage />
      </BrowserRouter>
    );

  const waitForLoad = () =>
    waitFor(() => {
      expect(screen.queryAllByRole('status')).toHaveLength(0);
    });

  // ---------------------------------------------------------------------------
  // Loading state
  // ---------------------------------------------------------------------------
  describe('Loading state', () => {
    it('shows loading indicator while fetching users', () => {
      vi.mocked(usersApi.getAll).mockImplementation(() => new Promise(() => {}));
      renderUsersPage();
      expect(screen.getAllByRole('status').length).toBeGreaterThan(0);
    });
  });

  // ---------------------------------------------------------------------------
  // Error state
  // ---------------------------------------------------------------------------
  describe('Error state', () => {
    it('shows a fallback error message when the API call fails', async () => {
      vi.mocked(usersApi.getAll).mockRejectedValue(new Error('Network error'));
      renderUsersPage();
      await waitFor(() => {
        expect(screen.getByText(/failed to fetch users/i)).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Admin view
  // ---------------------------------------------------------------------------
  describe('Admin view', () => {
    it('renders "Manage Users" heading for admins', async () => {
      renderUsersPage();
      await waitForLoad();
      expect(screen.getByRole('heading', { name: /manage users/i })).toBeInTheDocument();
    });

    it('shows "Add User" button for admins', async () => {
      renderUsersPage();
      await waitForLoad();
      expect(screen.getByRole('button', { name: /add user/i })).toBeInTheDocument();
    });

    it('renders a card for each user after loading', async () => {
      renderUsersPage();
      expect(await screen.findByText('admin_user')).toBeInTheDocument();
      expect(screen.getByText('john_doe')).toBeInTheDocument();
      expect(screen.getByText('jane_smith')).toBeInTheDocument();
    });

    it('displays full name on user cards when first and last name are set', async () => {
      renderUsersPage();
      expect(await screen.findByText('Admin User')).toBeInTheDocument();
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Non-admin / non-group-admin view
  // ---------------------------------------------------------------------------
  describe('Non-admin view', () => {
    beforeEach(() => {
      vi.mocked(useAuth).mockReturnValue(mockRegularAuth);
    });

    it('renders "Team Members" heading for non-admins', async () => {
      renderUsersPage();
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /team members/i })).toBeInTheDocument();
      });
    });

    it('does not show "Add User" button when user has no management permissions', async () => {
      renderUsersPage();
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /team members/i })).toBeInTheDocument();
      });
      expect(screen.queryByRole('button', { name: /add user/i })).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Search / filter
  // ---------------------------------------------------------------------------
  describe('Search', () => {
    it('filters the user list by username', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.type(screen.getByPlaceholderText(/search by username or email/i), 'john');

      await waitFor(() => {
        expect(screen.getByText('john_doe')).toBeInTheDocument();
        expect(screen.queryByText('jane_smith')).not.toBeInTheDocument();
      });
    });

    it('filters the user list by email', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.type(screen.getByPlaceholderText(/search by username or email/i), 'jane@');

      await waitFor(() => {
        expect(screen.getByText('jane_smith')).toBeInTheDocument();
        expect(screen.queryByText('john_doe')).not.toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Create User form — opening and fields
  // ---------------------------------------------------------------------------
  describe('Create User form — structure', () => {
    it('opens the form when "Add User" is clicked', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      expect(screen.getByRole('heading', { name: /add new user/i })).toBeInTheDocument();
    });

    it('renders username, first name, last name, and email fields', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/first name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/last name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/email address/i)).toBeInTheDocument();
    });

    it('shows the "send setup email" checkbox checked by default', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      const checkbox = screen.getByRole('checkbox', {
        name: /send password setup email/i,
      }) as HTMLInputElement;
      expect(checkbox.checked).toBe(true);
    });

    it('hides the password field when "send setup email" is checked', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      // Default is checked — password should not be visible
      expect(screen.queryByLabelText(/^password/i)).not.toBeInTheDocument();
    });

    it('shows the password field when "send setup email" is unchecked', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.click(screen.getByRole('checkbox', { name: /send password setup email/i }));

      await waitFor(() => {
        expect(screen.getByLabelText(/^password/i)).toBeInTheDocument();
      });
    });

    it('enforces maxLength={100} on first name and last name inputs', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      const firstNameInput = screen.getByLabelText(/first name/i) as HTMLInputElement;
      const lastNameInput = screen.getByLabelText(/last name/i) as HTMLInputElement;

      expect(firstNameInput).toHaveAttribute('maxLength', '100');
      expect(lastNameInput).toHaveAttribute('maxLength', '100');
    });

    it('closes the form when Cancel is clicked', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      expect(screen.getByRole('heading', { name: /add new user/i })).toBeInTheDocument();

      // Two "Cancel" buttons exist: the toolbar toggle and the form button.
      // Click the form's own Cancel (btn-secondary).
      const cancelButtons = screen.getAllByRole('button', { name: /cancel/i });
      await user.click(cancelButtons[cancelButtons.length - 1]);

      await waitFor(() => {
        expect(screen.queryByRole('heading', { name: /add new user/i })).not.toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Create User form — validation
  // ---------------------------------------------------------------------------
  describe('Create User form — validation', () => {
    it('shows a username error when username is too short', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      const usernameInput = screen.getByLabelText(/username/i);
      await user.type(usernameInput, 'ab');
      await user.tab();

      await waitFor(() => {
        expect(screen.getByText(/at least 3 characters/i)).toBeInTheDocument();
      });
    });

    it('shows an email error for an invalid email', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      const emailInput = screen.getByLabelText(/email address/i);
      await user.type(emailInput, 'not-valid');
      await user.tab();

      await waitFor(() => {
        expect(screen.getByText(/valid email address/i)).toBeInTheDocument();
      });
    });

    it('shows a password error when sending setup email is disabled and no password is given', async () => {
      const user = userEvent.setup();
      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));

      // Fill required fields to avoid unrelated validation errors
      await user.type(screen.getByLabelText(/username/i), 'validuser');
      await user.type(screen.getByLabelText(/email address/i), 'valid@example.com');

      // Disable setup email to expose password field
      await user.click(screen.getByRole('checkbox', { name: /send password setup email/i }));

      // Submit without filling in password
      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(screen.getByText(/password is required/i)).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Create User form — successful submission
  // ---------------------------------------------------------------------------
  describe('Create User form — submission', () => {
    it('sends trimmed first_name and last_name to the API', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockResolvedValue({
        data: {
          user: { id: 4, username: 'newuser', email: 'new@example.com', is_admin: false },
          message: 'User created successfully. Password setup email sent to new@example.com',
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'newuser');
      await user.type(screen.getByLabelText(/first name/i), '  Jane  ');
      await user.type(screen.getByLabelText(/last name/i), '  Doe  ');
      await user.type(screen.getByLabelText(/email address/i), 'new@example.com');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(usersApi.create).toHaveBeenCalledWith(
          expect.objectContaining({
            username: 'newuser',
            first_name: 'Jane',
            last_name: 'Doe',
            email: 'new@example.com',
            send_setup_email: true,
          })
        );
      });
    });

    it('omits first_name and last_name from the API call when left blank', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockResolvedValue({
        data: {
          user: { id: 4, username: 'newuser', email: 'new@example.com', is_admin: false },
          message: 'User created successfully.',
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'newuser');
      // Leave first/last name blank
      await user.type(screen.getByLabelText(/email address/i), 'new@example.com');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(usersApi.create).toHaveBeenCalledWith(
          expect.objectContaining({
            first_name: undefined,
            last_name: undefined,
          })
        );
      });
    });

    it('shows a success message after creation with setup email (wrapped response)', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockResolvedValue({
        data: {
          user: { id: 4, username: 'newuser', email: 'new@example.com', is_admin: false },
          message: 'User created successfully. Password setup email sent to new@example.com',
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'newuser');
      await user.type(screen.getByLabelText(/email address/i), 'new@example.com');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(screen.getByText(/password setup email sent/i)).toBeInTheDocument();
      });
    });

    it('shows a success message after creation with a direct password (User response)', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockResolvedValue({
        data: { id: 4, username: 'newuser', email: 'new@example.com', is_admin: false },
      } as AxiosResponse);

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'newuser');
      await user.type(screen.getByLabelText(/email address/i), 'new@example.com');

      // Switch to direct password mode
      await user.click(screen.getByRole('checkbox', { name: /send password setup email/i }));
      await user.type(screen.getByLabelText(/^password/i), 'securepass123');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(screen.getByText(/user created successfully/i)).toBeInTheDocument();
      });
    });

    it('shows a warning when user is created but setup email fails', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockResolvedValue({
        data: {
          user: { id: 4, username: 'newuser', email: 'new@example.com', is_admin: false },
          warning: 'User created successfully, but the setup email could not be sent.',
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'newuser');
      await user.type(screen.getByLabelText(/email address/i), 'new@example.com');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(screen.getByText(/setup email could not be sent/i)).toBeInTheDocument();
      });
    });

    it('shows an error message when the API returns a conflict', async () => {
      const user = userEvent.setup();
      vi.mocked(usersApi.create).mockRejectedValue({
        response: { data: { error: 'Username or email already exists' } },
      });

      renderUsersPage();
      await waitForLoad();

      await user.click(screen.getByRole('button', { name: /add user/i }));
      await user.type(screen.getByLabelText(/username/i), 'existinguser');
      await user.type(screen.getByLabelText(/email address/i), 'existing@example.com');

      await user.click(screen.getByRole('button', { name: /create user/i }));

      await waitFor(() => {
        expect(screen.getByText(/username or email already exists/i)).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Account lockout display — isUserLocked edge cases
  // ---------------------------------------------------------------------------
  describe('Account lockout display', () => {
    const futureTimestamp = new Date(Date.now() + 30 * 60 * 1000).toISOString(); // 30 min from now
    const pastTimestamp = new Date(Date.now() - 30 * 60 * 1000).toISOString();   // 30 min ago

    it('shows Locked badge for a user whose locked_until is in the future', async () => {
      vi.mocked(usersApi.getAll).mockResolvedValue({
        data: {
          data: [{ id: 10, username: 'locked_user', email: 'locked@example.com', is_admin: false, groups: [], locked_until: futureTimestamp, failed_login_attempts: 5 }],
          total: 1, limit: 100, offset: 0, hasMore: false,
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitFor(() => expect(screen.getByText('locked_user')).toBeInTheDocument());

      expect(screen.getByText('Locked')).toBeInTheDocument();
    });

    it('does not show Locked badge when locked_until is in the past', async () => {
      vi.mocked(usersApi.getAll).mockResolvedValue({
        data: {
          data: [{ id: 11, username: 'prev_locked', email: 'prev@example.com', is_admin: false, groups: [], locked_until: pastTimestamp, failed_login_attempts: 0 }],
          total: 1, limit: 100, offset: 0, hasMore: false,
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitFor(() => expect(screen.getByText('prev_locked')).toBeInTheDocument());

      expect(screen.queryByText('Locked')).not.toBeInTheDocument();
    });

    it('does not show Locked badge when locked_until is null', async () => {
      vi.mocked(usersApi.getAll).mockResolvedValue({
        data: {
          data: [{ id: 12, username: 'unlocked_user', email: 'unl@example.com', is_admin: false, groups: [], locked_until: null, failed_login_attempts: 0 }],
          total: 1, limit: 100, offset: 0, hasMore: false,
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitFor(() => expect(screen.getByText('unlocked_user')).toBeInTheDocument());

      expect(screen.queryByText('Locked')).not.toBeInTheDocument();
    });

    it('does not show Locked badge for a locked user when the viewer is non-admin', async () => {
      vi.mocked(useAuth).mockReturnValue(mockRegularAuth);
      // Non-admins with no groups see an empty list; locked badge is never rendered
      renderUsersPage();
      await waitFor(() => expect(screen.getByRole('heading', { name: /team members/i })).toBeInTheDocument());

      expect(screen.queryByText('Locked')).not.toBeInTheDocument();
    });

    it('shows Unlock Account button for admin viewing a locked user', async () => {
      vi.mocked(usersApi.getAll).mockResolvedValue({
        data: {
          data: [{ id: 14, username: 'btn_locked', email: 'btnl@example.com', is_admin: false, groups: [], locked_until: futureTimestamp, failed_login_attempts: 3 }],
          total: 1, limit: 100, offset: 0, hasMore: false,
        },
      } as AxiosResponse);

      renderUsersPage();
      await waitFor(() => expect(screen.getByText('btn_locked')).toBeInTheDocument());

      expect(screen.getByRole('button', { name: /unlock account/i })).toBeInTheDocument();
    });

    it('does not show Unlock Account button when viewer is non-admin', async () => {
      vi.mocked(useAuth).mockReturnValue(mockRegularAuth);
      // Non-admins with no groups see an empty list; Unlock Account button is never rendered
      renderUsersPage();
      await waitFor(() => expect(screen.getByRole('heading', { name: /team members/i })).toBeInTheDocument());

      expect(screen.queryByRole('button', { name: /unlock account/i })).not.toBeInTheDocument();
    });
  });
});
