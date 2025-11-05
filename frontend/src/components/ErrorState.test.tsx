import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ErrorState from './ErrorState';

describe('ErrorState', () => {
  describe('Rendering', () => {
    it('should render with required props', () => {
      render(<ErrorState message="An error occurred" />);
      
      expect(screen.getByRole('alert')).toBeInTheDocument();
      expect(screen.getByText('Something went wrong')).toBeInTheDocument(); // default title
      expect(screen.getByText('An error occurred')).toBeInTheDocument();
    });

    it('should render with custom title', () => {
      render(<ErrorState title="Custom Error" message="Custom message" />);
      
      expect(screen.getByText('Custom Error')).toBeInTheDocument();
      expect(screen.getByText('Custom message')).toBeInTheDocument();
    });

    it('should render default icon when no icon provided', () => {
      render(<ErrorState message="Error" />);
      
      const icon = document.querySelector('.error-state-icon svg');
      expect(icon).toBeInTheDocument();
    });

    it('should render custom icon', () => {
      const CustomIcon = () => <div data-testid="custom-icon">Custom</div>;
      
      render(<ErrorState message="Error" icon={<CustomIcon />} />);
      
      expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    });
  });

  describe('Retry action', () => {
    it('should render retry button when onRetry provided', () => {
      const handleRetry = vi.fn();
      
      render(<ErrorState message="Error" onRetry={handleRetry} />);
      
      expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
    });

    it('should call onRetry when retry button clicked', async () => {
      const handleRetry = vi.fn();
      const user = userEvent.setup();
      
      render(<ErrorState message="Error" onRetry={handleRetry} />);
      
      await user.click(screen.getByRole('button', { name: /try again/i }));
      
      expect(handleRetry).toHaveBeenCalledTimes(1);
    });

    it('should render custom retry label', () => {
      render(<ErrorState message="Error" onRetry={vi.fn()} retryLabel="Reload Page" />);
      
      expect(screen.getByRole('button', { name: /reload page/i })).toBeInTheDocument();
    });

    it('should have aria-label on retry button', () => {
      render(<ErrorState message="Error" onRetry={vi.fn()} retryLabel="Retry" />);
      
      const button = screen.getByRole('button', { name: /retry/i });
      expect(button).toHaveAttribute('aria-label', 'Retry');
    });
  });

  describe('Go Back action', () => {
    it('should render go back button when onGoBack provided', () => {
      const handleGoBack = vi.fn();
      
      render(<ErrorState message="Error" onGoBack={handleGoBack} />);
      
      expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument();
    });

    it('should call onGoBack when go back button clicked', async () => {
      const handleGoBack = vi.fn();
      const user = userEvent.setup();
      
      render(<ErrorState message="Error" onGoBack={handleGoBack} />);
      
      await user.click(screen.getByRole('button', { name: /go back/i }));
      
      expect(handleGoBack).toHaveBeenCalledTimes(1);
    });

    it('should render custom go back label', () => {
      render(<ErrorState message="Error" onGoBack={vi.fn()} goBackLabel="Return Home" />);
      
      expect(screen.getByRole('button', { name: /return home/i })).toBeInTheDocument();
    });

    it('should have aria-label on go back button', () => {
      render(<ErrorState message="Error" onGoBack={vi.fn()} goBackLabel="Back" />);
      
      const button = screen.getByRole('button', { name: /back/i });
      expect(button).toHaveAttribute('aria-label', 'Back');
    });
  });

  describe('Multiple actions', () => {
    it('should render both retry and go back buttons', () => {
      render(
        <ErrorState
          message="Error"
          onRetry={vi.fn()}
          onGoBack={vi.fn()}
        />
      );
      
      expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument();
    });

    it('should not render actions container when no actions provided', () => {
      render(<ErrorState message="Error" />);
      
      const actionsContainer = document.querySelector('.error-state-actions');
      expect(actionsContainer).not.toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have role="alert"', () => {
      render(<ErrorState message="Error" />);
      
      const errorState = document.querySelector('.error-state');
      expect(errorState).toHaveAttribute('role', 'alert');
    });

    it('should have aria-live="assertive"', () => {
      render(<ErrorState message="Error" />);
      
      const errorState = document.querySelector('.error-state');
      expect(errorState).toHaveAttribute('aria-live', 'assertive');
    });

    it('should have aria-hidden on default icon', () => {
      render(<ErrorState message="Error" />);
      
      const icon = document.querySelector('.error-state-icon svg');
      expect(icon).toHaveAttribute('aria-hidden', 'true');
    });
  });

  describe('CSS classes', () => {
    it('should apply correct class to retry button', () => {
      render(<ErrorState message="Error" onRetry={vi.fn()} />);
      
      const button = screen.getByRole('button', { name: /try again/i });
      expect(button).toHaveClass('btn-primary');
    });

    it('should apply correct class to go back button', () => {
      render(<ErrorState message="Error" onGoBack={vi.fn()} />);
      
      const button = screen.getByRole('button', { name: /go back/i });
      expect(button).toHaveClass('btn-secondary');
    });
  });
});
