import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import FormField from './FormField';

describe('FormField', () => {
  describe('Text input rendering', () => {
    it('should render text input with label', () => {
      const handleChange = vi.fn();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          value=""
          onChange={handleChange}
        />
      );
      
      expect(screen.getByLabelText('Test Label')).toBeInTheDocument();
      expect(screen.getByRole('textbox')).toBeInTheDocument();
    });

    it('should display current value', () => {
      const handleChange = vi.fn();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          value="test value"
          onChange={handleChange}
        />
      );
      
      expect(screen.getByDisplayValue('test value')).toBeInTheDocument();
    });

    it('should call onChange when value changes', async () => {
      const handleChange = vi.fn();
      const user = userEvent.setup();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          value=""
          onChange={handleChange}
        />
      );
      
      const input = screen.getByRole('textbox');
      await user.type(input, 'new');
      
      expect(handleChange).toHaveBeenCalledWith('n');
      expect(handleChange).toHaveBeenCalledWith('e');
      expect(handleChange).toHaveBeenCalledWith('w');
    });

    it('should call onBlur when field loses focus', async () => {
      const handleBlur = vi.fn();
      const user = userEvent.setup();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          value=""
          onChange={vi.fn()}
          onBlur={handleBlur}
        />
      );
      
      const input = screen.getByRole('textbox');
      await user.click(input);
      await user.tab();
      
      expect(handleBlur).toHaveBeenCalledTimes(1);
    });
  });

  describe('Textarea rendering', () => {
    it('should render textarea when type is textarea', () => {
      const handleChange = vi.fn();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          type="textarea"
          value=""
          onChange={handleChange}
        />
      );
      
      expect(screen.getByRole('textbox')).toBeInTheDocument();
      const textarea = screen.getByRole('textbox');
      expect(textarea.tagName).toBe('TEXTAREA');
    });

    it('should set rows attribute on textarea', () => {
      const handleChange = vi.fn();
      
      render(
        <FormField
          id="test-field"
          label="Test Label"
          type="textarea"
          value=""
          onChange={handleChange}
          rows={10}
        />
      );
      
      const textarea = screen.getByRole('textbox');
      expect(textarea).toHaveAttribute('rows', '10');
    });
  });

  describe('Input types', () => {
    it('should render email input', () => {
      render(
        <FormField
          id="email"
          label="Email"
          type="email"
          value=""
          onChange={vi.fn()}
        />
      );
      
      const input = screen.getByLabelText('Email');
      expect(input).toHaveAttribute('type', 'email');
    });

    it('should render password input', () => {
      render(
        <FormField
          id="password"
          label="Password"
          type="password"
          value=""
          onChange={vi.fn()}
        />
      );
      
      const input = screen.getByLabelText('Password');
      expect(input).toHaveAttribute('type', 'password');
    });

    it('should render number input', () => {
      render(
        <FormField
          id="age"
          label="Age"
          type="number"
          value={0}
          onChange={vi.fn()}
        />
      );
      
      const input = screen.getByLabelText('Age');
      expect(input).toHaveAttribute('type', 'number');
    });
  });

  describe('Required field', () => {
    it('should show required indicator', () => {
      render(
        <FormField
          id="test"
          label="Required Field"
          value=""
          onChange={vi.fn()}
          required
        />
      );
      
      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByLabelText('required')).toBeInTheDocument();
    });

    it('should set required attribute on input', () => {
      render(
        <FormField
          id="test"
          label="Required Field"
          value=""
          onChange={vi.fn()}
          required
        />
      );
      
      expect(screen.getByRole('textbox')).toHaveAttribute('required');
    });
  });

  describe('Disabled state', () => {
    it('should disable input when disabled prop is true', () => {
      render(
        <FormField
          id="test"
          label="Disabled Field"
          value=""
          onChange={vi.fn()}
          disabled
        />
      );
      
      expect(screen.getByRole('textbox')).toBeDisabled();
    });

    it('should not accept input when disabled', async () => {
      const handleChange = vi.fn();
      const user = userEvent.setup();
      
      render(
        <FormField
          id="test"
          label="Disabled Field"
          value=""
          onChange={handleChange}
          disabled
        />
      );
      
      const input = screen.getByRole('textbox');
      await user.type(input, 'test');
      
      expect(handleChange).not.toHaveBeenCalled();
    });
  });

  describe('Error state', () => {
    it('should display error message', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          error="This field is required"
        />
      );
      
      expect(screen.getByRole('alert')).toBeInTheDocument();
      expect(screen.getByText('This field is required')).toBeInTheDocument();
    });

    it('should apply error class to form field', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          error="Error message"
        />
      );
      
      const formField = document.querySelector('.form-field');
      expect(formField).toHaveClass('form-field--error');
    });

    it('should set aria-invalid on input', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          error="Error message"
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('aria-invalid', 'true');
    });

    it('should link error to input with aria-describedby', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          error="Error message"
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('aria-describedby', 'test-error');
    });
  });

  describe('Success state', () => {
    it('should show success icon when success is true and value exists', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value="valid value"
          onChange={vi.fn()}
          success
        />
      );
      
      const successIcon = document.querySelector('.form-field__success-icon');
      expect(successIcon).toBeInTheDocument();
    });

    it('should not show success icon when value is empty', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          success
        />
      );
      
      const successIcon = document.querySelector('.form-field__success-icon');
      expect(successIcon).not.toBeInTheDocument();
    });

    it('should not show success icon when there is an error', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value="value"
          onChange={vi.fn()}
          success
          error="Error message"
        />
      );
      
      const successIcon = document.querySelector('.form-field__success-icon');
      expect(successIcon).not.toBeInTheDocument();
    });

    it('should apply success class to form field', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value="valid value"
          onChange={vi.fn()}
          success
        />
      );
      
      const formField = document.querySelector('.form-field');
      expect(formField).toHaveClass('form-field--success');
    });
  });

  describe('Helper text', () => {
    it('should display helper text', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          helperText="This is helper text"
        />
      );
      
      expect(screen.getByText('This is helper text')).toBeInTheDocument();
    });

    it('should not display helper text when there is an error', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          helperText="Helper text"
          error="Error message"
        />
      );
      
      expect(screen.queryByText('Helper text')).not.toBeInTheDocument();
      expect(screen.getByText('Error message')).toBeInTheDocument();
    });

    it('should link helper text to input with aria-describedby', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          helperText="Helper text"
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('aria-describedby', 'test-helper');
    });
  });

  describe('Additional attributes', () => {
    it('should set placeholder', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          placeholder="Enter text"
        />
      );
      
      expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
    });

    it('should set autoComplete', () => {
      render(
        <FormField
          id="email"
          label="Email"
          type="email"
          value=""
          onChange={vi.fn()}
          autoComplete="email"
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('autocomplete', 'email');
    });

    it('should set minLength', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          minLength={5}
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('minlength', '5');
    });

    it('should set maxLength', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          maxLength={100}
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('maxlength', '100');
    });

    it('should set pattern', () => {
      render(
        <FormField
          id="test"
          label="Test Field"
          value=""
          onChange={vi.fn()}
          pattern="[0-9]*"
        />
      );
      
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('pattern', '[0-9]*');
    });
  });
});
