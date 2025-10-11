import React from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './Navigation.css';

const Navigation: React.FC = () => {
  const { user, logout, isAuthenticated } = useAuth();

  return (
    <nav className="navigation">
      <div className="nav-container">
        <Link to={isAuthenticated ? '/dashboard' : '/'} className="nav-brand">
          Haws Volunteers
        </Link>
        <div className="nav-right">
          {!isAuthenticated ? (
            <Link to="/login" className="nav-login">Login</Link>
          ) : (
            <>
              <span className="nav-user">
                {user?.username}
                {user?.is_admin && <span className="admin-badge">Admin</span>}
              </span>
              <button onClick={logout} className="nav-logout">Logout</button>
            </>
          )}
        </div>
      </div>
    </nav>
  );
};

export default Navigation;
