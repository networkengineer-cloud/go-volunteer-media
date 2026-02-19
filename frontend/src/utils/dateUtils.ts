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
