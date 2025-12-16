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
          ignoreWidth: false,
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

    return () => {
      isMounted = false;
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
