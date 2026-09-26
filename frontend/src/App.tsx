import React from 'react';
import { ThemeProvider, createTheme, CssBaseline } from '@mui/material';
import { AppRouter } from './router/AppRouter';

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#2563eb',
      dark: '#1d4ed8',
      light: '#60a5fa',
    },
    background: {
      default: '#090d16',
      paper: '#111827',
    },
    text: {
      primary: '#f9fafb',
      secondary: '#9ca3af',
    },
  },
  typography: {
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, sans-serif",
  },
  shape: {
    borderRadius: 8,
  },
});

export const App: React.FC = () => {
  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />
      <AppRouter />
    </ThemeProvider>
  );
};
