import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import axios from 'axios';
import './Login.css';

const Login: React.FC = () => {
  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [showForgotPassword, setShowForgotPassword] = useState(false);
  const [resetEmail, setResetEmail] = useState('');
  const [resetSuccess, setResetSuccess] = useState('');
  const [resetError, setResetError] = useState('');
  const [isSubmittingReset, setIsSubmittingReset] = useState(false);
  const { login, register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      if (isLogin) {
        await login(username, password);
      } else {
        await register(username, email, password);
      }
      navigate('/');
    } catch (err: any) {
      // Enhanced error handling for account lockout
      const errorResponse = err.response?.data;
      if (errorResponse?.locked_until) {
        setError(`${errorResponse.error}. You can try again in ${errorResponse.retry_in_mins} minutes or reset your password.`);
      } else if (errorResponse?.attempts_remaining !== undefined) {
        setError(`${errorResponse.error}. ${errorResponse.attempts_remaining} attempts remaining before account lockout.`);
      } else {
        setError(errorResponse?.error || 'An error occurred');
      }
    }
  };

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setResetError('');
    setResetSuccess('');
    setIsSubmittingReset(true);

    try {
      const response = await axios.post('/api/request-password-reset', {
        email: resetEmail,
      });
      setResetSuccess(response.data.message || 'If the email exists, a password reset link will be sent.');
      setResetEmail('');
      // Close modal after 3 seconds
      setTimeout(() => {
        setShowForgotPassword(false);
        setResetSuccess('');
      }, 3000);
    } catch (err: any) {
      setResetError(err.response?.data?.error || 'Failed to send reset email');
    } finally {
      setIsSubmittingReset(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>Haws Volunteers</h1>
        <h2>{isLogin ? 'Login' : 'Register'}</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </div>
          {!isLogin && (
            <div className="form-group">
              <label htmlFor="email">Email</label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
          )}
          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
          </div>
          {error && <div className="error">{error}</div>}
          <button type="submit" className="btn-primary">
            {isLogin ? 'Login' : 'Register'}
          </button>
        </form>
        {isLogin && (
          <button
            onClick={() => setShowForgotPassword(true)}
            className="forgot-password-link"
            type="button"
          >
            Forgot Password?
          </button>
        )}
        <button
          onClick={() => {
            setIsLogin(!isLogin);
            setError('');
          }}
          className="btn-secondary"
        >
          {isLogin ? 'Need an account? Register' : 'Have an account? Login'}
        </button>
      </div>

      {/* Forgot Password Modal */}
      {showForgotPassword && (
        <div className="modal-overlay" onClick={() => setShowForgotPassword(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h2>Reset Password</h2>
            <p>Enter your email address and we'll send you a link to reset your password.</p>
            <form onSubmit={handleForgotPassword}>
              <div className="form-group">
                <label htmlFor="reset-email">Email</label>
                <input
                  id="reset-email"
                  type="email"
                  value={resetEmail}
                  onChange={(e) => setResetEmail(e.target.value)}
                  required
                  placeholder="your-email@example.com"
                />
              </div>
              {resetError && <div className="error">{resetError}</div>}
              {resetSuccess && <div className="success">{resetSuccess}</div>}
              <div className="modal-actions">
                <button
                  type="button"
                  onClick={() => {
                    setShowForgotPassword(false);
                    setResetError('');
                    setResetSuccess('');
                    setResetEmail('');
                  }}
                  className="btn-secondary"
                  disabled={isSubmittingReset}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={isSubmittingReset}
                >
                  {isSubmittingReset ? 'Sending...' : 'Send Reset Link'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Login;
