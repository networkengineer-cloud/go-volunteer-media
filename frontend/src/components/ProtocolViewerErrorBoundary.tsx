import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ProtocolViewerErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
    };
  }

  static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('Protocol Viewer Error Boundary caught error:', error, errorInfo);
    
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div 
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '2rem',
            textAlign: 'center',
            minHeight: '300px',
            color: 'var(--danger)',
          }}
          role="alert"
        >
          <svg 
            width="64" 
            height="64" 
            viewBox="0 0 24 24" 
            fill="none" 
            stroke="currentColor" 
            strokeWidth="2"
            style={{ marginBottom: '1rem' }}
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <h3 style={{ margin: '0 0 0.5rem 0' }}>
            Unable to Display Protocol Document
          </h3>
          <p style={{ margin: '0 0 1rem 0', color: 'var(--text-secondary)' }}>
            An unexpected error occurred while loading the document viewer.
          </p>
          <p style={{ margin: '0', fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
            Please try using the Download button to access the document directly.
          </p>
          {import.meta.env.MODE === 'development' && this.state.error && (
            <details style={{ marginTop: '1rem', textAlign: 'left', width: '100%' }}>
              <summary style={{ cursor: 'pointer', color: 'var(--text-secondary)' }}>
                Error Details (Development Only)
              </summary>
              <pre style={{
                marginTop: '0.5rem',
                padding: '1rem',
                backgroundColor: 'var(--background-secondary)',
                borderRadius: '4px',
                overflow: 'auto',
                fontSize: '0.75rem',
                textAlign: 'left',
              }}>
                {this.state.error.toString()}
                {'\n\n'}
                {this.state.error.stack}
              </pre>
            </details>
          )}
        </div>
      );
    }

    return this.props.children;
  }
}

export default ProtocolViewerErrorBoundary;
