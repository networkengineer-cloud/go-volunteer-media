# Video Upload Modal UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the raw native file input and hand-rolled modal wrapper with a styled drop zone, thumbnail preview banner, and the project's `Modal` component.

**Architecture:** `VideoUpload.tsx` gains a hidden file input triggered by a styled drop zone (pre-selection) or thumbnail banner (post-selection). Thumbnail URL is derived from the already-eager extraction promise. `PhotoGallery.tsx` swaps its hand-rolled `modal-overlay`/`modal-content` wrapper for the `Modal` component.

**Tech Stack:** React, TypeScript, CSS custom properties, Vitest, @testing-library/react, @testing-library/user-event

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `frontend/src/components/VideoUpload.css` | Drop zone, thumbnail banner, caption, error, action layout |
| Modify | `frontend/src/components/VideoUpload.tsx` | New UI states, hidden input ref, thumbnail URL state, Button component |
| Create | `frontend/src/components/VideoUpload.test.tsx` | Tests for all three modal states |
| Modify | `frontend/src/pages/PhotoGallery.tsx` | Swap `modal-overlay`/`modal-content` for `Modal` component |

---

### Task 1: Create VideoUpload.css

**Files:**
- Create: `frontend/src/components/VideoUpload.css`

- [ ] **Step 1: Create the CSS file**

```css
.video-upload {
  display: flex;
  flex-direction: column;
  gap: var(--space-md, 1rem);
}

.video-upload__dropzone {
  border: 2px dashed var(--border, #cbd5e1);
  border-radius: var(--radius-md, 0.375rem);
  padding: 2rem 1rem;
  text-align: center;
  cursor: pointer;
  background: var(--surface-secondary, #f8fafc);
  transition: border-color 0.15s ease, background-color 0.15s ease;
  user-select: none;
}

.video-upload__dropzone:hover,
.video-upload__dropzone:focus-visible {
  border-color: var(--brand, #0e6c55);
  background: var(--neutral-50, #f0fdf4);
  outline: none;
}

.video-upload__dropzone-icon {
  font-size: 1.75rem;
  margin-bottom: 0.5rem;
  line-height: 1;
}

.video-upload__dropzone-label {
  font-weight: 500;
  color: var(--text-primary);
  font-size: var(--text-sm, 0.875rem);
}

.video-upload__dropzone-hint {
  color: var(--text-secondary, #6b7280);
  font-size: var(--text-xs, 0.75rem);
  margin-top: 0.25rem;
}

.video-upload__thumbnail {
  position: relative;
  border-radius: var(--radius-md, 0.375rem);
  overflow: hidden;
  height: 140px;
  background: #111;
  cursor: pointer;
}

.video-upload__thumbnail-bg {
  position: absolute;
  inset: 0;
  background-color: #1a3344;
  background-size: cover;
  background-position: center;
}

.video-upload__thumbnail-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(transparent 45%, rgba(0, 0, 0, 0.75) 100%);
}

.video-upload__thumbnail-play {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-upload__thumbnail-play-circle {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  border: 2px solid rgba(255, 255, 255, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 1.125rem;
  padding-left: 3px;
}

.video-upload__thumbnail-meta {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0.375rem 0.625rem 0.5rem;
  color: #fff;
  font-size: var(--text-xs, 0.75rem);
  font-weight: 500;
}

.video-upload__thumbnail-repick {
  position: absolute;
  top: 0.4rem;
  right: 0.4rem;
  width: 24px;
  height: 24px;
  padding: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: #fff;
  font-size: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s ease;
  line-height: 1;
}

.video-upload__thumbnail-repick:hover {
  background: rgba(0, 0, 0, 0.8);
}

.video-upload__caption {
  width: 100%;
  box-sizing: border-box;
  border: 1px solid var(--border, #d1d5db);
  border-radius: var(--radius-md, 0.375rem);
  padding: 0.5rem 0.625rem;
  font-size: var(--text-sm, 0.875rem);
  color: var(--text-primary);
  background: var(--surface);
  transition: border-color 0.15s ease;
  font-family: inherit;
}

.video-upload__caption:focus {
  outline: none;
  border-color: var(--brand, #0e6c55);
}

.video-upload__caption:disabled {
  background: var(--neutral-50, #f9fafb);
  color: var(--text-secondary, #6b7280);
  cursor: not-allowed;
}

.video-upload__error {
  color: var(--danger, #dc2626);
  font-size: var(--text-sm, 0.875rem);
  margin: 0;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/VideoUpload.css
git commit -m "feat: add VideoUpload CSS for drop zone and thumbnail banner"
```

---

### Task 2: Write failing tests for VideoUpload

**Files:**
- Create: `frontend/src/components/VideoUpload.test.tsx`

The tests cover all three modal states. They mock `document.createElement` for `video`/`canvas` tags so thumbnail extraction runs synchronously in JSDOM without needing real media APIs. The original `document.createElement` is preserved for all other tags so React renders normally.

- [ ] **Step 1: Create the test file**

```tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import VideoUpload from './VideoUpload';
import { animalsApi } from '../api/client';

vi.mock('../api/client', () => ({
  animalsApi: {
    uploadVideo: vi.fn(),
  },
}));

const originalCreateElement = document.createElement.bind(document);

function makeMockVideo() {
  const el = originalCreateElement('div') as any;
  el.videoWidth = 640;
  el.videoHeight = 480;
  el.duration = 5;
  el.muted = false;
  el.playsInline = false;
  el.preload = '';
  el.src = '';
  el.onloadedmetadata = null;
  el.onseeked = null;
  el.onerror = null;
  el.load = function () {
    Promise.resolve().then(() => el.onloadedmetadata?.());
  };
  Object.defineProperty(el, 'currentTime', {
    set(_val: number) {
      Promise.resolve().then(() => el.onseeked?.());
    },
    get() { return 0; },
    configurable: true,
  });
  return el;
}

function makeMockCanvas() {
  const el = originalCreateElement('div') as any;
  el.width = 0;
  el.height = 0;
  el.getContext = () => ({ drawImage: vi.fn() });
  el.toBlob = (cb: (blob: Blob) => void) => {
    cb(new Blob(['fake-jpeg'], { type: 'image/jpeg' }));
  };
  return el;
}

const defaultProps = {
  groupId: 1,
  animalId: 2,
  onSuccess: vi.fn(),
  onCancel: vi.fn(),
};

beforeEach(() => {
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock-thumbnail');
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
    if (tag === 'video') return makeMockVideo();
    if (tag === 'canvas') return makeMockCanvas();
    return originalCreateElement(tag);
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('VideoUpload', () => {
  describe('initial state (no file selected)', () => {
    it('renders the drop zone', () => {
      render(<VideoUpload {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Choose a video file' })).toBeInTheDocument();
    });

    it('does not render the thumbnail banner', () => {
      render(<VideoUpload {...defaultProps} />);
      expect(screen.queryByRole('button', { name: 'Change video' })).not.toBeInTheDocument();
    });

    it('disables the caption field', () => {
      render(<VideoUpload {...defaultProps} />);
      expect(screen.getByPlaceholderText('Caption (optional)')).toBeDisabled();
    });

    it('disables the upload button', () => {
      render(<VideoUpload {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Upload Video' })).toBeDisabled();
    });

    it('enables the cancel button', () => {
      render(<VideoUpload {...defaultProps} />);
      expect(screen.getByRole('button', { name: 'Cancel' })).not.toBeDisabled();
    });
  });

  describe('file selected state', () => {
    async function selectFile(filename = 'clip.mp4', type = 'video/mp4') {
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      const file = new File(['video-data'], filename, { type });
      await userEvent.upload(input, file);
    }

    it('shows the thumbnail banner after a valid file is selected', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      expect(screen.getByRole('button', { name: 'Change video' })).toBeInTheDocument();
    });

    it('hides the drop zone after a file is selected', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      expect(screen.queryByRole('button', { name: 'Choose a video file' })).not.toBeInTheDocument();
    });

    it('displays the filename in the thumbnail meta', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile('my-clip.mp4');
      expect(screen.getByText(/my-clip\.mp4/)).toBeInTheDocument();
    });

    it('enables the caption field', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      expect(screen.getByPlaceholderText('Caption (optional)')).not.toBeDisabled();
    });

    it('enables the upload button', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      expect(screen.getByRole('button', { name: 'Upload Video' })).not.toBeDisabled();
    });

    it('sets the thumbnail background image once the extraction promise resolves', async () => {
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      await waitFor(() => {
        const bg = document.querySelector('.video-upload__thumbnail-bg') as HTMLElement;
        expect(bg.style.backgroundImage).toBe('url(blob:mock-thumbnail)');
      });
    });

    it('clears the selection and returns to drop zone when ✕ is clicked', async () => {
      const user = userEvent.setup();
      render(<VideoUpload {...defaultProps} />);
      await selectFile();
      await user.click(screen.getByRole('button', { name: 'Remove selected video' }));
      expect(screen.getByRole('button', { name: 'Choose a video file' })).toBeInTheDocument();
    });

    it('clears an existing error when a new valid file is selected', async () => {
      render(<VideoUpload {...defaultProps} />);
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      // Select invalid file to trigger an error
      await userEvent.upload(input, new File(['data'], 'clip.avi', { type: 'video/avi' }));
      expect(screen.getByRole('alert')).toBeInTheDocument();
      // Select a valid file — error should clear
      await userEvent.upload(input, new File(['video'], 'clip.mp4', { type: 'video/mp4' }));
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });
  });

  describe('validation errors', () => {
    it('shows error for unsupported file type', async () => {
      render(<VideoUpload {...defaultProps} />);
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      await userEvent.upload(input, new File(['data'], 'clip.avi', { type: 'video/avi' }));
      expect(screen.getByRole('alert')).toHaveTextContent('Only MP4 and MOV videos are supported.');
    });

    it('shows error for oversized file', async () => {
      render(<VideoUpload {...defaultProps} />);
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      const bigFile = new File(['x'.repeat(1)], 'huge.mp4', { type: 'video/mp4' });
      Object.defineProperty(bigFile, 'size', { value: 201 * 1024 * 1024 });
      await userEvent.upload(input, bigFile);
      expect(screen.getByRole('alert')).toHaveTextContent('This video is too large.');
    });
  });

  describe('uploading state', () => {
    it('disables both buttons while uploading', async () => {
      vi.mocked(animalsApi.uploadVideo).mockReturnValue(new Promise(() => {}) as any);
      const user = userEvent.setup();
      render(<VideoUpload {...defaultProps} />);

      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      await userEvent.upload(input, new File(['video'], 'clip.mp4', { type: 'video/mp4' }));

      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Upload Video' })).not.toBeDisabled()
      );

      await user.click(screen.getByRole('button', { name: 'Upload Video' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
        expect(document.querySelector('.button--loading')).toBeInTheDocument();
      });
    });
  });
});
```

- [ ] **Step 2: Run the tests — confirm they fail**

```bash
cd frontend && npx vitest run src/components/VideoUpload.test.tsx
```

Expected: All tests fail with "Cannot find module" or component rendering errors (the component has not been updated yet).

---

### Task 3: Rewrite VideoUpload.tsx

**Files:**
- Modify: `frontend/src/components/VideoUpload.tsx`

Replace the entire file. Key changes: hidden file input with a `ref`, drop zone div, thumbnail banner with repick/clear button, `thumbnailUrl` state derived from the extraction promise, `Button` component for actions, `modal__actions` footer, no `<h3>` heading (Modal provides it).

- [ ] **Step 1: Replace VideoUpload.tsx**

```tsx
import React, { useState, useEffect, useRef } from 'react';
import { animalsApi } from '../api/client';
import Button from './Button';
import './VideoUpload.css';

const MAX_VIDEO_SIZE = 200 * 1024 * 1024;
const ALLOWED_TYPES = ['video/mp4', 'video/quicktime'];

interface VideoUploadProps {
  groupId: number;
  animalId: number;
  onSuccess: () => void;
  onCancel: () => void;
  preselectedFile?: File;
}

const VideoUpload: React.FC<VideoUploadProps> = ({ groupId, animalId, onSuccess, onCancel, preselectedFile }) => {
  const [videoFile, setVideoFile] = useState<File | null>(preselectedFile ?? null);
  const [caption, setCaption] = useState('');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(null);
  const thumbnailPromiseRef = useRef<Promise<{ blob: Blob; duration: number }> | null>(null);
  const thumbnailObjectUrlRef = useRef<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    return () => {
      if (thumbnailObjectUrlRef.current) {
        URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      }
    };
  }, []);

  const applyThumbnailPromise = (promise: Promise<{ blob: Blob; duration: number }>) => {
    promise
      .then(({ blob }) => {
        const url = URL.createObjectURL(blob);
        thumbnailObjectUrlRef.current = url;
        setThumbnailUrl(url);
      })
      .catch(() => {});
  };

  useEffect(() => {
    if (!preselectedFile) return;
    if (!ALLOWED_TYPES.includes(preselectedFile.type)) {
      setError('Only MP4 and MOV videos are supported.');
      setVideoFile(null);
    } else if (preselectedFile.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      setVideoFile(null);
    } else {
      const promise = extractThumbnail(preselectedFile);
      thumbnailPromiseRef.current = promise;
      applyThumbnailPromise(promise);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectedFile]);

  const extractThumbnail = (file: File): Promise<{ blob: Blob; duration: number }> =>
    new Promise((resolve, reject) => {
      const video = document.createElement('video');
      video.preload = 'auto';
      video.muted = true;
      video.playsInline = true;
      const objectUrl = URL.createObjectURL(file);

      const capture = () => {
        if (video.videoWidth === 0 || video.videoHeight === 0) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Video dimensions unavailable'));
          return;
        }
        const MAX_THUMB_WIDTH = 640;
        const scale = Math.min(1, MAX_THUMB_WIDTH / video.videoWidth);
        const canvas = document.createElement('canvas');
        canvas.width = Math.round(video.videoWidth * scale);
        canvas.height = Math.round(video.videoHeight * scale);
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Canvas context unavailable'));
          return;
        }
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
        canvas.toBlob(
          (blob) => {
            URL.revokeObjectURL(objectUrl);
            if (!blob) {
              reject(new Error('Failed to generate thumbnail'));
              return;
            }
            resolve({ blob, duration: Math.round(video.duration) });
          },
          'image/jpeg',
          0.85,
        );
      };

      video.onloadedmetadata = () => {
        let settled = false;
        const done = () => {
          if (!settled) {
            settled = true;
            capture();
          }
        };

        video.onseeked = () => {
          video.onseeked = null;
          done();
        };

        video.currentTime = isFinite(video.duration) ? Math.min(0.5, video.duration) : 0;

        setTimeout(() => {
          video.onseeked = null;
          if (!settled) {
            settled = true;
            URL.revokeObjectURL(objectUrl);
            reject(new Error('Video seek timed out'));
          }
        }, 5000);
      };

      video.onerror = () => {
        video.onseeked = null;
        URL.revokeObjectURL(objectUrl);
        reject(new Error('Failed to load video'));
      };

      video.src = objectUrl;
      video.load();
    });

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError('Only MP4 and MOV videos are supported.');
      return;
    }
    if (file.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      return;
    }
    setError(null);
    setVideoFile(file);
    if (thumbnailObjectUrlRef.current) {
      URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      thumbnailObjectUrlRef.current = null;
    }
    setThumbnailUrl(null);
    const promise = extractThumbnail(file);
    thumbnailPromiseRef.current = promise;
    applyThumbnailPromise(promise);
  };

  const clearSelection = () => {
    setVideoFile(null);
    setError(null);
    setThumbnailUrl(null);
    thumbnailPromiseRef.current = null;
    if (thumbnailObjectUrlRef.current) {
      URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      thumbnailObjectUrlRef.current = null;
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleUpload = async () => {
    if (!videoFile) return;
    setUploading(true);
    setError(null);

    let thumbnailBlob: Blob;
    let duration: number;
    try {
      ({ blob: thumbnailBlob, duration } = await (thumbnailPromiseRef.current ?? extractThumbnail(videoFile)));
    } catch {
      setError("Couldn't generate a preview for this video. Please try a different file.");
      setUploading(false);
      return;
    }

    try {
      await animalsApi.uploadVideo(groupId, animalId, videoFile, thumbnailBlob, caption, duration);
      onSuccess();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || 'Upload failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const fileSizeMB = videoFile ? (videoFile.size / 1024 / 1024).toFixed(1) : '';

  return (
    <div className="video-upload">
      <input
        ref={fileInputRef}
        type="file"
        accept="video/mp4,video/quicktime,.mp4,.mov"
        onChange={handleFileSelect}
        disabled={uploading}
        style={{ display: 'none' }}
        aria-hidden="true"
      />

      {!videoFile ? (
        <div
          className="video-upload__dropzone"
          role="button"
          tabIndex={0}
          aria-label="Choose a video file"
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click();
          }}
        >
          <div className="video-upload__dropzone-icon" aria-hidden="true">🎬</div>
          <div className="video-upload__dropzone-label">Tap to choose a video</div>
          <div className="video-upload__dropzone-hint">MP4 or MOV · up to 200 MB</div>
        </div>
      ) : (
        <div
          className="video-upload__thumbnail"
          role="button"
          tabIndex={0}
          aria-label="Change video"
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click();
          }}
        >
          <div
            className="video-upload__thumbnail-bg"
            style={thumbnailUrl ? { backgroundImage: `url(${thumbnailUrl})` } : undefined}
          />
          <div className="video-upload__thumbnail-overlay" />
          <div className="video-upload__thumbnail-play">
            <div className="video-upload__thumbnail-play-circle">▶</div>
          </div>
          <div className="video-upload__thumbnail-meta">
            {videoFile.name} · {fileSizeMB} MB
          </div>
          <button
            type="button"
            className="video-upload__thumbnail-repick"
            aria-label="Remove selected video"
            onClick={(e) => {
              e.stopPropagation();
              clearSelection();
            }}
          >
            ✕
          </button>
        </div>
      )}

      <input
        type="text"
        className="video-upload__caption"
        placeholder="Caption (optional)"
        value={caption}
        onChange={(e) => setCaption(e.target.value)}
        disabled={!videoFile || uploading}
      />

      {error && (
        <p className="video-upload__error" role="alert">
          {error}
        </p>
      )}

      <div className="modal__actions">
        <Button variant="secondary" onClick={onCancel} disabled={uploading}>
          Cancel
        </Button>
        <Button onClick={handleUpload} disabled={!videoFile} loading={uploading}>
          {uploading ? 'Uploading…' : 'Upload Video'}
        </Button>
      </div>
    </div>
  );
};

export default VideoUpload;
```

- [ ] **Step 2: Run the tests — confirm they pass**

```bash
cd frontend && npx vitest run src/components/VideoUpload.test.tsx
```

Expected: All tests pass.

- [ ] **Step 3: Type-check**

```bash
cd frontend && npm run type-check
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/VideoUpload.tsx frontend/src/components/VideoUpload.test.tsx
git commit -m "feat: replace native file input with drop zone and thumbnail banner in VideoUpload"
```

---

### Task 4: Swap modal wrapper in PhotoGallery.tsx

**Files:**
- Modify: `frontend/src/pages/PhotoGallery.tsx`

The video upload modal currently uses hand-rolled `modal-overlay`/`modal-content` divs. Replace with the `Modal` component, which adds Escape key, backdrop click to close, focus trap, and scroll lock for free. No `onCancel` behaviour changes — both Modal's `onClose` and VideoUpload's Cancel button call the same cleanup.

- [ ] **Step 1: Add Modal import to PhotoGallery.tsx**

At the top of [frontend/src/pages/PhotoGallery.tsx](frontend/src/pages/PhotoGallery.tsx), find the existing imports block and add:

```tsx
import Modal from '../components/Modal';
```

The existing imports look like:
```tsx
import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { animalsApi } from '../api/client';
import type { Animal, AnimalImage, AnimalVideo } from '../api/client';
import VideoUpload from '../components/VideoUpload';
```

Add `import Modal from '../components/Modal';` after the `VideoUpload` import.

- [ ] **Step 2: Replace the video upload modal wrapper**

Find this block in [frontend/src/pages/PhotoGallery.tsx](frontend/src/pages/PhotoGallery.tsx):

```tsx
        {/* Video upload modal */}
        {showVideoUpload && groupId && id && (
          <div className="modal-overlay">
            <div className="modal-content">
              <VideoUpload
                groupId={Number(groupId)}
                animalId={Number(id)}
                preselectedFile={pendingVideoFile ?? undefined}
                onSuccess={() => {
                  setShowVideoUpload(false);
                  setPendingVideoFile(null);
                  showToast('Video uploaded successfully', 'success');
                  loadData(Number(groupId), Number(id));
                }}
                onCancel={() => { setShowVideoUpload(false); setPendingVideoFile(null); }}
              />
            </div>
          </div>
        )}
```

Replace it with:

```tsx
        {/* Video upload modal */}
        {groupId && id && (
          <Modal
            isOpen={showVideoUpload}
            onClose={() => { setShowVideoUpload(false); setPendingVideoFile(null); }}
            title="Upload Video"
            size="small"
          >
            <VideoUpload
              groupId={Number(groupId)}
              animalId={Number(id)}
              preselectedFile={pendingVideoFile ?? undefined}
              onSuccess={() => {
                setShowVideoUpload(false);
                setPendingVideoFile(null);
                showToast('Video uploaded successfully', 'success');
                loadData(Number(groupId), Number(id));
              }}
              onCancel={() => { setShowVideoUpload(false); setPendingVideoFile(null); }}
            />
          </Modal>
        )}
```

- [ ] **Step 3: Type-check**

```bash
cd frontend && npm run type-check
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/PhotoGallery.tsx
git commit -m "feat: use Modal component for video upload modal in PhotoGallery"
```

---

### Task 5: Full test run and verification

- [ ] **Step 1: Run all frontend tests**

```bash
cd frontend && npx vitest run
```

Expected: All tests pass. New `VideoUpload.test.tsx` tests should be visible in the output.

- [ ] **Step 2: Type-check the full project**

```bash
cd frontend && npm run type-check
```

Expected: No errors.

- [ ] **Step 3: Final commit (if any stray changes)**

If there are no unstaged changes this step is a no-op. Only commit if there is something to commit.

```bash
git status
```
