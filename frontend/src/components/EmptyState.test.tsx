import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import EmptyState from './EmptyState';

describe('EmptyState', () => {
  describe('Rendering', () => {
    it('should render title and description', () => {
      render(
        <EmptyState
          title="No items found"
          description="There are no items to display"
        />
      );

      expect(screen.getByRole('heading', { name: /no items found/i })).toBeInTheDocument();
      expect(screen.getByText(/there are no items to display/i)).toBeInTheDocument();
    });

    it('should render with icon', () => {
      const TestIcon = () => <svg data-testid="test-icon">icon</svg>;
      
      render(
        <EmptyState
          icon={<TestIcon />}
          title="No items"
          description="Add some items"
        />
      );

      expect(screen.getByTestId('test-icon')).toBeInTheDocument();
    });

    it('should render without icon', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
        />
      );

      const iconContainer = document.querySelector('.empty-state-icon');
      expect(iconContainer).not.toBeInTheDocument();
    });
  });

  describe('Actions', () => {
    it('should render primary action button', () => {
      const handleClick = vi.fn();
      
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          primaryAction={{
            label: 'Add Item',
            onClick: handleClick,
          }}
        />
      );

      expect(screen.getByRole('button', { name: /add item/i })).toBeInTheDocument();
    });

    it('should call primary action onClick', async () => {
      const handleClick = vi.fn();
      const user = userEvent.setup();
      
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          primaryAction={{
            label: 'Add Item',
            onClick: handleClick,
          }}
        />
      );

      await user.click(screen.getByRole('button', { name: /add item/i }));
      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should render secondary action button', () => {
      const handleClick = vi.fn();
      
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          secondaryAction={{
            label: 'Learn More',
            onClick: handleClick,
          }}
        />
      );

      expect(screen.getByRole('button', { name: /learn more/i })).toBeInTheDocument();
    });

    it('should call secondary action onClick', async () => {
      const handleClick = vi.fn();
      const user = userEvent.setup();
      
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          secondaryAction={{
            label: 'Learn More',
            onClick: handleClick,
          }}
        />
      );

      await user.click(screen.getByRole('button', { name: /learn more/i }));
      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should render both primary and secondary actions', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          primaryAction={{
            label: 'Add Item',
            onClick: vi.fn(),
          }}
          secondaryAction={{
            label: 'Learn More',
            onClick: vi.fn(),
          }}
        />
      );

      expect(screen.getByRole('button', { name: /add item/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /learn more/i })).toBeInTheDocument();
    });

    it('should not render actions container when no actions provided', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
        />
      );

      const actionsContainer = document.querySelector('.empty-state-actions');
      expect(actionsContainer).not.toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have role="status"', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
        />
      );

      const emptyState = document.querySelector('.empty-state');
      expect(emptyState).toHaveAttribute('role', 'status');
    });

    it('should have aria-live="polite"', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
        />
      );

      const emptyState = document.querySelector('.empty-state');
      expect(emptyState).toHaveAttribute('aria-live', 'polite');
    });

    it('should have aria-label on primary action button', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          primaryAction={{
            label: 'Add Item',
            onClick: vi.fn(),
          }}
        />
      );

      const button = screen.getByRole('button', { name: /add item/i });
      expect(button).toHaveAttribute('aria-label', 'Add Item');
    });

    it('should have aria-label on secondary action button', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          secondaryAction={{
            label: 'Learn More',
            onClick: vi.fn(),
          }}
        />
      );

      const button = screen.getByRole('button', { name: /learn more/i });
      expect(button).toHaveAttribute('aria-label', 'Learn More');
    });
  });

  describe('Styling classes', () => {
    it('should apply correct CSS classes to primary action', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          primaryAction={{
            label: 'Add Item',
            onClick: vi.fn(),
          }}
        />
      );

      const button = screen.getByRole('button', { name: /add item/i });
      expect(button).toHaveClass('btn-primary');
    });

    it('should apply correct CSS classes to secondary action', () => {
      render(
        <EmptyState
          title="No items"
          description="Add some items"
          secondaryAction={{
            label: 'Learn More',
            onClick: vi.fn(),
          }}
        />
      );

      const button = screen.getByRole('button', { name: /learn more/i });
      expect(button).toHaveClass('btn-secondary');
    });
  });
});
