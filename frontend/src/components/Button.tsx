import React from 'react';
import './Button.css';

interface ButtonProps {
  children: React.ReactNode;
  onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void;
  type?: 'button' | 'submit' | 'reset';
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'small' | 'medium' | 'large';
  disabled?: boolean;
  loading?: boolean;
  fullWidth?: boolean;
  'aria-label'?: string;
  className?: string;
}

const Button: React.FC<ButtonProps> = ({
  children,
  onClick,
  type = 'button',
  variant = 'primary',
  size = 'medium',
  disabled = false,
  loading = false,
  fullWidth = false,
  'aria-label': ariaLabel,
  className = '',
}) => {
  const isDisabled = disabled || loading;

  return (
    <button
      type={type}
      className={`
        button
        button--${variant}
        button--${size}
        ${fullWidth ? 'button--full-width' : ''}
        ${loading ? 'button--loading' : ''}
        ${className}
      `.trim()}
      onClick={onClick}
      disabled={isDisabled}
      aria-label={ariaLabel}
      aria-busy={loading}
    >
      {loading && (
        <span className="button__spinner" aria-hidden="true">
          <svg
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M8 1.5v3M8 11.5v3M14.5 8h-3M4.5 8h-3M12.3033 3.69668l-2.12132 2.12132M5.81802 10.182l-2.12132 2.1213M12.3033 12.3033l-2.12132-2.1213M5.81802 5.81802l-2.12132-2.12132"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            />
          </svg>
        </span>
      )}
      <span className={loading ? 'button__content--loading' : ''}>{children}</span>
    </button>
  );
};

export default Button;
