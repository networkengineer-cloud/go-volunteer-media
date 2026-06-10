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
