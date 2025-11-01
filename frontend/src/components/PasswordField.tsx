import React, { useState } from 'react';
import './PasswordField.css';

interface PasswordFieldProps {
  label: string;
  id: string;
  value: string;
  onChange: (value: string) => void;
  onBlur?: () => void;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
  autoComplete?: string;
  showStrengthIndicator?: boolean;
  helperText?: string;
}

const PasswordField: React.FC<PasswordFieldProps> = ({
  label,
  id,
  value,
  onChange,
  onBlur,
  error,
  required,
  disabled,
  placeholder,
  autoComplete = 'current-password',
  showStrengthIndicator = false,
  helperText,
}) => {
  const [showPassword, setShowPassword] = useState(false);
  const hasError = !!error;

  // Password strength calculation
  const getPasswordStrength = (password: string): number => {
    if (!password) return 0;
    
    let strength = 0;
    
    // Length check
    if (password.length >= 8) strength += 1;
    if (password.length >= 12) strength += 1;
    
    // Character variety checks
    if (/[a-z]/.test(password)) strength += 1;
    if (/[A-Z]/.test(password)) strength += 1;
    if (/[0-9]/.test(password)) strength += 1;
    if (/[^a-zA-Z0-9]/.test(password)) strength += 1;
    
    // Return a value from 0-4
    return Math.min(strength, 4);
  };

  const strength = showStrengthIndicator ? getPasswordStrength(value) : 0;
  
  const getStrengthLabel = (strength: number): string => {
    if (strength === 0) return '';
    if (strength <= 1) return 'Weak';
    if (strength === 2) return 'Fair';
    if (strength === 3) return 'Good';
    return 'Strong';
  };

  const getStrengthColor = (strength: number): string => {
    if (strength <= 1) return 'var(--danger, #ef4444)';
    if (strength === 2) return 'var(--warning, #f59e0b)';
    if (strength === 3) return 'var(--success, #10b981)';
    return 'var(--success, #10b981)';
  };

  return (
    <div className={`password-field ${hasError ? 'password-field--error' : ''}`}>
      <label htmlFor={id} className="password-field__label">
        {label}
        {required && <span className="password-field__required" aria-label="required">*</span>}
      </label>
      
      <div className="password-field__input-wrapper">
        <input
          id={id}
          type={showPassword ? 'text' : 'password'}
          className="password-field__input"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          required={required}
          disabled={disabled}
          placeholder={placeholder}
          autoComplete={autoComplete}
          aria-invalid={hasError}
          aria-describedby={
            hasError ? `${id}-error` : helperText ? `${id}-helper` : undefined
          }
        />
        
        <button
          type="button"
          className="password-field__toggle"
          onClick={() => setShowPassword(!showPassword)}
          aria-label={showPassword ? 'Hide password' : 'Show password'}
          disabled={disabled}
          tabIndex={-1}
        >
          {showPassword ? (
            // Eye off icon
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M14.95 14.95a7 7 0 01-9.9 0M3.05 5.05a7 7 0 019.9 0M10 3v1m0 12v1m7-7h-1M4 10H3" />
              <line x1="3" y1="3" x2="17" y2="17" />
            </svg>
          ) : (
            // Eye icon
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M1 10s3-6 9-6 9 6 9 6-3 6-9 6-9-6-9-6z" />
              <circle cx="10" cy="10" r="3" />
            </svg>
          )}
        </button>
      </div>

      {showStrengthIndicator && value && (
        <div className="password-field__strength" aria-live="polite">
          <div className="password-field__strength-bars">
            {[1, 2, 3, 4].map((level) => (
              <div
                key={level}
                className={`password-field__strength-bar ${
                  strength >= level ? 'password-field__strength-bar--active' : ''
                }`}
                style={{
                  backgroundColor: strength >= level ? getStrengthColor(strength) : undefined,
                }}
              />
            ))}
          </div>
          {strength > 0 && (
            <span
              className="password-field__strength-label"
              style={{ color: getStrengthColor(strength) }}
            >
              {getStrengthLabel(strength)}
            </span>
          )}
        </div>
      )}

      {helperText && !hasError && (
        <p id={`${id}-helper`} className="password-field__helper">
          {helperText}
        </p>
      )}

      {hasError && (
        <p id={`${id}-error`} className="password-field__error" role="alert">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <circle cx="8" cy="8" r="7" stroke="currentColor" strokeWidth="2" />
            <path d="M8 4v5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            <circle cx="8" cy="11.5" r="0.75" fill="currentColor" />
          </svg>
          {error}
        </p>
      )}
    </div>
  );
};

export default PasswordField;
