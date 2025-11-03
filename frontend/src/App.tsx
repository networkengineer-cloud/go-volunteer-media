import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ToastProvider } from './contexts/ToastContext';
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
              <AdminRoute>
                <AnimalForm />
              </AdminRoute>
            }
          />
          <Route
            path="/groups/:groupId/animals/:id"
            element={
              <AdminRoute>
                <AnimalForm />
              </AdminRoute>
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
            path="/admin/dashboard"
            element={
              <AdminRoute>
                <AdminDashboard />
              </AdminRoute>
            }
          />
          <Route
            path="/admin/users"
            element={
              <AdminRoute>
                <UsersPage />
              </AdminRoute>
            }
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
              <AdminRoute>
                <AdminAnimalTagsPage />
              </AdminRoute>
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
