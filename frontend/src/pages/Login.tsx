import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { useSiteSettings } from '../hooks/useSiteSettings';
import axios from 'axios';
import FormField from '../components/FormField';
import PasswordField from '../components/PasswordField';
import Button from '../components/Button';
import Modal from '../components/Modal';
import './Login.css';

const SUCCESS_MESSAGE_TIMEOUT = 3000; // milliseconds

const Login: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showForgotPassword, setShowForgotPassword] = useState(false);
  const [resetEmail, setResetEmail] = useState('');
  const [resetSuccess, setResetSuccess] = useState('');
  const [resetError, setResetError] = useState('');
  const [isSubmittingReset, setIsSubmittingReset] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  
  // Form validation states
  const [touched, setTouched] = useState({
    username: false,
    password: false,
  });
  const [errors, setErrors] = useState({
    username: '',
    password: '',
  });
  
  const { login } = useAuth();
  const navigate = useNavigate();
  const toast = useToast();
  const [searchParams] = useSearchParams();
  const { settings } = useSiteSettings();

  // Check if redirected due to session expiration
  useEffect(() => {
    if (searchParams.get('expired') === 'true') {
      setSessionExpired(true);
      toast.showWarning('Your session has expired. Please log in again.');
    }
  }, [searchParams, toast]);

  // Validation functions
  const validateUsername = (value: string): string => {
    if (!value) return 'Username is required';
    if (value.length < 3) return 'Username must be at least 3 characters';
    return '';
  };

  const validatePassword = (value: string): string => {
    if (!value) return 'Password is required';
    if (value.length < 8) return 'Password must be at least 8 characters';
    return '';
  };

  // Handle field blur for validation
  const handleBlur = (field: 'username' | 'password') => {
    setTouched({ ...touched, [field]: true });
    
    let error = '';
    if (field === 'username') {
      error = validateUsername(username);
    } else if (field === 'password') {
      error = validatePassword(password);
    }
    
    setErrors({ ...errors, [field]: error });
  };

  // Handle field changes
  const handleUsernameChange = (value: string) => {
    setUsername(value);
    if (touched.username) {
      setErrors({ ...errors, username: validateUsername(value) });
    }
  };

  const handlePasswordChange = (value: string) => {
    setPassword(value);
    if (touched.password) {
      setErrors({ ...errors, password: validatePassword(value) });
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsSubmitting(true);

    // Validate all fields
    const usernameError = validateUsername(username);
    const passwordError = validatePassword(password);

    if (usernameError || passwordError) {
      setErrors({
        username: usernameError,
        password: passwordError,
      });
      setTouched({
        username: true,
        password: true,
      });
      setIsSubmitting(false);
      return;
    }

    try {
      await login(username, password);
      toast.showSuccess('Successfully logged in!');
      
      // Redirect back to original page if there was one
      const redirectPath = sessionStorage.getItem('redirectAfterLogin');
      if (redirectPath) {
        sessionStorage.removeItem('redirectAfterLogin');
        navigate(redirectPath);
      } else {
        navigate('/dashboard');
      }
    } catch (err: unknown) {
      // Enhanced error handling for account lockout
      const errorResponse = err.response?.data;
      let errorMessage = '';
      
      if (errorResponse?.locked_until) {
        errorMessage = `${errorResponse.error}. You can try again in ${errorResponse.retry_in_mins} minutes or reset your password.`;
      } else if (errorResponse?.attempts_remaining !== undefined) {
        errorMessage = `${errorResponse.error}. ${errorResponse.attempts_remaining} attempts remaining before account lockout.`;
      } else {
        errorMessage = errorResponse?.error || 'An error occurred';
      }
      
      setError(errorMessage);
      toast.showError(errorMessage);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setResetError('');
    setResetSuccess('');
    setIsSubmittingReset(true);

    // Simple email validation
    if (!resetEmail || !resetEmail.includes('@')) {
      setResetError('Please enter a valid email address');
      setIsSubmittingReset(false);
      return;
    }

    try {
      const response = await axios.post('/api/request-password-reset', {
        email: resetEmail,
      });
      const successMsg = response.data.message || 'If the email exists, a password reset link will be sent.';
      setResetSuccess(successMsg);
      toast.showSuccess(successMsg);
      setResetEmail('');
      // Close modal after timeout
      setTimeout(() => {
        setShowForgotPassword(false);
        setResetSuccess('');
      }, SUCCESS_MESSAGE_TIMEOUT);
    } catch (err: unknown) {
      const errorMsg = err.response?.data?.error || 'Failed to send reset email';
      setResetError(errorMsg);
      toast.showError(errorMsg);
    } finally {
      setIsSubmittingReset(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>{settings.site_name}</h1>
        <h2>Login</h2>
        
        {sessionExpired && (
          <div className="session-expired-notice" role="alert">
            Your session has expired. Please log in again to continue.
          </div>
        )}
        
        <form onSubmit={handleSubmit}>
          <FormField
            label="Username"
            id="username"
            type="text"
            value={username}
            onChange={handleUsernameChange}
            onBlur={() => handleBlur('username')}
            error={touched.username ? errors.username : ''}
            success={touched.username && !errors.username && !!username}
            required
            autoComplete="username"
            helperText="Minimum 3 characters"
          />
          
          <PasswordField
            label="Password"
            id="password"
            value={password}
            onChange={handlePasswordChange}
            onBlur={() => handleBlur('password')}
            error={touched.password ? errors.password : ''}
            required
            autoComplete="current-password"
            helperText="Minimum 8 characters"
          />
          
          {error && <div className="error" role="alert">{error}</div>}
          
          <Button
            type="submit"
            variant="primary"
            size="large"
            fullWidth
            loading={isSubmitting}
            disabled={isSubmitting}
          >
            Login
          </Button>
        </form>
        
        <button
          onClick={() => setShowForgotPassword(true)}
          className="forgot-password-link"
          type="button"
        >
          Forgot Password?
        </button>
        
        <div className="login-notice">
          <p>This is an invite-only system. Contact an administrator to create an account.</p>
        </div>
      </div>

      {/* Forgot Password Modal */}
      <Modal
        isOpen={showForgotPassword}
        onClose={() => {
          setShowForgotPassword(false);
          setResetError('');
          setResetSuccess('');
          setResetEmail('');
        }}
        title="Reset Password"
        size="small"
      >
        <p className="modal-description">
          Enter your email address and we'll send you a link to reset your password.
        </p>
        <form onSubmit={handleForgotPassword}>
          <FormField
            label="Email"
            id="reset-email"
            type="email"
            value={resetEmail}
            onChange={setResetEmail}
            error={resetError}
            required
            placeholder="your-email@example.com"
            autoComplete="email"
          />
          {resetSuccess && (
            <div className="success" role="status">
              {resetSuccess}
            </div>
          )}
          <div className="modal__actions">
            <Button
              type="button"
              onClick={() => {
                setShowForgotPassword(false);
                setResetError('');
                setResetSuccess('');
                setResetEmail('');
              }}
              variant="secondary"
              disabled={isSubmittingReset}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={isSubmittingReset}
              disabled={isSubmittingReset}
            >
              Send Reset Link
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default Login;
