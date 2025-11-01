import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import axios from 'axios';
import FormField from '../components/FormField';
import PasswordField from '../components/PasswordField';
import Button from '../components/Button';
import './Login.css';

const SUCCESS_MESSAGE_TIMEOUT = 3000; // milliseconds

const Login: React.FC = () => {
  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showForgotPassword, setShowForgotPassword] = useState(false);
  const [resetEmail, setResetEmail] = useState('');
  const [resetSuccess, setResetSuccess] = useState('');
  const [resetError, setResetError] = useState('');
  const [isSubmittingReset, setIsSubmittingReset] = useState(false);
  
  // Form validation states
  const [touched, setTouched] = useState({
    username: false,
    email: false,
    password: false,
  });
  const [errors, setErrors] = useState({
    username: '',
    email: '',
    password: '',
  });
  
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const toast = useToast();

  // Validation functions
  const validateUsername = (value: string): string => {
    if (!value) return 'Username is required';
    if (value.length < 3) return 'Username must be at least 3 characters';
    return '';
  };

  const validateEmail = (value: string): string => {
    if (!value) return 'Email is required';
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(value)) return 'Please enter a valid email address';
    return '';
  };

  const validatePassword = (value: string): string => {
    if (!value) return 'Password is required';
    if (value.length < 8) return 'Password must be at least 8 characters';
    return '';
  };

  // Handle field blur for validation
  const handleBlur = (field: 'username' | 'email' | 'password') => {
    setTouched({ ...touched, [field]: true });
    
    let error = '';
    if (field === 'username') {
      error = validateUsername(username);
    } else if (field === 'email') {
      error = validateEmail(email);
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

  const handleEmailChange = (value: string) => {
    setEmail(value);
    if (touched.email) {
      setErrors({ ...errors, email: validateEmail(value) });
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
    const emailError = !isLogin ? validateEmail(email) : '';
    const passwordError = validatePassword(password);

    if (usernameError || emailError || passwordError) {
      setErrors({
        username: usernameError,
        email: emailError,
        password: passwordError,
      });
      setTouched({
        username: true,
        email: !isLogin,
        password: true,
      });
      setIsSubmitting(false);
      return;
    }

    try {
      if (isLogin) {
        await login(username, password);
        toast.showSuccess('Successfully logged in!');
      } else {
        await register(username, email, password);
        toast.showSuccess('Account created successfully!');
      }
      navigate('/');
    } catch (err: any) {
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

    // Validate email
    const emailError = validateEmail(resetEmail);
    if (emailError) {
      setResetError(emailError);
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
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || 'Failed to send reset email';
      setResetError(errorMsg);
      toast.showError(errorMsg);
    } finally {
      setIsSubmittingReset(false);
    }
  };

  const handleModeSwitch = () => {
    setIsLogin(!isLogin);
    setError('');
    setErrors({ username: '', email: '', password: '' });
    setTouched({ username: false, email: false, password: false });
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>Haws Volunteers</h1>
        <h2>{isLogin ? 'Login' : 'Register'}</h2>
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
          
          {!isLogin && (
            <FormField
              label="Email"
              id="email"
              type="email"
              value={email}
              onChange={handleEmailChange}
              onBlur={() => handleBlur('email')}
              error={touched.email ? errors.email : ''}
              success={touched.email && !errors.email && !!email}
              required
              autoComplete="email"
              placeholder="your-email@example.com"
            />
          )}
          
          <PasswordField
            label="Password"
            id="password"
            value={password}
            onChange={handlePasswordChange}
            onBlur={() => handleBlur('password')}
            error={touched.password ? errors.password : ''}
            required
            autoComplete={isLogin ? 'current-password' : 'new-password'}
            showStrengthIndicator={!isLogin}
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
            {isLogin ? 'Login' : 'Register'}
          </Button>
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
        
        <Button
          onClick={handleModeSwitch}
          variant="secondary"
          size="medium"
          fullWidth
        >
          {isLogin ? 'Need an account? Register' : 'Have an account? Login'}
        </Button>
      </div>

      {/* Forgot Password Modal */}
      {showForgotPassword && (
        <div className="modal-overlay" onClick={() => setShowForgotPassword(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h2>Reset Password</h2>
            <p>Enter your email address and we'll send you a link to reset your password.</p>
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
              {resetSuccess && <div className="success" role="status">{resetSuccess}</div>}
              <div className="modal-actions">
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
          </div>
        </div>
      )}
    </div>
  );
};

export default Login;
