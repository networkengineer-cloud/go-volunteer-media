import React, { useEffect, useRef, useState } from 'react';
import * as pdfjsLib from 'pdfjs-dist';
import './ProtocolPdfViewer.css';

// Configure PDF.js worker to use local bundled worker (avoids CDN dependency)
pdfjsLib.GlobalWorkerOptions.workerSrc = '/pdf.worker.min.mjs';

interface ProtocolPdfViewerProps {
  blob: Blob;
  fileName?: string;
}

const ProtocolPdfViewer: React.FC<ProtocolPdfViewerProps> = ({ blob, fileName }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [pdf, setPdf] = useState<pdfjsLib.PDFDocumentProxy | null>(null);
  const [totalPages, setTotalPages] = useState(0);
  const [scale, setScale] = useState(1.0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const canvasRefs = useRef<Map<number, HTMLCanvasElement>>(new Map());

  useEffect(() => {
    let isMounted = true;
    let pdfDocument: pdfjsLib.PDFDocumentProxy | null = null;
    
    const loadPdf = async () => {
      try {
        setLoading(true);
        setError(null);

        const arrayBuffer = await blob.arrayBuffer();
        const loadingTask = pdfjsLib.getDocument({ data: arrayBuffer });
        pdfDocument = await loadingTask.promise;

        if (!isMounted) {
          // Clean up if component unmounted during load
          pdfDocument.destroy();
          return;
        }

        setPdf(pdfDocument);
        setTotalPages(pdfDocument.numPages);
        setLoading(false);
      } catch (err) {
        console.error('Failed to load PDF:', err);
        if (isMounted) {
          setError('Failed to load PDF document');
          setLoading(false);
        }
      }
    };

    loadPdf();

    return () => {
      isMounted = false;
      // Properly clean up the pdfDocument from this effect's scope
      if (pdfDocument) {
        pdfDocument.destroy();
      }
    };
  }, [blob]);

  useEffect(() => {
    if (!pdf || !containerRef.current) return;

    const renderPage = async (pageNum: number) => {
      try {
        const page = await pdf.getPage(pageNum);
        const viewport = page.getViewport({ scale });

        // Get or create canvas for this page
        let canvas = canvasRefs.current.get(pageNum);
        if (!canvas) {
          canvas = document.createElement('canvas');
          canvas.className = 'pdf-page-canvas';
          canvasRefs.current.set(pageNum, canvas);
        }

        const context = canvas.getContext('2d');
        if (!context) return;

        canvas.height = viewport.height;
        canvas.width = viewport.width;

        const renderContext = {
          canvasContext: context,
          viewport: viewport,
        };

        await page.render(renderContext).promise;

        // Append canvas to container if not already there
        if (containerRef.current && !containerRef.current.contains(canvas)) {
          const pageWrapper = document.createElement('div');
          pageWrapper.className = 'pdf-page-wrapper';
          pageWrapper.dataset.pageNum = String(pageNum);
          pageWrapper.appendChild(canvas);
          containerRef.current.appendChild(pageWrapper);
        }
      } catch (err) {
        console.error(`Failed to render page ${pageNum}:`, err);
      }
    };

    // Clear container
    if (containerRef.current) {
      containerRef.current.innerHTML = '';
      canvasRefs.current.clear();
    }

    // Render all pages
    const renderAllPages = async () => {
      for (let i = 1; i <= totalPages; i++) {
        await renderPage(i);
      }
    };

    renderAllPages();
  }, [pdf, scale, totalPages]);

  const handleZoomIn = () => {
    setScale(prev => Math.min(prev + 0.25, 3.0));
  };

  const handleZoomOut = () => {
    setScale(prev => Math.max(prev - 0.25, 0.5));
  };

  const handleResetZoom = () => {
    setScale(1.0);
  };

  if (loading) {
    return (
      <div className="protocol-pdf-loading">
        <span className="loading-spinner" aria-label="Loading PDF document"></span>
        <p>Loading PDF...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="protocol-pdf-error" role="alert">
        <p>{error}</p>
      </div>
    );
  }

  return (
    <div className="protocol-pdf-viewer">
      <div className="protocol-pdf-controls">
        <div className="pdf-zoom-controls">
          <button
            onClick={handleZoomOut}
            disabled={scale <= 0.5}
            aria-label="Zoom out"
            className="btn-zoom"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
              <line x1="8" y1="11" x2="14" y2="11" />
            </svg>
          </button>
          <span className="zoom-level">{Math.round(scale * 100)}%</span>
          <button
            onClick={handleZoomIn}
            disabled={scale >= 3.0}
            aria-label="Zoom in"
            className="btn-zoom"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
              <line x1="11" y1="8" x2="11" y2="14" />
              <line x1="8" y1="11" x2="14" y2="11" />
            </svg>
          </button>
          {scale !== 1.0 && (
            <button
              onClick={handleResetZoom}
              aria-label="Reset zoom"
              className="btn-reset-zoom"
            >
              Reset
            </button>
          )}
        </div>
        <div className="pdf-page-info">
          Page {currentPage} of {totalPages}
        </div>
      </div>
      <div className="protocol-pdf-container" ref={containerRef} />
    </div>
  );
};

export default ProtocolPdfViewer;
