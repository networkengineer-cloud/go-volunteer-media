import type { User } from '../api/client';

/**
 * Returns a display name for a user in public-facing contexts.
 * Format: "FirstName L." where L is the first letter of the last name.
 * Falls back to username if no name is available.
 */
export function formatDisplayName(user: Pick<User, 'username' | 'first_name' | 'last_name'>): string {
  const firstName = user.first_name?.trim();
  const lastName = user.last_name?.trim();
  if (firstName) {
    return lastName ? `${firstName} ${lastName.charAt(0).toUpperCase()}.` : firstName;
  }
  if (lastName) {
    return lastName;
  }
  return user.username;
}
