import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { useAuth } from './hooks/useAuth';
import { ToastProvider } from './contexts/ToastContext';
import { groupsApi } from './api/client';
import Navigation from './components/Navigation';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import GroupPage from './pages/GroupPage';
import AnimalForm from './pages/AnimalForm';
import AnimalDetailPage from './pages/AnimalDetailPage';
import PhotoGallery from './pages/PhotoGallery';
import UpdateForm from './pages/UpdateForm';
import Home from './pages/Home';
import UsersPage from './pages/UsersPage';
import AdminSettingsPage from './pages/AdminSettingsPage';
import AdminAnimalTagsPage from './pages/AdminAnimalTagsPage';
import GroupsPage from './pages/GroupsPage';
import BulkEditAnimalsPage from './pages/BulkEditAnimalsPage';
import ResetPassword from './pages/ResetPassword';
import Settings from './pages/Settings';
import UserProfilePage from './pages/UserProfilePage';
import AdminDashboard from './pages/AdminDashboard';
import './App.css';

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};


const PublicRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return !isAuthenticated ? <>{children}</> : <Navigate to="/dashboard" />;
};

const AdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isAdmin } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" />;
  if (!isAdmin) return <Navigate to="/dashboard" />;
  return <>{children}</>;
};

// GroupAdminRoute - allows access if user is site admin OR group admin for the specific group
// The groupId is extracted from URL params (supports both :id and :groupId patterns)
const GroupAdminRouteInner: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isAdmin } = useAuth();
  const params = useParams();
  const groupId = params.groupId || params.id;
  const [hasAccess, setHasAccess] = useState<boolean | null>(null);

  useEffect(() => {
    // Site admins have access to everything
    if (isAdmin) {
      setHasAccess(true);
      return;
    }

    // Check if user is group admin for this group
    if (groupId) {
      groupsApi.getMembership(parseInt(groupId))
        .then(response => {
          setHasAccess(response.data.is_group_admin || response.data.is_site_admin);
        })
        .catch(() => {
          setHasAccess(false);
        });
    } else {
      setHasAccess(false);
    }
  }, [groupId, isAdmin]);

  if (!isAuthenticated) return <Navigate to="/login" />;
  if (hasAccess === null) return <div className="loading-spinner">Loading...</div>;
  if (!hasAccess) return <Navigate to="/dashboard" />;
  return <>{children}</>;
};

const GroupAdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return <GroupAdminRouteInner>{children}</GroupAdminRouteInner>;
};

// UsersRoute - allows access if user is site admin or group admin
const UsersRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isAdmin, user } = useAuth();
  // Check if user is a group admin (has group membership or is admin)
  const isGroupAdmin = isAuthenticated && user ? (user.groups?.length ?? 0) > 0 : false;

  if (!isAuthenticated) return <Navigate to="/login" />;
  if (!isAdmin && !isGroupAdmin) return <Navigate to="/dashboard" />;
  return <>{children}</>;
};

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          <Navigation />
          <main id="main-content" role="main">
            <Routes>
          <Route
            path="/"
            element={
              <PublicRoute>
                <Home />
              </PublicRoute>
            }
          />
          <Route
            path="/login"
            element={
              <PublicRoute>
                <Login />
              </PublicRoute>
            }
          />
          <Route
            path="/reset-password"
            element={
              <PublicRoute>
                <ResetPassword />
              </PublicRoute>
            }
          />
          <Route
            path="/dashboard"
            element={
              <PrivateRoute>
                <Dashboard />
              </PrivateRoute>
            }
          />
          <Route
            path="/groups/:id"
            element={
              <PrivateRoute>
                <GroupPage />
              </PrivateRoute>
            }
          />
          <Route
            path="/groups/:groupId/animals/new"
            element={
              <GroupAdminRoute>
                <AnimalForm />
              </GroupAdminRoute>
            }
          />
          <Route
            path="/groups/:groupId/animals/:id"
            element={
              <GroupAdminRoute>
                <AnimalForm />
              </GroupAdminRoute>
            }
          />
          <Route
            path="/groups/:groupId/animals/:id/view"
            element={
              <PrivateRoute>
                <AnimalDetailPage />
              </PrivateRoute>
            }
          />
          <Route
            path="/groups/:groupId/animals/:id/photos"
            element={
              <PrivateRoute>
                <PhotoGallery />
              </PrivateRoute>
            }
          />
          <Route
            path="/groups/:groupId/updates/new"
            element={
              <PrivateRoute>
                <UpdateForm />
              </PrivateRoute>
            }
          />
          <Route
            path="/settings"
            element={
              <PrivateRoute>
                <Settings />
              </PrivateRoute>
            }
          />
          <Route
            path="/users/:id/profile"
            element={
              <PrivateRoute>
                <UserProfilePage />
              </PrivateRoute>
            }
          />
          <Route
            path="/users"
            element={
              <UsersRoute>
                <UsersPage />
              </UsersRoute>
            }
          />
          <Route
            path="/admin/dashboard"
            element={
              <AdminRoute>
                <AdminDashboard />
              </AdminRoute>
            }
          />
          <Route
            path="/admin/users"
            element={<Navigate to="/users" replace />}
          />
          <Route
            path="/admin/groups"
            element={
              <AdminRoute>
                <GroupsPage />
              </AdminRoute>
            }
          />
          <Route
            path="/admin/site-settings"
            element={
              <AdminRoute>
                <AdminSettingsPage />
              </AdminRoute>
            }
          />
          <Route
            path="/admin/animal-tags"
            element={
              <PrivateRoute>
                <AdminAnimalTagsPage />
              </PrivateRoute>
            }
          />
          <Route
            path="/admin/animals"
            element={
              <AdminRoute>
                <BulkEditAnimalsPage />
              </AdminRoute>
            }
          />
        </Routes>
        </main>
        </ToastProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
