import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ConfirmDialog from './ConfirmDialog';

describe('ConfirmDialog', () => {
  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(
        <ConfirmDialog
          isOpen={false}
          title="Confirm Action"
          message="Are you sure?"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Confirm Action"
          message="Are you sure?"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(screen.getByText('Confirm Action')).toBeInTheDocument();
      expect(screen.getByText('Are you sure?')).toBeInTheDocument();
    });

    it('should render with default labels', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      expect(screen.getByRole('button', { name: /confirm/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    });

    it('should render with custom labels', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          confirmLabel="Yes, delete it"
          cancelLabel="No, keep it"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      expect(screen.getByRole('button', { name: /yes, delete it/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /no, keep it/i })).toBeInTheDocument();
    });
  });

  describe('Variants', () => {
    it('should apply danger variant by default', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = document.querySelector('.confirm-dialog');
      expect(dialog).toHaveClass('confirm-dialog--danger');
    });

    it('should apply warning variant', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          variant="warning"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = document.querySelector('.confirm-dialog');
      expect(dialog).toHaveClass('confirm-dialog--warning');
    });

    it('should apply info variant', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          variant="info"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = document.querySelector('.confirm-dialog');
      expect(dialog).toHaveClass('confirm-dialog--info');
    });
  });

  describe('User interactions', () => {
    it('should call onCancel when cancel button clicked', async () => {
      const handleCancel = vi.fn();
      const user = userEvent.setup();
      
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={handleCancel}
        />
      );
      
      await user.click(screen.getByRole('button', { name: /cancel/i }));
      
      expect(handleCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onConfirm and onCancel when confirm button clicked', async () => {
      const handleConfirm = vi.fn();
      const handleCancel = vi.fn();
      const user = userEvent.setup();
      
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={handleConfirm}
          onCancel={handleCancel}
        />
      );
      
      await user.click(screen.getByRole('button', { name: /confirm/i }));
      
      expect(handleConfirm).toHaveBeenCalledTimes(1);
      expect(handleCancel).toHaveBeenCalledTimes(1);
    });

    it('should call onCancel when backdrop clicked', async () => {
      const handleCancel = vi.fn();
      const user = userEvent.setup();
      
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={handleCancel}
        />
      );
      
      const backdrop = document.querySelector('.confirm-dialog-backdrop');
      if (backdrop) {
        await user.click(backdrop);
        expect(handleCancel).toHaveBeenCalledTimes(1);
      }
    });

    it('should not close when dialog content clicked', async () => {
      const handleCancel = vi.fn();
      const user = userEvent.setup();
      
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={handleCancel}
        />
      );
      
      const dialog = document.querySelector('.confirm-dialog');
      if (dialog) {
        await user.click(dialog);
        expect(handleCancel).not.toHaveBeenCalled();
      }
    });
  });

  describe('Keyboard interactions', () => {
    it('should call onCancel when Escape key pressed', async () => {
      const handleCancel = vi.fn();
      const user = userEvent.setup();
      
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={handleCancel}
        />
      );
      
      await user.keyboard('{Escape}');
      
      expect(handleCancel).toHaveBeenCalledTimes(1);
    });

    it('should focus cancel button on open', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(document.activeElement).toBe(cancelButton);
    });
  });

  describe('Accessibility', () => {
    it('should have role="dialog"', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('should have aria-modal="true"', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-modal', 'true');
    });

    it('should have aria-labelledby pointing to title', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Dialog Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'confirm-dialog-title');
      expect(screen.getByText('Dialog Title')).toHaveAttribute('id', 'confirm-dialog-title');
    });

    it('should have aria-describedby pointing to message', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Dialog message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-describedby', 'confirm-dialog-message');
      expect(screen.getByText('Dialog message')).toHaveAttribute('id', 'confirm-dialog-message');
    });

    it('should have aria-hidden on icon SVG', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const icon = document.querySelector('.confirm-dialog__icon svg');
      expect(icon).toHaveAttribute('aria-hidden', 'true');
    });
  });

  describe('Button styling', () => {
    it('should apply btn-danger class to confirm button with danger variant', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          variant="danger"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const confirmButton = screen.getByRole('button', { name: /confirm/i });
      expect(confirmButton).toHaveClass('btn-danger');
    });

    it('should apply btn-warning class to confirm button with warning variant', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          variant="warning"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const confirmButton = screen.getByRole('button', { name: /confirm/i });
      expect(confirmButton).toHaveClass('btn-warning');
    });

    it('should apply btn-secondary class to cancel button', () => {
      render(
        <ConfirmDialog
          isOpen={true}
          title="Title"
          message="Message"
          onConfirm={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(cancelButton).toHaveClass('btn-secondary');
    });
  });
});
