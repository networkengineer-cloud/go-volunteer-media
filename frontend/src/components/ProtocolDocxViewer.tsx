import React, { useEffect, useRef, useState } from 'react';
import mammoth from 'mammoth';
import DOMPurify from 'dompurify';
import './ProtocolDocxViewer.css';

interface ProtocolDocxViewerProps {
  blob: Blob;
  fileName?: string;
}

const ProtocolDocxViewer: React.FC<ProtocolDocxViewerProps> = ({ blob, fileName }) => {
  const contentRef = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const convertAndRender = async () => {
      try {
        setLoading(true);
        setError(null);

        // Convert ArrayBuffer to DOCX
        const arrayBuffer = await blob.arrayBuffer();
        
        // Use Mammoth to convert DOCX to HTML
        const result = await mammoth.convertToHtml({ arrayBuffer });
        
        if (!isMounted) return;

        // Configure DOMPurify to allow safe HTML elements
        const config = {
          ALLOWED_TAGS: [
            'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
            'p', 'br', 'hr',
            'ul', 'ol', 'li',
            'strong', 'em', 'u', 's', 'sub', 'sup',
            'a', 'img',
            'table', 'thead', 'tbody', 'tr', 'th', 'td',
            'blockquote', 'pre', 'code',
            'div', 'span'
          ],
          ALLOWED_ATTR: [
            'href', 'src', 'alt', 'title',
            'width', 'height',
            'colspan', 'rowspan',
            'class', 'id'
          ],
          ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i,
        };

        // Sanitize the HTML
        const sanitizedHtml = DOMPurify.sanitize(result.value, config);

        // Process links to add security attributes
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = sanitizedHtml;
        
        // Add rel and target attributes to external links
        const links = tempDiv.querySelectorAll('a[href]');
        links.forEach((link) => {
          const href = link.getAttribute('href');
          if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
          }
        });

        // Ensure images scale properly
        const images = tempDiv.querySelectorAll('img');
        images.forEach((img) => {
          img.style.maxWidth = '100%';
          img.style.height = 'auto';
        });

        if (contentRef.current) {
          contentRef.current.innerHTML = tempDiv.innerHTML;
        }

        setLoading(false);

        // Log any conversion warnings/messages
        if (result.messages && result.messages.length > 0) {
          console.info('DOCX conversion messages:', result.messages);
        }
      } catch (err) {
        console.error('Failed to convert DOCX:', err);
        if (isMounted) {
          setError('Failed to display DOCX document. The file may be corrupted or in an unsupported format.');
          setLoading(false);
        }
      }
    };

    convertAndRender();

    return () => {
      isMounted = false;
      if (contentRef.current) {
        contentRef.current.innerHTML = '';
      }
    };
  }, [blob]);

  if (loading) {
    return (
      <div className="protocol-docx-loading">
        <span className="loading-spinner" aria-label="Loading DOCX document"></span>
        <p>Converting document...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="protocol-docx-error" role="alert">
        <p>{error}</p>
      </div>
    );
  }

  return (
    <div className="protocol-docx-viewer">
      <div 
        ref={contentRef} 
        className="protocol-docx-content"
        role="document"
        aria-label={fileName || 'Protocol document content'}
      />
    </div>
  );
};

export default ProtocolDocxViewer;
