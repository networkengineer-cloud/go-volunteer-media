import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, renderHook, act, waitFor } from '@testing-library/react';
import { ToastProvider } from './ToastContext';
import { useToast } from '../hooks/useToast';

// Mock the Toast component to avoid testing its internals
vi.mock('../components/Toast', () => ({
  default: ({ message, type, onClose }: { message: string; type: string; onClose: () => void }) => (
    <div data-testid="toast" data-type={type}>
      {message}
      <button onClick={onClose}>Close</button>
    </div>
  ),
}));

describe('ToastContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('ToastProvider', () => {
    it('should render children', () => {
      render(
        <ToastProvider>
          <div data-testid="child">Child content</div>
        </ToastProvider>
      );
      
      expect(screen.getByTestId('child')).toBeInTheDocument();
    });

    it('should render toast container with aria attributes', () => {
      render(<ToastProvider><div>Test</div></ToastProvider>);
      
      const container = document.querySelector('.toast-container');
      expect(container).toBeInTheDocument();
      expect(container).toHaveAttribute('aria-live', 'polite');
      expect(container).toHaveAttribute('aria-atomic', 'false');
    });
  });

  describe('showToast', () => {
    it('should display a toast with default type', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showToast('Test message');
      });

      expect(screen.getByTestId('toast')).toBeInTheDocument();
      expect(screen.getByText('Test message')).toBeInTheDocument();
      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'info');
    });

    it('should display a toast with specified type', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showToast('Success message', 'success');
      });

      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'success');
    });

    it('should display multiple toasts', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showToast('First message');
        result.current.showToast('Second message');
        result.current.showToast('Third message');
      });

      const toasts = screen.getAllByTestId('toast');
      expect(toasts).toHaveLength(3);
      expect(screen.getByText('First message')).toBeInTheDocument();
      expect(screen.getByText('Second message')).toBeInTheDocument();
      expect(screen.getByText('Third message')).toBeInTheDocument();
    });
  });

  describe('showSuccess', () => {
    it('should display a success toast', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showSuccess('Operation successful');
      });

      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'success');
      expect(screen.getByText('Operation successful')).toBeInTheDocument();
    });
  });

  describe('showError', () => {
    it('should display an error toast', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showError('Operation failed');
      });

      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'error');
      expect(screen.getByText('Operation failed')).toBeInTheDocument();
    });
  });

  describe('showWarning', () => {
    it('should display a warning toast', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showWarning('Warning message');
      });

      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'warning');
      expect(screen.getByText('Warning message')).toBeInTheDocument();
    });
  });

  describe('showInfo', () => {
    it('should display an info toast', () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showInfo('Info message');
      });

      expect(screen.getByTestId('toast')).toHaveAttribute('data-type', 'info');
      expect(screen.getByText('Info message')).toBeInTheDocument();
    });
  });

  describe('Toast removal', () => {
    it('should remove toast when close button is clicked', async () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showToast('Removable message');
      });

      expect(screen.getByText('Removable message')).toBeInTheDocument();

      const closeButton = screen.getByRole('button', { name: /close/i });
      act(() => {
        closeButton.click();
      });

      await waitFor(() => {
        expect(screen.queryByText('Removable message')).not.toBeInTheDocument();
      });
    });

    it('should remove correct toast when multiple toasts exist', async () => {
      const { result } = renderHook(() => useToast(), {
        wrapper: ToastProvider,
      });

      act(() => {
        result.current.showToast('First toast');
        result.current.showToast('Second toast');
        result.current.showToast('Third toast');
      });

      const toasts = screen.getAllByTestId('toast');
      expect(toasts).toHaveLength(3);

      // Close the second toast
      const closeButtons = screen.getAllByRole('button', { name: /close/i });
      act(() => {
        closeButtons[1].click();
      });

      await waitFor(() => {
        expect(screen.queryByText('Second toast')).not.toBeInTheDocument();
        expect(screen.getByText('First toast')).toBeInTheDocument();
        expect(screen.getByText('Third toast')).toBeInTheDocument();
      });
    });
  });

  describe('useToast hook error handling', () => {
    it('should throw error when used outside ToastProvider', () => {
      // Suppress console.error for this test
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        renderHook(() => useToast());
      }).toThrow('useToast must be used within ToastProvider');

      consoleError.mockRestore();
    });
  });
});
