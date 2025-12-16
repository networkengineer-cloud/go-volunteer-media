import React, { useEffect, useRef, useState } from 'react';
import ProtocolPdfViewer from './ProtocolPdfViewer';
import ProtocolDocxViewer from './ProtocolDocxViewer';
import { animalsApi } from '../api/client';
import { useToast } from '../hooks/useToast';
import './ProtocolViewer.css';

interface ProtocolViewerProps {
  documentUrl: string;
  documentName?: string;
  onClose: () => void;
}

type DocumentKind = 'pdf' | 'docx' | 'unknown';

const ProtocolViewer: React.FC<ProtocolViewerProps> = ({ 
  documentUrl, 
  documentName,
  onClose 
}) => {
  const overlayRef = useRef<HTMLDivElement>(null);
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [documentBlob, setDocumentBlob] = useState<Blob | null>(null);
  const [documentKind, setDocumentKind] = useState<DocumentKind>('unknown');

  useEffect(() => {
    overlayRef.current?.focus();
  }, []);

  useEffect(() => {
    const loadDocument = async () => {
      try {
        setLoading(true);
        setError(null);

        // Extract UUID from the document URL (format: /api/documents/UUID)
        const uuid = documentUrl.split('/').pop();
        if (!uuid) {
          setError('Invalid document URL');
          setLoading(false);
          return;
        }

        // Fetch document with authorization header via API client
        const response = await animalsApi.getProtocolDocument(uuid);

        if (!(response.data instanceof Blob)) {
          throw new Error('Unexpected response type');
        }

        const responseBlob = response.data;
        const rawHeaderContentType = response.headers?.['content-type'] || response.headers?.['Content-Type'];
        const headerContentType = typeof rawHeaderContentType === 'string' ? rawHeaderContentType : undefined;
        const filename = (documentName || '').toLowerCase();

        const normalizeContentType = (value?: string) => {
          if (!value) return undefined;
          const normalized = value.split(';')[0]?.trim().toLowerCase();
          return normalized || undefined;
        };

        const headerType = normalizeContentType(headerContentType);
        const blobType = normalizeContentType(responseBlob.type);

        const kindFromFilename: DocumentKind = filename.endsWith('.pdf')
          ? 'pdf'
          : filename.endsWith('.docx')
            ? 'docx'
            : 'unknown';

        const kindFromType: DocumentKind =
          blobType === 'application/pdf' || headerType === 'application/pdf'
            ? 'pdf'
            : (blobType?.includes('officedocument.wordprocessingml.document') ||
                headerType?.includes('officedocument.wordprocessingml.document'))
              ? 'docx'
              : 'unknown';

        const detectedKind: DocumentKind = kindFromFilename !== 'unknown' ? kindFromFilename : kindFromType;

        if (detectedKind === 'unknown') {
          setError('Unsupported document type. Please upload a PDF or DOCX file.');
          setLoading(false);
          return;
        }

        const effectiveContentType =
          detectedKind === 'docx'
            ? 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
            : 'application/pdf';

        const finalBlob = new Blob([responseBlob], { type: effectiveContentType });
        
        setDocumentBlob(finalBlob);
        setDocumentKind(detectedKind);
        setLoading(false);
      } catch (err) {
        console.error('Failed to load protocol document:', err);
        setError('Failed to load protocol document. Please try again.');
        setLoading(false);
        toast.showError('Failed to load protocol document. Please try again.');
      }
    };

    loadDocument();
  }, [documentUrl, documentName, toast]);

  const handleDownload = async () => {
    if (!documentBlob) return;

    try {
      const url = window.URL.createObjectURL(documentBlob);
      const link = document.createElement('a');
      link.href = url;
      link.download = documentName || 'protocol-document';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Document downloaded successfully!');
    } catch (err) {
      console.error('Failed to download document:', err);
      toast.showError('Failed to download document. Please try again.');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  };

  return (
    <div 
      className="protocol-viewer-overlay"
      ref={overlayRef}
      onClick={onClose}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      aria-labelledby="protocol-viewer-title"
      tabIndex={-1}
    >
      <div 
        className="protocol-viewer-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="protocol-viewer-header">
          <div className="protocol-viewer-header-content">
            <div>
              <h2 id="protocol-viewer-title">📋 Protocol Document</h2>
              {documentName && (
                <p className="protocol-viewer-filename">{documentName}</p>
              )}
            </div>
            <button
              onClick={handleDownload}
              className="btn-download-protocol"
              aria-label="Download protocol document"
              title="Download document"
              disabled={!documentBlob}
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <span className="download-label">Download</span>
            </button>
          </div>
          <button
            onClick={onClose}
            className="protocol-viewer-close"
            aria-label="Close protocol document"
            title="Close (Esc)"
          >
            ✕
          </button>
        </div>

        <div className="protocol-viewer-body">
          {loading ? (
            <div className="protocol-viewer-loading">
              <span className="loading-spinner" aria-label="Loading protocol document"></span>
              <p>Loading protocol document...</p>
            </div>
          ) : error ? (
            <div className="protocol-viewer-error" role="alert">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
              <p>{error}</p>
              <p className="error-hint">Please use the Download button to access the document.</p>
            </div>
          ) : documentBlob && documentKind === 'pdf' ? (
            <ProtocolPdfViewer blob={documentBlob} fileName={documentName} />
          ) : documentBlob && documentKind === 'docx' ? (
            <ProtocolDocxViewer blob={documentBlob} fileName={documentName} />
          ) : null}
        </div>
      </div>
    </div>
  );
};

export default ProtocolViewer;
