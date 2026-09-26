import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  CircularProgress,
  TextField,
  Typography,
} from '@mui/material';
import { Zap } from 'lucide-react';
import { prospectorApi } from '../api';

export const LoginView: React.FC = () => {
  const [secret, setSecret] = useState('sales_analizer_secret_key');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!secret.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const res = await prospectorApi.login(secret.trim());
      localStorage.setItem('salesanalizer_token', res.token);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Clau d\'administrador incorrecta');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        bgcolor: '#0f172a',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: 2,
      }}
    >
      <Card
        sx={{
          bgcolor: '#1e293b',
          border: '1px solid #334155',
          borderRadius: '12px',
          width: '100%',
          maxWidth: 420,
          p: { xs: 3, sm: 4 },
          boxShadow: '0 20px 25px -5px rgba(0,0,0,0.5)',
        }}
      >
        <Box
          sx={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 1,
            bgcolor: 'rgba(25, 118, 210, 0.15)',
            border: '1px solid rgba(25, 118, 210, 0.4)',
            px: 1.5,
            py: 0.5,
            borderRadius: '9999px',
            color: '#60a5fa',
            fontSize: '13px',
            fontWeight: 600,
            mb: 2,
          }}
        >
          <Zap size={14} /> SalesAnalizer Prospector
        </Box>

        <Typography variant="h5" sx={{ fontWeight: 700, color: '#f8fafc', mb: 1, letterSpacing: '-0.02em' }}>
          Accés intern
        </Typography>
        <Typography variant="body2" sx={{ color: '#94a3b8', mb: 3 }}>
          Introdueix la clau mestra d'administrador (<code>ADMIN_SECRET</code>) per accedir al radar.
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2.5, bgcolor: 'rgba(239, 68, 68, 0.15)', color: '#fca5a5' }}>
            {error}
          </Alert>
        )}

        <Box component="form" onSubmit={handleLogin}>
          <Box sx={{ mb: 3 }}>
            <Typography variant="caption" sx={{ color: '#94a3b8', fontWeight: 500, mb: 1, display: 'block' }}>
              Clau d'accés mestra
            </Typography>
            <TextField
              fullWidth
              type="password"
              placeholder="••••••••••••••••"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              disabled={loading}
              sx={{
                bgcolor: '#0f172a',
                borderRadius: '8px',
                '& .MuiOutlinedInput-root': {
                  color: '#fff',
                  '& fieldset': { borderColor: '#334155' },
                  '&:hover fieldset': { borderColor: '#475569' },
                  '&.Mui-focused fieldset': { borderColor: '#2563eb' },
                },
              }}
            />
          </Box>

          <Button
            fullWidth
            type="submit"
            variant="contained"
            disabled={loading}
            sx={{
              py: 1.4,
              bgcolor: '#2563eb',
              '&:hover': { bgcolor: '#1d4ed8' },
              fontWeight: 600,
              fontSize: '15px',
              textTransform: 'none',
              borderRadius: '8px',
            }}
          >
            {loading ? <CircularProgress size={22} color="inherit" /> : 'Entrar al Radar'}
          </Button>
        </Box>

        <Typography variant="caption" sx={{ color: '#64748b', textAlign: 'center', display: 'block', mt: 3 }}>
          Eina d'ús exclusivament intern • Cost d'operació 0€
        </Typography>
      </Card>
    </Box>
  );
};
