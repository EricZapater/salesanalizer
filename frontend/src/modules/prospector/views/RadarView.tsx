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
  LinearProgress,
  Chip,
} from '@mui/material';
import { RotateCw, LogOut, Zap, Sparkles, Filter, Clock } from 'lucide-react';
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
    statusFilter,
    isScraping,
    progressState,
    toastMessage,
    fetchOffers,
    setStatusFilter,
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
    if (!offer.raw_text) {
      await loadOfferDetail(offer.id);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('salesanalizer_token');
    navigate('/login');
  };

  const filterTabs = [
    { label: 'Actius (Tots)', value: 'active', color: '#38bdf8' },
    { label: 'Pendents', value: 'pendent', color: '#60a5fa' },
    { label: 'Enviades', value: 'enviada', color: '#c084fc' },
    { label: 'Acceptades', value: 'acceptada', color: '#34d399' },
    { label: 'Rebutjades', value: 'rebutjada', color: '#f87171' },
    { label: 'Descartades', value: 'descartada', color: '#9ca3af' },
    { label: 'Històric Total', value: 'all', color: '#e2e8f0' },
  ];

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
              bgcolor: isScraping ? 'rgba(37, 99, 235, 0.2)' : '#1f2937',
              borderColor: isScraping ? '#3b82f6' : '#374151',
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
            {isScraping ? 'Rastrejant en paral·lel...' : 'Rastrejar portals ara'}
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

        {/* Live Parallel Scraping Progress & Entertainment Banner */}
        {progressState && (
          <Box
            sx={{
              mb: 3,
              p: 2.5,
              borderRadius: '12px',
              background: 'linear-gradient(135deg, rgba(30, 41, 59, 0.95), rgba(15, 23, 42, 0.95))',
              border: '1px solid rgba(59, 130, 246, 0.4)',
              boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
              position: 'relative',
              overflow: 'hidden',
            }}
          >
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <CircularProgress size={18} sx={{ color: '#60a5fa' }} />
                <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#93c5fd' }}>
                  {progressState.message}
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                {progressState.estimated_secs !== undefined && progressState.estimated_secs > 0 && (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: '#9ca3af', fontSize: '12px' }}>
                    <Clock size={14} /> ~{progressState.estimated_secs}s restants
                  </Box>
                )}
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#34d399', fontSize: '13px' }}>
                  {progressState.progress}%
                </Typography>
              </Box>
            </Box>

            <LinearProgress
              variant="determinate"
              value={progressState.progress}
              sx={{
                height: 8,
                borderRadius: 4,
                bgcolor: 'rgba(255,255,255,0.08)',
                '& .MuiLinearProgress-bar': {
                  background: 'linear-gradient(90deg, #3b82f6, #10b981)',
                  borderRadius: 4,
                },
                mb: 2,
              }}
            />

            {/* Fun Fact / Joke Card */}
            {(progressState.fun_fact || progressState.joke) && (
              <Box
                sx={{
                  p: 1.5,
                  borderRadius: '8px',
                  bgcolor: 'rgba(59, 130, 246, 0.08)',
                  border: '1px dashed rgba(59, 130, 246, 0.3)',
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 1.5,
                }}
              >
                <Sparkles size={18} color="#fbbf24" style={{ flexShrink: 0, marginTop: 2 }} />
                <Typography variant="body2" sx={{ color: '#cbd5e1', fontSize: '13px', lineHeight: 1.4 }}>
                  {progressState.fun_fact || progressState.joke}
                </Typography>
              </Box>
            )}
          </Box>
        )}

        {/* Section Title & Status Filters */}
        <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, justifyContent: 'space-between', alignItems: { xs: 'flex-start', md: 'center' }, gap: 2, mb: 2.5 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, letterSpacing: '-0.01em', color: '#f9fafb' }}>
              El Radar (Prioritzat per Viabilitat PLG)
            </Typography>
            <Typography variant="body2" sx={{ color: '#9ca3af', fontSize: '13px' }}>
              Gestiona el pipeline d'oportunitats segons el seu estat de prospecció.
            </Typography>
          </Box>

          {/* Status Filter Bar */}
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.8, alignItems: 'center' }}>
            <Filter size={15} color="#9ca3af" style={{ marginRight: 4 }} />
            {filterTabs.map((tab) => {
              const isActive = statusFilter === tab.value;
              return (
                <Chip
                  key={tab.value}
                  label={tab.label}
                  size="small"
                  clickable
                  onClick={() => setStatusFilter(tab.value)}
                  sx={{
                    fontWeight: 700,
                    fontSize: '12px',
                    px: 0.5,
                    bgcolor: isActive ? tab.color : '#1f2937',
                    color: isActive ? '#090d16' : '#9ca3af',
                    border: `1px solid ${isActive ? tab.color : '#374151'}`,
                    '&:hover': {
                      bgcolor: isActive ? tab.color : 'rgba(255,255,255,0.06)',
                      borderColor: tab.color,
                    },
                  }}
                />
              );
            })}
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
