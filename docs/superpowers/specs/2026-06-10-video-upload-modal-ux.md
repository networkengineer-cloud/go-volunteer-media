# Video Upload Modal UX Improvements

**Date:** 2026-06-10
**Status:** Approved

## Overview

The current video upload modal uses a raw native `<input type="file">` button, shows no visual preview of the selected clip, and is wired up with a hand-rolled `modal-overlay`/`modal-content` wrapper instead of the project's `Modal` component. This spec covers a focused UX polish pass: replace the native file input with a styled drop zone, surface the already-extracted thumbnail as a preview banner once a file is selected, and migrate to the proper `Modal` component.

No changes to upload logic, thumbnail extraction, API calls, or backend.

## Components Affected

- `frontend/src/components/VideoUpload.tsx` — primary changes
- `frontend/src/pages/PhotoGallery.tsx` — swap modal wrapper
- New CSS in a `VideoUpload.css` file alongside the component

## Design

### Two Modal States

**State 1 — No file selected:**

- Modal header (from `Modal` component): title "Upload Video" + close button
- Drop zone: dashed border, film emoji, "Tap to choose a video", subtext "MP4 or MOV · up to 200 MB". Clicking anywhere in the zone opens the hidden file input.
- Caption field: visible but `disabled` (grayed out) until a file is selected
- Upload button: `disabled`, muted style
- Cancel button: enabled

**State 2 — File selected, ready to upload:**

- Drop zone replaced by a thumbnail banner (same height): shows the extracted JPEG thumbnail as a background image (or a dark gradient placeholder while the thumbnail promise is still resolving), a centered play-icon circle, filename + file size in a bottom-gradient overlay, and a ✕ button in the top-right corner
- Tapping the banner body re-opens the native file picker so the user can swap the clip
- Tapping the ✕ button clears the selection and returns to State 1 (drop zone shown, file picker not auto-opened)
- Caption field: enabled, full styling
- Upload button: enabled, brand color
- Cancel button: enabled

**State 3 — Uploading:**

- Thumbnail banner stays visible (no change)
- Caption field: `disabled`
- Upload button: `disabled`, label changes to "Uploading…" with a small inline spinner (CSS animation, no library)
- Cancel button: `disabled`

### Error Display

Error message renders as red text between the caption field and the action buttons — same position as today. Errors clear on new file selection.

### Thumbnail Banner Details

The thumbnail extraction promise (`thumbnailPromiseRef`) is already kicked off eagerly on file select. The banner renders immediately on file select; it shows the gradient placeholder until the promise resolves, then sets the thumbnail as a CSS `background-image`. If the promise rejects, the placeholder gradient remains (no error shown in the banner — the extraction failure only surfaces if the user tries to upload).

### Modal Integration

`PhotoGallery.tsx` currently wraps `VideoUpload` in raw `<div className="modal-overlay"><div className="modal-content">`. This is replaced by the project's `Modal` component:

```tsx
<Modal
  isOpen={showVideoUpload}
  onClose={() => { setShowVideoUpload(false); setPendingVideoFile(null); }}
  title="Upload Video"
  size="small"
>
  <VideoUpload ... />
</Modal>
```

The `Modal` component already handles: backdrop blur, Escape key to close, focus trap, body scroll lock, entrance animation, and responsive sizing. `VideoUpload` no longer renders its own `<h3>Upload Video</h3>` heading (the `Modal` header provides it). The `onCancel` prop on `VideoUpload` calls `onClose` on the modal.

## CSS

A new `VideoUpload.css` file handles:

- `.video-upload` — flex column layout, gap between elements
- `.video-upload__dropzone` — dashed border, padding, center-aligned, cursor pointer, hover state (border color shift)
- `.video-upload__dropzone--active` — style for drag-over state (border color, background tint)
- `.video-upload__thumbnail` — relative positioned container, fixed height (140px), border-radius, overflow hidden; background-image set inline via style prop
- `.video-upload__thumbnail-overlay` — absolute inset, dark gradient for readability
- `.video-upload__thumbnail-play` — centered circle with play icon
- `.video-upload__thumbnail-meta` — bottom overlay with filename/size
- `.video-upload__thumbnail-repick` — ✕ button, top-right corner
- `.video-upload__caption` — full-width text input, consistent with other form fields in the app
- `.video-upload__error` — red error text
- `.video-upload__spinner` — small CSS keyframe spinner shown inside the upload button during uploading state

Dark mode variants for drop zone and thumbnail meta where needed.

## Out of Scope

- Upload progress bar (option C — deferred; requires XHR progress events wired into the API client)
- Drag-and-drop file detection on desktop (drop zone is tap/click only for now)
- Video playback preview inside the modal
