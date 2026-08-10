export const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
export const HOURS = [8, 9, 10, 11, 12, 13, 14, 15, 16, 17];

export function slotKey(dayOfWeek: number, hour: number): string {
  return `${dayOfWeek}-${hour}`;
}

export function formatHourLabel(hour: number): string {
  const period = hour < 12 ? 'AM' : 'PM';
  const displayHour = hour % 12 === 0 ? 12 : hour % 12;
  return `${displayHour}:00 ${period}`;
}
