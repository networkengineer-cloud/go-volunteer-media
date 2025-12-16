import React, { useEffect, useMemo, useRef, useState } from 'react';
import { renderAsync } from 'docx-preview';
import './ProtocolDocxViewer.css';

interface ProtocolDocxViewerProps {
  blob: Blob;
  fileName?: string;
}

const ProtocolDocxViewer: React.FC<ProtocolDocxViewerProps> = ({ blob, fileName }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Prevent edge clipping on narrow viewports (e.g. 4px padding on each side).
  const FIT_TO_WIDTH_PADDING_BUDGET_PX = 8;

  // Ensure a fresh container node per blob render.
  // `docx-preview` imperatively renders into a provided element; by remounting the
  // container we avoid manual DOM clearing (`innerHTML = ''`) and any accidental
  // cross-render state.
  const containerKey = useMemo(() => {
    const nonce = Math.random().toString(36).slice(2);
    return `docx-${Date.now()}-${nonce}`;
  }, [blob]);

  useEffect(() => {
    let cancelled = false;

    const container = containerRef.current;
    if (!container) return;

    let resizeObserver: ResizeObserver | null = null;
    let rafId: number | null = null;

    const applyFitToWidth = () => {
      try {
        const wrapper = container.querySelector('.docx-wrapper') as HTMLElement | null;
        const firstPage = container.querySelector('.docx-wrapper > section.docx') as HTMLElement | null;
        if (!wrapper || !firstPage) return;

        // Reset to measure accurately.
        wrapper.style.transform = '';
        wrapper.style.transformOrigin = '';

        // Ensure we are not horizontally scrolled.
        container.scrollLeft = 0;

        const containerWidth = container.clientWidth;
        const pageWidth = firstPage.getBoundingClientRect().width;
        if (!containerWidth || !pageWidth) return;

        const scale = Math.min(
          1,
          Math.max(0.1, (containerWidth - FIT_TO_WIDTH_PADDING_BUDGET_PX) / pageWidth)
        );

        wrapper.style.transformOrigin = 'top left';
        wrapper.style.transform = `scale(${scale})`;
      } catch (err) {
        // Best-effort: sizing issues should not fail rendering.
        console.warn('DOCX fit-to-width failed:', err);
      }
    };

    const renderDocument = async () => {
      try {
        setLoading(true);
        setError(null);

        // Render the DOCX using docx-preview library
        // Pass partial options - all missing properties use defaults
        await renderAsync(blob, container, undefined, {
          className: 'docx-preview-wrapper',
          inWrapper: true,
          ignoreWidth: true, // Allow width to be responsive
          ignoreHeight: false,
          ignoreFonts: false,
          breakPages: true,
          ignoreLastRenderedPageBreak: true,
          experimental: false,
          trimXmlDeclaration: true,
          debug: false,
          useBase64URL: true,
        });

        if (cancelled) return;

        // Post-process: add security attributes to external links
        const links = container.querySelectorAll('a[href]');
        links.forEach((link) => {
          const href = link.getAttribute('href');
          if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
          }
        });

        // Fit the rendered page(s) to the current viewport width.
        // Use rAF to ensure layout is complete before measurement.
        rafId = requestAnimationFrame(() => {
          applyFitToWidth();
        });

        setLoading(false);
      } catch (err) {
        console.error('Failed to render DOCX:', err);
        if (!cancelled) {
          setError('Failed to display DOCX document. The file may be corrupted or in an unsupported format.');
          setLoading(false);
        }
      }
    };

    renderDocument();

    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        try {
          applyFitToWidth();
        } catch (err) {
          console.warn('DOCX resize failed:', err);
        }
      });
      resizeObserver.observe(container);
    }

    return () => {
      cancelled = true;
      if (rafId != null) cancelAnimationFrame(rafId);
      resizeObserver?.disconnect();
    };
    // `containerKey` is derived from `blob` and forces a fresh container element.
    // Including it makes the dependency relationship explicit and avoids subtle
    // mismatch between the rendered container node and the effect lifecycle.
  }, [blob, containerKey]);

  if (error) {
    return (
      <div className="protocol-docx-error" role="alert">
        <p>{error}</p>
      </div>
    );
  }

  return (
    <div className="protocol-docx-viewer">
      {loading && (
        <div className="protocol-docx-loading">
          <span className="loading-spinner" aria-label="Loading DOCX document"></span>
          <p>Converting document...</p>
        </div>
      )}
      <div 
        key={containerKey}
        ref={containerRef} 
        className="protocol-docx-content"
        role="document"
        aria-label={fileName || 'Protocol document content'}
        style={{ display: loading ? 'none' : 'block' }}
      />
    </div>
  );
};

export default ProtocolDocxViewer;
