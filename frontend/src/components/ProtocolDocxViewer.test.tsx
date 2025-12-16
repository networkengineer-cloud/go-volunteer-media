import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import ProtocolDocxViewer from './ProtocolDocxViewer';

const renderAsyncMock = vi.fn();

vi.mock('docx-preview', () => ({
  renderAsync: (...args: unknown[]) => renderAsyncMock(...args),
}));

describe('ProtocolDocxViewer', () => {
  const originalResizeObserver = globalThis.ResizeObserver;
  const originalRAF = globalThis.requestAnimationFrame;
  const originalCAF = globalThis.cancelAnimationFrame;

  beforeEach(() => {
    renderAsyncMock.mockReset();

    globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    };
    globalThis.cancelAnimationFrame = vi.fn();
  });

  afterEach(() => {
    globalThis.ResizeObserver = originalResizeObserver;
    globalThis.requestAnimationFrame = originalRAF;
    globalThis.cancelAnimationFrame = originalCAF;
  });

  it('shows loading state while rendering', () => {
    renderAsyncMock.mockImplementation(() => new Promise(() => {}));

    render(<ProtocolDocxViewer blob={new Blob(['x'], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' })} />);

    expect(screen.getByText('Converting document...')).toBeInTheDocument();
    expect(screen.queryByRole('document')).not.toBeInTheDocument();
  });

  it('renders and hardens external links', async () => {
    renderAsyncMock.mockImplementation(async (_blob: Blob, container: HTMLElement) => {
      const wrapper = document.createElement('div');
      wrapper.className = 'docx-wrapper';

      const page = document.createElement('section');
      page.className = 'docx';
      // Ensure `getBoundingClientRect().width` is non-zero for fit-to-width.
      page.getBoundingClientRect = () => ({
        width: 1000,
        height: 0,
        top: 0,
        left: 0,
        bottom: 0,
        right: 0,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as DOMRect);

      const link = document.createElement('a');
      link.href = 'https://example.com';
      link.textContent = 'Example';

      page.appendChild(link);
      wrapper.appendChild(page);
      container.appendChild(wrapper);

      Object.defineProperty(container, 'clientWidth', { value: 500, configurable: true });
    });

    render(
      <ProtocolDocxViewer
        blob={new Blob(['x'], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' })}
        fileName="protocol.docx"
      />
    );

    await waitFor(() => {
      expect(screen.queryByText('Converting document...')).not.toBeInTheDocument();
    });

    const documentEl = screen.getByRole('document');
    expect(documentEl).toHaveAttribute('aria-label', 'protocol.docx');

    const anchor = documentEl.querySelector('a[href]');
    expect(anchor).toBeTruthy();
    expect(anchor).toHaveAttribute('target', '_blank');
    expect(anchor).toHaveAttribute('rel', 'noopener noreferrer');

    const wrapper = documentEl.querySelector('.docx-wrapper') as HTMLElement | null;
    expect(wrapper).toBeTruthy();
    expect(wrapper?.style.transform).toMatch(/^scale\(/);
  });

  it('shows an error message when rendering fails', async () => {
    renderAsyncMock.mockRejectedValueOnce(new Error('boom'));

    render(<ProtocolDocxViewer blob={new Blob(['x'], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' })} />);

    expect(await screen.findByRole('alert')).toHaveTextContent('Failed to display DOCX document');
  });

  it('disconnects ResizeObserver on unmount when available', async () => {
    const disconnect = vi.fn();
    const observe = vi.fn();

    globalThis.ResizeObserver = class MockResizeObserver {
      constructor(_cb: ResizeObserverCallback) {}
      observe = observe;
      disconnect = disconnect;
      unobserve = vi.fn();
    } as unknown as typeof ResizeObserver;

    renderAsyncMock.mockResolvedValueOnce(undefined);

    const { unmount } = render(
      <ProtocolDocxViewer blob={new Blob(['x'], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' })} />
    );

    await waitFor(() => {
      expect(renderAsyncMock).toHaveBeenCalled();
    });

    unmount();
    expect(disconnect).toHaveBeenCalledTimes(1);
  });
});
