import { render } from '@testing-library/react';
import type { RenderOptions } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import { ToastProvider } from '../contexts/ToastContext';
import type { ReactElement } from 'react';

/**
 * Custom render function that wraps components with necessary providers
 */
export const renderWithProviders = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          {children}
        </ToastProvider>
      </AuthProvider>
    </BrowserRouter>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};

/**
 * Custom render for testing components without routing
 */
export const renderWithAuth = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>
      <ToastProvider>
        {children}
      </ToastProvider>
    </AuthProvider>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};

// Re-export everything from React Testing Library
// eslint-disable-next-line react-refresh/only-export-components
export * from '@testing-library/react';
export { default as userEvent } from '@testing-library/user-event';
