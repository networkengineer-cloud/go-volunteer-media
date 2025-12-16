import React, { useEffect, useRef, useState } from 'react';
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

  useEffect(() => {
    let isMounted = true;

    const applyFitToWidth = () => {
      const container = containerRef.current;
      if (!container) return;

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

      // Small padding budget to avoid edge clipping.
      const paddingBudget = 8;
      const scale = Math.min(1, Math.max(0.1, (containerWidth - paddingBudget) / pageWidth));

      wrapper.style.transformOrigin = 'top left';
      wrapper.style.transform = `scale(${scale})`;
    };

    const renderDocument = async () => {
      if (!containerRef.current) return;

      try {
        setLoading(true);
        setError(null);

        // Clear any previous content
        containerRef.current.innerHTML = '';

        // Render the DOCX using docx-preview library
        // Pass partial options - all missing properties use defaults
        await renderAsync(blob, containerRef.current, undefined, {
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

        if (!isMounted) return;

        // Post-process: add security attributes to external links
        const links = containerRef.current.querySelectorAll('a[href]');
        links.forEach((link) => {
          const href = link.getAttribute('href');
          if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
          }
        });

        // Fit the rendered page(s) to the current viewport width.
        // Use rAF to ensure layout is complete before measurement.
        requestAnimationFrame(() => {
          applyFitToWidth();
        });

        setLoading(false);
      } catch (err) {
        console.error('Failed to render DOCX:', err);
        if (isMounted) {
          setError('Failed to display DOCX document. The file may be corrupted or in an unsupported format.');
          setLoading(false);
        }
      }
    };

    renderDocument();

    const resizeObserver = typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(() => {
          applyFitToWidth();
        })
      : null;

    if (containerRef.current && resizeObserver) {
      resizeObserver.observe(containerRef.current);
    }

    return () => {
      isMounted = false;
      resizeObserver?.disconnect();
      if (containerRef.current) {
        containerRef.current.innerHTML = '';
      }
    };
  }, [blob]);

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
