// Shared with GroupPage.tsx's site-admin "viewing as" preview, which
// persists the active preview role per-group in sessionStorage rather than
// component state alone (so it survives tab-switching within the group).
const PREFIX = 'previewRole:';

export const previewRoleKey = (groupId: string | number) => `${PREFIX}${groupId}`;

// sessionStorage isn't cleared by logging out - only by closing the tab - so
// without this, a preview role chosen by one site admin would still be
// sitting in storage for whichever user logs into the same browser tab
// next, showing them a stale "Previewing as ..." banner (or, worse, a
// group-admin/member view that isn't their own real membership). Call this
// from every code path that invalidates the current session.
export function clearPreviewRoleState() {
  try {
    for (let i = sessionStorage.length - 1; i >= 0; i--) {
      const key = sessionStorage.key(i);
      if (key?.startsWith(PREFIX)) {
        sessionStorage.removeItem(key);
      }
    }
  } catch {
    // ignore storage errors
  }
}
