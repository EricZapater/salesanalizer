import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  CircularProgress,
  IconButton,
  Snackbar,
  Typography,
  Alert,
} from '@mui/material';
import { RotateCw, LogOut, Zap } from 'lucide-react';
import { useProspectorStore } from '../store';
import { IngestCard } from '../components/IngestCard';
import { RadarTable } from '../components/RadarTable';
import { LateralDrawer } from '../components/LateralDrawer';
import { JobOffer } from '../types';

export const RadarView: React.FC = () => {
  const navigate = useNavigate();
  const [drawerOpen, setDrawerOpen] = useState(false);

  const {
    offers,
    selectedOffer,
    systemStatus,
    isScraping,
    toastMessage,
    fetchOffers,
    fetchSystemStatus,
    runScrapers,
    selectOffer,
    loadOfferDetail,
    setToast,
  } = useProspectorStore();

  useEffect(() => {
    fetchOffers();
    fetchSystemStatus();
  }, []);

  const handleSelectOffer = async (offer: JobOffer) => {
    selectOffer(offer);
    setDrawerOpen(true);
    // Load full text if missing
    if (!offer.raw_text) {
      await loadOfferDetail(offer.id);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('salesanalizer_token');
    navigate('/login');
  };

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#090d16', color: '#f9fafb', display: 'flex', flexDirection: 'column' }}>
      {/* Top Header */}
      <Box
        component="header"
        sx={{
          bgcolor: '#111827',
          borderBottom: '1px solid #374151',
          height: 64,
          px: 3,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          position: 'sticky',
          top: 0,
          zIndex: 40,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box
            sx={{
              background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
              color: '#fff',
              px: 1.2,
              py: 0.5,
              borderRadius: '6px',
              fontSize: '12px',
              fontWeight: 700,
              display: 'flex',
              alignItems: 'center',
              gap: 0.5,
            }}
          >
            <Zap size={14} /> SalesAnalizer
          </Box>
          <Typography variant="h6" sx={{ fontWeight: 700, fontSize: '17px', letterSpacing: '-0.02em' }}>
            Radar d'Oportunitats Micro-SaaS
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {/* Scrape button */}
          <Button
            variant="outlined"
            size="small"
            onClick={runScrapers}
            disabled={isScraping}
            startIcon={isScraping ? <CircularProgress size={14} color="inherit" /> : <RotateCw size={14} />}
            sx={{
              bgcolor: '#1f2937',
              borderColor: '#374151',
              color: '#93c5fd',
              textTransform: 'none',
              fontWeight: 600,
              fontSize: '13px',
              borderRadius: '6px',
              '&:hover': {
                bgcolor: 'rgba(37, 99, 235, 0.15)',
                borderColor: '#3b82f6',
              },
            }}
          >
            {isScraping ? 'Rastrejant portals...' : 'Rastrejar portals ara'}
          </Button>

          {/* Quota Badge */}
          <Box
            sx={{
              bgcolor: '#1f2937',
              border: '1px solid #374151',
              px: 1.8,
              py: 0.7,
              borderRadius: '9999px',
              fontSize: '13px',
              display: 'flex',
              alignItems: 'center',
              gap: 1,
            }}
          >
            <Box
              sx={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                bgcolor: '#10b981',
                boxShadow: '0 0 8px #10b981',
              }}
            />
            <Typography variant="body2" sx={{ fontSize: '13px', color: '#9ca3af' }}>
              Consum d'avui: <strong style={{ color: '#fff' }}>{systemStatus?.signals_today ?? offers.length} / {systemStatus?.daily_limit ?? 50}</strong> senyals
            </Typography>
          </Box>

          {/* Logout */}
          <IconButton onClick={handleLogout} size="small" sx={{ color: '#9ca3af', '&:hover': { color: '#ef4444' } }}>
            <LogOut size={18} />
          </IconButton>
        </Box>
      </Box>

      {/* Main container */}
      <Box
        component="main"
        sx={{
          flex: 1,
          p: { xs: 2, md: 3 },
          maxWidth: 1440,
          margin: '0 auto',
          width: '100%',
        }}
      >
        <IngestCard />

        {/* Section Title */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', mb: 2 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, letterSpacing: '-0.01em', color: '#f9fafb' }}>
              El Radar (Prioritzat per Viabilitat PLG)
            </Typography>
            <Typography variant="body2" sx={{ color: '#9ca3af', fontSize: '13px' }}>
              Les millors oportunitats de Micro-SaaS (Score 5 i 4) amb self-onboarding autònom.
            </Typography>
          </Box>
        </Box>

        <RadarTable onSelectOffer={handleSelectOffer} />
      </Box>

      {/* Drawer */}
      <LateralDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        offer={selectedOffer}
      />

      {/* Toast Notification */}
      <Snackbar
        open={Boolean(toastMessage)}
        autoHideDuration={3500}
        onClose={() => setToast(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Alert
          onClose={() => setToast(null)}
          severity="success"
          sx={{
            bgcolor: '#1e293b',
            color: '#f8fafc',
            border: '1px solid #10b981',
            '& .MuiAlert-icon': { color: '#10b981' },
          }}
        >
          {toastMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};
