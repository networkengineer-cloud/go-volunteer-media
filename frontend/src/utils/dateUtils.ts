export function formatDateShort(dateString?: string): string {
  if (!dateString) return '-';
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '-';
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatDateLong(dateString?: string): string {
  if (!dateString) return '-';
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '-';
  return date.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' });
}

export function formatRelativeTime(dateString: string, cutoffDays = 30): string {
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return '-';
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();

  if (diffMs < 0) return formatDateShort(dateString); // future date

  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < cutoffDays) return `${diffDays}d ago`;
  return formatDateShort(dateString);
}

// Calculates the quarantine end date: 10 days after the start date,
// shifted forward if it falls on a weekend.
export function calculateQuarantineEndDate(startDateString?: string, format: 'short' | 'long' = 'short'): string {
  if (!startDateString) return '-';

  const startDate = new Date(startDateString);
  if (isNaN(startDate.getTime())) return '-';
  const endDate = new Date(startDate);
  endDate.setDate(endDate.getDate() + 10);

  while (endDate.getDay() === 0 || endDate.getDay() === 6) {
    endDate.setDate(endDate.getDate() + 1);
  }

  return endDate.toLocaleDateString('en-US', format === 'long'
    ? { year: 'numeric', month: 'long', day: 'numeric' }
    : { month: 'short', day: 'numeric', year: 'numeric' });
}

export function calculateDaysSince(dateString?: string): number {
  if (!dateString) return 0;
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return 0;
  const now = new Date();
  const diffTime = now.getTime() - date.getTime();
  if (diffTime < 0) return 0; // future date
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
}

/**
 * Calculate age in years and months from a birth date string.
 * Falls back to { years: fallbackYears, months: 0 } when birthDate is missing/invalid.
 */
export function calculateAge(birthDateString?: string, fallbackYears?: number): { years: number; months: number } {
  if (!birthDateString) {
    return { years: fallbackYears ?? 0, months: 0 };
  }
  const bd = new Date(birthDateString);
  if (isNaN(bd.getTime())) {
    return { years: fallbackYears ?? 0, months: 0 };
  }
  const now = new Date();
  let years = now.getFullYear() - bd.getFullYear();
  let months = now.getMonth() - bd.getMonth();
  if (now.getDate() < bd.getDate()) {
    months--;
  }
  if (months < 0) {
    years--;
    months += 12;
  }
  if (years < 0) {
    return { years: 0, months: 0 };
  }
  return { years, months };
}

/**
 * Format age as a human-readable string.
 * Examples: "1 yr 6 mo", "2 yrs", "3 mo", "< 1 mo"
 */
export function formatAge(years: number, months: number): string {
  if (years <= 0 && months <= 0) return '< 1 mo';
  const parts: string[] = [];
  if (years > 0) {
    parts.push(`${years} ${years === 1 ? 'yr' : 'yrs'}`);
  }
  if (months > 0) {
    parts.push(`${months} mo`);
  }
  return parts.join(' ') || '< 1 mo';
}

/**
 * Compute an estimated birth date by subtracting years and months from today.
 * The day-of-month is today's day (implied from when the record is entered).
 */
export function computeEstimatedBirthDate(years: number, months: number): string {
  const now = new Date();
  const bd = new Date(now.getFullYear() - years, now.getMonth() - months, now.getDate());
  // Format as YYYY-MM-DD
  const y = bd.getFullYear();
  const m = String(bd.getMonth() + 1).padStart(2, '0');
  const d = String(bd.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}
