import React from 'react';
import './FormField.css';

interface FormFieldProps {
  label: string;
  id: string;
  type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'textarea';
  value: string | number;
  onChange: (value: string) => void;
  onBlur?: () => void;
  error?: string;
  success?: boolean;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
  helperText?: string;
  autoComplete?: string;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  rows?: number;
}

const FormField: React.FC<FormFieldProps> = ({
  label,
  id,
  type = 'text',
  value,
  onChange,
  onBlur,
  error,
  success,
  required,
  disabled,
  placeholder,
  helperText,
  autoComplete,
  minLength,
  maxLength,
  pattern,
  rows = 4,
}) => {
  const hasError = !!error;
  const showSuccess = success && !hasError && value;

  return (
    <div className={`form-field ${hasError ? 'form-field--error' : ''} ${showSuccess ? 'form-field--success' : ''}`}>
      <label htmlFor={id} className="form-field__label">
        {label}
        {required && <span className="form-field__required" aria-label="required">*</span>}
      </label>
      
      {type === 'textarea' ? (
        <textarea
          id={id}
          className="form-field__textarea"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          required={required}
          disabled={disabled}
          placeholder={placeholder}
          aria-invalid={hasError}
          aria-describedby={
            hasError ? `${id}-error` : helperText ? `${id}-helper` : undefined
          }
          rows={rows}
        />
      ) : (
        <div className="form-field__input-wrapper">
          <input
            id={id}
            type={type}
            className="form-field__input"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onBlur={onBlur}
            required={required}
            disabled={disabled}
            placeholder={placeholder}
            autoComplete={autoComplete}
            minLength={minLength}
            maxLength={maxLength}
            pattern={pattern}
            aria-invalid={hasError}
            aria-describedby={
              hasError ? `${id}-error` : helperText ? `${id}-helper` : undefined
            }
          />
          {showSuccess && (
            <div className="form-field__success-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M16.25 5.625L7.5 14.375L3.75 10.625"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
          )}
        </div>
      )}

      {helperText && !hasError && (
        <p id={`${id}-helper`} className="form-field__helper">
          {helperText}
        </p>
      )}

      {hasError && (
        <p id={`${id}-error`} className="form-field__error" role="alert">
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

export default FormField;
