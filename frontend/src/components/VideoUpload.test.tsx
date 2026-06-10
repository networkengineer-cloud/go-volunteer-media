import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import VideoUpload from './VideoUpload';
import { animalsApi } from '../api/client';

vi.mock('../api/client', () => ({
  animalsApi: {
    uploadVideo: vi.fn(),
  },
}));

const originalCreateElement = document.createElement.bind(document);

function makeMockVideo() {
  const el = originalCreateElement('div') as unknown as HTMLVideoElement;
  Object.defineProperty(el, 'videoWidth', { value: 640, writable: true });
  Object.defineProperty(el, 'videoHeight', { value: 480, writable: true });
  Object.defineProperty(el, 'duration', { value: 5, writable: true });
  el.muted = false;
  el.playsInline = false;
  el.preload = '';
  el.src = '';
  (el as any).onloadedmetadata = null;
  (el as any).onseeked = null;
  (el as any).onerror = null;
  (el as any).load = function () {
    Promise.resolve().then(() => (el as any).onloadedmetadata?.());
  };
  Object.defineProperty(el, 'currentTime', {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    set(_val: number) {
      Promise.resolve().then(() => (el as any).onseeked?.());
    },
    get() { return 0; },
    configurable: true,
  });
  return el;
}

function makeMockCanvas() {
  const el = originalCreateElement('div') as unknown as HTMLCanvasElement;
  el.width = 0;
  el.height = 0;
  (el as any).getContext = () => ({ drawImage: vi.fn() });
  el.toBlob = (cb: (blob: Blob | null) => void) => {
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
  defaultProps.onSuccess.mockReset();
  defaultProps.onCancel.mockReset();
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
    async function selectFile(container: HTMLElement, filename = 'clip.mp4', type = 'video/mp4') {
      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      const file = new File(['video-data'], filename, { type });
      await userEvent.upload(input, file);
    }

    it('shows the thumbnail banner after a valid file is selected', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      expect(screen.getByRole('button', { name: 'Change video' })).toBeInTheDocument();
    });

    it('hides the drop zone after a file is selected', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      expect(screen.queryByRole('button', { name: 'Choose a video file' })).not.toBeInTheDocument();
    });

    it('displays the filename in the thumbnail meta', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container, 'my-clip.mp4');
      expect(screen.getByText(/my-clip\.mp4/)).toBeInTheDocument();
    });

    it('enables the caption field', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      expect(screen.getByPlaceholderText('Caption (optional)')).not.toBeDisabled();
    });

    it('enables the upload button', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      expect(screen.getByRole('button', { name: 'Upload Video' })).not.toBeDisabled();
    });

    it('sets the thumbnail background image once the extraction promise resolves', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      await waitFor(() => {
        const bg = container.querySelector('.video-upload__thumbnail-bg') as HTMLElement;
        // JSDOM's CSS parser normalizes URLs to quoted format
        expect(bg.style.backgroundImage).toMatch(/url\(.*blob:mock-thumbnail.*\)/);
      });
    });

    it('clears the selection and returns to drop zone when ✕ is clicked', async () => {
      const user = userEvent.setup();
      const { container } = render(<VideoUpload {...defaultProps} />);
      await selectFile(container);
      await user.click(screen.getByRole('button', { name: 'Remove selected video' }));
      expect(screen.getByRole('button', { name: 'Choose a video file' })).toBeInTheDocument();
    });

    it('clears an existing error when a new valid file is selected', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      // Select invalid file to trigger an error
      const invalidFile = new File(['data'], 'clip.avi', { type: 'video/avi' });
      Object.defineProperty(input, 'files', {
        value: [invalidFile],
        writable: false,
        configurable: true,
      });
      input.dispatchEvent(new Event('change', { bubbles: true }));

      await waitFor(() => {
        expect(screen.getByRole('alert')).toBeInTheDocument();
      });

      // Select a valid file — error should clear
      const validFile = new File(['video'], 'clip.mp4', { type: 'video/mp4' });
      Object.defineProperty(input, 'files', {
        value: [validFile],
        writable: false,
        configurable: true,
      });
      input.dispatchEvent(new Event('change', { bubbles: true }));

      await waitFor(() => {
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
      });
    });
  });

  describe('validation errors', () => {
    it('shows error for unsupported file type', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      // Try bypassing the input's accept attribute by directly triggering the change event
      const file = new File(['data'], 'clip.avi', { type: 'video/avi' });
      // Manually set the files property and trigger change
      Object.defineProperty(input, 'files', {
        value: [file],
        writable: false,
        configurable: true,
      });
      const event = new Event('change', { bubbles: true });
      input.dispatchEvent(event);
      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent('Only MP4 and MOV videos are supported.');
      });
    });

    it('shows error for oversized file', async () => {
      const { container } = render(<VideoUpload {...defaultProps} />);
      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      const bigFile = new File(['x'.repeat(1)], 'huge.mp4', { type: 'video/mp4' });
      Object.defineProperty(bigFile, 'size', { value: 201 * 1024 * 1024 });
      await userEvent.upload(input, bigFile);
      expect(screen.getByRole('alert')).toHaveTextContent('This video is too large.');
    });
  });

  describe('uploading state', () => {
    it('disables both buttons while uploading', async () => {
      vi.mocked(animalsApi.uploadVideo).mockImplementation(() => new Promise(() => {}));
      const user = userEvent.setup();
      const { container } = render(<VideoUpload {...defaultProps} />);

      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      await userEvent.upload(input, new File(['video'], 'clip.mp4', { type: 'video/mp4' }));

      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Upload Video' })).not.toBeDisabled()
      );

      await user.click(screen.getByRole('button', { name: 'Upload Video' }));

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
        expect(container.querySelector('.button--loading')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Uploading…' })).toBeDisabled();
      });
    });
  });
});
