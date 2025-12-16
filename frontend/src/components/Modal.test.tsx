import { describe, it, expect, vi, afterEach } from 'vitest';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import Modal from './Modal';

describe('Modal', () => {
  // Clean up body overflow style after each test
  afterEach(() => {
    document.body.style.overflow = '';
  });

  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(
        <Modal isOpen={false} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('should display title', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );
      expect(screen.getByText('Test Modal')).toBeInTheDocument();
    });

    it('should display children content', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          <div>Modal Content</div>
        </Modal>
      );
      expect(screen.getByText('Modal Content')).toBeInTheDocument();
    });
  });

  describe('Size variants', () => {
    it('should apply medium size by default', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );
      const modal = screen.getByRole('dialog');
      expect(modal).toHaveClass('modal--medium');
    });

    it('should apply small size', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal" size="small">
          Content
        </Modal>
      );
      const modal = screen.getByRole('dialog');
      expect(modal).toHaveClass('modal--small');
    });

    it('should apply large size', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal" size="large">
          Content
        </Modal>
      );
      const modal = screen.getByRole('dialog');
      expect(modal).toHaveClass('modal--large');
    });
  });

  describe('Close functionality', () => {
    it('should call onClose when close button is clicked', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test Modal">
          Content
        </Modal>
      );

      const closeButton = screen.getByLabelText('Close modal');
      await user.click(closeButton);

      expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('should call onClose when backdrop is clicked', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test Modal">
          Content
        </Modal>
      );

      const backdrop = document.querySelector('.modal-backdrop');
      if (backdrop) {
        await user.click(backdrop);
      }

      expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('should not call onClose when modal content is clicked', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test Modal">
          Content
        </Modal>
      );

      const modal = screen.getByRole('dialog');
      await user.click(modal);

      expect(handleClose).not.toHaveBeenCalled();
    });

    it('should call onClose when Escape key is pressed', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose} title="Test Modal">
          Content
        </Modal>
      );

      await user.keyboard('{Escape}');

      expect(handleClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('Body scroll lock', () => {
    it('should set body overflow to hidden when open', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');
    });

    it('should restore body overflow when closed', () => {
      const { rerender } = render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');

      rerender(
        <Modal isOpen={false} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );

      expect(document.body.style.overflow).toBe('');
    });
  });

  describe('Focus management', () => {
    it('should not re-run effects when onClose reference is stable', async () => {
      const handleClose = vi.fn();
      const effectSpy = vi.fn();

      // Component that tracks effect runs
      const TestComponent = ({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) => {
        const [count, setCount] = React.useState(0);

        React.useEffect(() => {
          effectSpy();
        }, [isOpen, onClose]);

        return (
          <div>
            <button onClick={() => setCount(c => c + 1)}>Increment</button>
            <Modal isOpen={isOpen} onClose={onClose} title="Test Modal">
              <textarea placeholder="Type here" />
              <div>Count: {count}</div>
            </Modal>
          </div>
        );
      };

      const user = userEvent.setup();
      render(<TestComponent isOpen={true} onClose={handleClose} />);

      // Initial render triggers effect once
      expect(effectSpy).toHaveBeenCalledTimes(1);

      // Trigger state change in parent
      const button = screen.getByText('Increment');
      await user.click(button);

      // Effect should NOT run again because onClose is stable
      expect(effectSpy).toHaveBeenCalledTimes(1);
    });

    it('should preserve focus in textarea when parent re-renders', async () => {
      let triggerRerender: (() => void) | null = null;

      const TestComponent = () => {
        const [isOpen, setIsOpen] = React.useState(true);
        const [renderCount, setRenderCount] = React.useState(0);

        // Memoized close handler (simulates the fix)
        const handleClose = React.useCallback(() => {
          setIsOpen(false);
        }, []);

        React.useEffect(() => {
          triggerRerender = () => setRenderCount((c) => c + 1);
          return () => {
            triggerRerender = null;
          };
        }, []);

        return (
          <div>
            <Modal isOpen={isOpen} onClose={handleClose} title="Test Modal">
              <textarea id="test-textarea" placeholder="Type here" />
              <div>Renders: {renderCount}</div>
            </Modal>
          </div>
        );
      };

      const user = userEvent.setup();
      render(<TestComponent />);

      const textarea = screen.getByPlaceholderText('Type here') as HTMLTextAreaElement;

      // Focus textarea
      textarea.focus();
      expect(document.activeElement).toBe(textarea);

      // Type something
      await user.type(textarea, 'Test');
      expect(textarea.value).toBe('Test');

      // Trigger parent re-render
      act(() => {
        if (!triggerRerender) throw new Error('triggerRerender not set');
        triggerRerender();
      });

      // Focus should still be on textarea
      expect(document.activeElement).toBe(textarea);

      // Should still be able to type
      await user.type(textarea, ' more');
      expect(textarea.value).toBe('Test more');
    });
  });

  describe('Focus trap', () => {
    it('should trap focus within modal', async () => {
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          <button>First</button>
          <button>Second</button>
          <button>Third</button>
        </Modal>
      );

      // Tab through buttons
      await user.tab();
      await user.tab();
      await user.tab();
      await user.tab();

      // Focus should stay within modal (cycle back to first element)
      await waitFor(() => {
        const activeElement = document.activeElement;
        const modal = screen.getByRole('dialog');
        expect(modal.contains(activeElement)).toBe(true);
      });
    });

    it('should handle shift-tab to move backwards', async () => {
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          <button>First</button>
          <button>Last</button>
        </Modal>
      );

      // Tab to second button
      await user.tab();
      await user.tab();

      // Shift-tab back
      await user.keyboard('{Shift>}{Tab}{/Shift}');

      // Should cycle focus within modal
      await waitFor(() => {
        const activeElement = document.activeElement;
        const modal = screen.getByRole('dialog');
        expect(modal.contains(activeElement)).toBe(true);
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );

      const modal = screen.getByRole('dialog');
      expect(modal).toHaveAttribute('aria-modal', 'true');
      expect(modal).toHaveAttribute('aria-labelledby', 'modal-title');
    });

    it('should have close button with aria-label', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
          Content
        </Modal>
      );

      const closeButton = screen.getByLabelText('Close modal');
      expect(closeButton).toBeInTheDocument();
    });
  });
});
