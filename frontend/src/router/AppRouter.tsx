import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { LoginView } from '../modules/prospector/views/LoginView';
import { RadarView } from '../modules/prospector/views/RadarView';

const PrivateRoute: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const token = localStorage.getItem('salesanalizer_token');
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return children;
};

export const AppRouter: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginView />} />
        <Route
          path="/"
          element={
            <PrivateRoute>
              <RadarView />
            </PrivateRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
};
