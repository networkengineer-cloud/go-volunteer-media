import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useSiteSettings } from '../hooks/useSiteSettings';
import axios from 'axios';
import './Login.css';

const REDIRECT_TIMEOUT = 2000; // milliseconds

const ResetPassword: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [token, setToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const navigate = useNavigate();
  const { settings } = useSiteSettings();

  useEffect(() => {
    const tokenParam = searchParams.get('token');
    if (!tokenParam) {
      setError('Invalid reset link. Please request a new password reset.');
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
      const response = await axios.post('/api/reset-password', {
        token,
        new_password: newPassword,
      });

      setSuccess(response.data.message || 'Password has been reset successfully!');
      
      // Redirect to login after timeout
      setTimeout(() => {
        navigate('/login');
      }, REDIRECT_TIMEOUT);
    } catch (err: unknown) {
      setError((err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to reset password. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>{settings.site_name}</h1>
        <h2>Reset Your Password</h2>
        
        {!token ? (
          <div className="error">
            Invalid reset link. Please request a new password reset from the login page.
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="new-password">New Password</label>
              <input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={8}
                placeholder="Enter new password"
                disabled={isSubmitting || !!success}
              />
            </div>
            <div className="form-group">
              <label htmlFor="confirm-password">Confirm Password</label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={8}
                placeholder="Confirm new password"
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
              {isSubmitting ? 'Resetting...' : success ? 'Success! Redirecting...' : 'Reset Password'}
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

export default ResetPassword;
