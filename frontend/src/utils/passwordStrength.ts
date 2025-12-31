/**
 * Password strength utility
 * Shared between UsersPage and SetupPassword components
 */

export interface PasswordStrength {
  strength: 'weak' | 'medium' | 'strong';
  label: string;
  color: string;
}

/**
 * Evaluates password strength based on length and character variety
 * @param password - The password to evaluate
 * @returns PasswordStrength object with strength level, label, and color
 */
export function getPasswordStrength(password: string): PasswordStrength {
  if (password.length < 8) {
    return { strength: 'weak', label: 'Too short', color: '#ef4444' };
  }
  
  let score = 0;
  if (password.length >= 12) score++;
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score++;
  if (/[0-9]/.test(password)) score++;
  if (/[^a-zA-Z0-9]/.test(password)) score++;
  
  if (score <= 1) {
    return { strength: 'weak', label: 'Weak', color: '#ef4444' };
  }
  if (score <= 2) {
    return { strength: 'medium', label: 'Medium', color: '#f59e0b' };
  }
  return { strength: 'strong', label: 'Strong', color: '#10b981' };
}
