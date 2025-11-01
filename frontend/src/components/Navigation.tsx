import React from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './Navigation.css';

const Navigation: React.FC = () => {
  const { user, logout, isAuthenticated } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = React.useState(false);

  // Theme state and persistence
  const [theme, setTheme] = React.useState<'light' | 'dark'>(() => {
    if (typeof window === 'undefined') return 'light';
    const stored = localStorage.getItem('theme');
    if (stored === 'light' || stored === 'dark') return stored;
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  });

  React.useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.setAttribute('data-theme', 'dark');
    } else {
      root.removeAttribute('data-theme');
    }
    try {
      localStorage.setItem('theme', theme);
    } catch {
      // ignore write errors (e.g., private mode)
    }
  }, [theme]);

  const toggleTheme = () => setTheme((t) => (t === 'dark' ? 'light' : 'dark'));
  const toggleMobileMenu = () => setMobileMenuOpen(!mobileMenuOpen);
  const closeMobileMenu = () => setMobileMenuOpen(false);

  return (
    <>
      {/* Skip link for accessibility */}
      <a href="#main-content" className="skip-link">
        Skip to main content
      </a>
      <nav className="navigation" role="navigation" aria-label="Main navigation">
        <div className="nav-container">
          <Link to={isAuthenticated ? '/dashboard' : '/'} className="nav-brand" onClick={closeMobileMenu}>
          {/* HAWS logo placeholder - using paw icon */}
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <circle cx="12" cy="4" r="2"></circle>
            <circle cx="19" cy="8" r="1.5"></circle>
            <circle cx="5" cy="8" r="1.5"></circle>
            <circle cx="17" cy="14" r="1.5"></circle>
            <circle cx="7" cy="14" r="1.5"></circle>
            <path d="M8.5 17.5 A 4 4 0 0 0 15.5 17.5"></path>
          </svg>
          <span className="nav-brand-text">HAWS Volunteer Portal</span>
        </Link>
        
        {/* Mobile menu toggle */}
        <button
          type="button"
          className="mobile-menu-toggle"
          aria-label="Toggle navigation menu"
          aria-expanded={mobileMenuOpen}
          onClick={toggleMobileMenu}
        >
          {mobileMenuOpen ? (
            // Close icon
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          ) : (
            // Hamburger icon
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="3" y1="12" x2="21" y2="12"></line>
              <line x1="3" y1="6" x2="21" y2="6"></line>
              <line x1="3" y1="18" x2="21" y2="18"></line>
            </svg>
          )}
        </button>

        <div className={`nav-right ${mobileMenuOpen ? 'mobile-menu-open' : ''}`}>
          <button
            type="button"
            className="theme-toggle"
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            aria-pressed={theme === 'dark'}
            onClick={toggleTheme}
          >
            {theme === 'dark' ? (
              // Sun icon
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="12" cy="12" r="4"></circle>
                <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
              </svg>
            ) : (
              // Moon icon
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
              </svg>
            )}
          </button>
          {!isAuthenticated ? (
            <Link to="/login" className="nav-login" onClick={closeMobileMenu}>Login</Link>
          ) : (
            <>
              {user?.is_admin && (
                <>
                  <Link to="/admin/users" className="nav-admin-users" onClick={closeMobileMenu}>Users</Link>
                  <Link to="/admin/groups" className="nav-admin-groups" onClick={closeMobileMenu}>Groups</Link>
                  <Link to="/admin/animals" className="nav-admin-animals" onClick={closeMobileMenu}>Animals</Link>
                  <Link to="/admin/site-settings" className="nav-admin-settings" onClick={closeMobileMenu}>Admin</Link>
                </>
              )}
              <Link to="/settings" className="nav-settings" onClick={closeMobileMenu}>My Settings</Link>
              <span className="nav-user" aria-label={`Logged in as ${user?.username}${user?.is_admin ? ', Admin' : ''}`}>
                {user?.username}
                {user?.is_admin && <span className="admin-badge" role="status">Admin</span>}
              </span>
              <button onClick={() => { logout(); closeMobileMenu(); }} className="nav-logout" aria-label="Logout">Logout</button>
            </>
          )}
        </div>
      </div>
      </nav>
    </>
  );
};

export default Navigation;
