import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import axios from 'axios';
import { getPasswordStrength } from '../utils/passwordStrength';
import './Login.css';

const REDIRECT_TIMEOUT = 2000; // milliseconds

const SetupPassword: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [token, setToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    const tokenParam = searchParams.get('token');
    if (!tokenParam) {
      setError('Invalid setup link. Please contact your administrator.');
    } else {
      setToken(tokenParam);
    }
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters long');
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await axios.post('/api/setup-password', {
        token,
        new_password: newPassword,
      });

      setSuccess(response.data.message || 'Password has been set successfully!');
      
      // Redirect to login after timeout
      setTimeout(() => {
        navigate('/login');
      }, REDIRECT_TIMEOUT);
    } catch (err: unknown) {
      let errorMessage = 'Failed to set password. Please try again or contact your administrator.';
      if (axios.isAxiosError(err)) {
        errorMessage = err.response?.data?.error || errorMessage;
      }
      setError(errorMessage);
    } finally {
      setIsSubmitting(false);
    }
  };

  const passwordStrength = newPassword ? getPasswordStrength(newPassword) : null;

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>Haws Volunteers</h1>
        <h2>Welcome! Set Your Password</h2>
        <p style={{ textAlign: 'center', color: '#666', marginBottom: '1.5rem' }}>
          Create a secure password to access your account
        </p>
        
        {!token ? (
          <div className="error">
            Invalid setup link. Please contact your administrator for a new invitation.
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="new-password">Password</label>
              <div style={{ position: 'relative' }}>
                <input
                  id="new-password"
                  type={showPassword ? 'text' : 'password'}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  minLength={8}
                  placeholder="Enter your password"
                  disabled={isSubmitting || !!success}
                  style={{ paddingRight: '2.5rem' }}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  style={{
                    position: 'absolute',
                    right: '0.5rem',
                    top: '50%',
                    transform: 'translateY(-50%)',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    padding: '0.25rem',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: '#666'
                  }}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24M1 1l22 22"/>
                    </svg>
                  ) : (
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                      <circle cx="12" cy="12" r="3"/>
                    </svg>
                  )}
                </button>
              </div>
              {passwordStrength && (
                <div style={{ marginTop: '0.5rem' }}>
                  <div style={{ 
                    width: '100%', 
                    height: '4px', 
                    backgroundColor: '#e5e7eb', 
                    borderRadius: '2px',
                    overflow: 'hidden'
                  }}>
                    <div style={{
                      width: passwordStrength.strength === 'weak' ? '33%' : passwordStrength.strength === 'medium' ? '66%' : '100%',
                      height: '100%',
                      backgroundColor: passwordStrength.color,
                      transition: 'width 0.3s ease'
                    }} />
                  </div>
                  <span style={{ 
                    fontSize: '0.875rem', 
                    color: passwordStrength.color,
                    marginTop: '0.25rem',
                    display: 'block'
                  }}>
                    {passwordStrength.label}
                  </span>
                </div>
              )}
              <small style={{ display: 'block', marginTop: '0.25rem', color: '#666' }}>
                At least 8 characters. Use a mix of letters, numbers, and symbols for better security.
              </small>
            </div>
            <div className="form-group">
              <label htmlFor="confirm-password">Confirm Password</label>
              <input
                id="confirm-password"
                type={showPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={8}
                placeholder="Confirm your password"
                disabled={isSubmitting || !!success}
              />
            </div>
            {error && <div className="error">{error}</div>}
            {success && <div className="success">{success}</div>}
            <button
              type="submit"
              className="btn-primary"
              disabled={isSubmitting || !!success}
            >
              {isSubmitting ? 'Setting Password...' : success ? 'Success! Redirecting...' : 'Set Password & Sign In'}
            </button>
          </form>
        )}
        
        <button
          onClick={() => navigate('/login')}
          className="btn-secondary"
          disabled={isSubmitting}
        >
          Back to Login
        </button>
      </div>
    </div>
  );
};

export default SetupPassword;
