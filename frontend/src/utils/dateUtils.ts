export function formatDateShort(dateString?: string): string {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatDateLong(dateString?: string): string {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' });
}

export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return formatDateShort(dateString);
}

// Calculates the quarantine end date: 10 days after the start date,
// shifted forward if it falls on a weekend.
export function calculateQuarantineEndDate(startDateString?: string, long = false): string {
  if (!startDateString) return '-';

  const startDate = new Date(startDateString);
  const endDate = new Date(startDate);
  endDate.setDate(endDate.getDate() + 10);

  while (endDate.getDay() === 0 || endDate.getDay() === 6) {
    endDate.setDate(endDate.getDate() + 1);
  }

  return long ? formatDateLong(endDate.toISOString()) : formatDateShort(endDate.toISOString());
}

export function calculateDaysSince(dateString?: string): number {
  if (!dateString) return 0;
  const date = new Date(dateString);
  const now = new Date();
  const diffTime = Math.abs(now.getTime() - date.getTime());
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
}
