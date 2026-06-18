export function formatAnimalStatus(status: string): string {
  if (status === 'bite_quarantine') return 'Bite Quarantine';
  return status.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}
