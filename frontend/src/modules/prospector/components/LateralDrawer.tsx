import React, { useState } from 'react';
import {
  Box,
  Button,
  Chip,
  Drawer,
  IconButton,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import { X, Copy, ExternalLink, Check, AlertTriangle, Rocket, Users, MessageSquare, Tag } from 'lucide-react';
import { useProspectorStore } from '../store';
import { JobOffer } from '../types';

interface LateralDrawerProps {
  open: boolean;
  onClose: () => void;
  offer: JobOffer | null;
}

export const LateralDrawer: React.FC<LateralDrawerProps> = ({ open, onClose, offer }) => {
  const [currentTab, setCurrentTab] = useState(0);
  const [copied, setCopied] = useState(false);
  const { systemStatus, setToast, updateOfferStatus } = useProspectorStore();

  const handleCopyHook = () => {
    if (offer?.analysis?.ganxo_venda) {
      navigator.clipboard.writeText(offer.analysis.ganxo_venda);
      setCopied(true);
      setToast('Ganxo de venda copiat al portapapers!');
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const statusOptions = [
    { label: 'Pendent', value: 'pendent', color: '#60a5fa', activeBg: '#2563eb' },
    { label: 'Enviada', value: 'enviada', color: '#c084fc', activeBg: '#7e22ce' },
    { label: 'Acceptada', value: 'acceptada', color: '#34d399', activeBg: '#059669' },
    { label: 'Rebutjada', value: 'rebutjada', color: '#f87171', activeBg: '#dc2626' },
    { label: 'Descartada', value: 'descartada', color: '#9ca3af', activeBg: '#4b5563' },
  ];

  const currentOfferStatus = (offer?.status || 'pendent').toLowerCase();

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      variant="temporary"
      PaperProps={{
        sx: {
          width: { xs: '100%', sm: 480 },
          bgcolor: '#111827',
          color: '#f9fafb',
          borderLeft: '1px solid #374151',
          display: 'flex',
          flexDirection: 'column',
        },
      }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          bgcolor: '#151d2c',
          borderBottom: '1px solid #374151',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Box>
          <Typography variant="caption" sx={{ textTransform: 'uppercase', color: '#9ca3af', fontWeight: 700 }}>
            Detall de l'Oportunitat
          </Typography>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#fff' }}>
            {offer?.analysis?.proposta_micro_saas?.split(':')[0] || offer?.title || 'Detalls'}
          </Typography>
        </Box>
        <IconButton onClick={onClose} size="small" sx={{ color: '#9ca3af' }}>
          <X size={20} />
        </IconButton>
      </Box>

      {/* Tabs */}
      <Tabs
        value={currentTab}
        onChange={(_, val) => setCurrentTab(val)}
        sx={{
          bgcolor: '#090d16',
          borderBottom: '1px solid #374151',
          '& .MuiTab-root': {
            color: '#9ca3af',
            textTransform: 'none',
            fontWeight: 600,
            fontSize: '13px',
            '&.Mui-selected': { color: '#60a5fa' },
          },
          '& .MuiTabs-indicator': { backgroundColor: '#2563eb' },
        }}
      >
        <Tab label="💡 Anàlisi Groq" />
        <Tab label="⚙️ Estat del Sistema" />
      </Tabs>

      {/* Content */}
      <Box sx={{ p: 3, flex: 1, overflowY: 'auto' }}>
        {currentTab === 0 && offer && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
            {/* Score Banner */}
            <Box
              sx={{
                p: 2,
                borderRadius: '10px',
                bgcolor: 'rgba(37, 99, 235, 0.1)',
                border: '1px solid rgba(37, 99, 235, 0.3)',
                display: 'flex',
                alignItems: 'center',
                gap: 2,
              }}
            >
              <Box
                sx={{
                  width: 44,
                  height: 44,
                  borderRadius: '8px',
                  bgcolor: '#10b981',
                  color: '#fff',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 800,
                  fontSize: '20px',
                }}
              >
                {offer.analysis?.viabilitat_plg_score || 5}
              </Box>
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#34d399' }}>
                  Viabilitat PLG {offer.analysis?.viabilitat_plg_score || 5}/5
                </Typography>
                <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                  {offer.analysis?.viabilitat_plg_score === 5
                    ? '100% Self-Onboarding sense integració tècnica.'
                    : 'Potencial alt de micro-SaaS.'}
                </Typography>
              </Box>
            </Box>

            {/* Status Selector */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
                <Tag size={15} color="#60a5fa" />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase' }}>
                  Estat de l'Oportunitat
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {statusOptions.map((st) => {
                  const isCurrent = currentOfferStatus === st.value || (st.value === 'pendent' && currentOfferStatus === 'analyzed');
                  return (
                    <Button
                      key={st.value}
                      size="small"
                      variant={isCurrent ? 'contained' : 'outlined'}
                      onClick={() => updateOfferStatus(offer.id, st.value)}
                      sx={{
                        fontSize: '11px',
                        fontWeight: 700,
                        textTransform: 'none',
                        borderRadius: '6px',
                        py: 0.5,
                        px: 1.2,
                        bgcolor: isCurrent ? st.activeBg : 'transparent',
                        borderColor: isCurrent ? st.activeBg : '#374151',
                        color: isCurrent ? '#fff' : st.color,
                        '&:hover': {
                          bgcolor: isCurrent ? st.activeBg : 'rgba(255,255,255,0.05)',
                          borderColor: st.activeBg,
                        },
                      }}
                    >
                      {isCurrent ? `✓ ${st.label}` : st.label}
                    </Button>
                  );
                })}
              </Box>
            </Box>

            {/* Inefficiency */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <AlertTriangle size={15} color="#f59e0b" />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase' }}>
                  Ineficiència Manual Detectada
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ color: '#f1f5f9', lineHeight: 1.5 }}>
                {offer.analysis?.ineficiencia_manual || 'Sense descripció.'}
              </Typography>
            </Box>

            {/* Micro-SaaS Solution */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <Rocket size={15} color="#3b82f6" />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase' }}>
                  Proposta Micro-SaaS (&lt; 100€/mes)
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ color: '#f1f5f9', lineHeight: 1.5 }}>
                {offer.analysis?.proposta_micro_saas || '-'}
              </Typography>
            </Box>

            {/* Buyer Persona */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <Users size={15} color="#a855f7" />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase' }}>
                  Decisor de Compra
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ color: '#60a5fa', fontWeight: 600 }}>
                {offer.analysis?.decisor_compra || '-'}
              </Typography>
            </Box>

            {/* Cold Hook */}
            <Box
              sx={{
                p: 2,
                borderRadius: '10px',
                bgcolor: '#0f172a',
                border: '1px solid #3b82f6',
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <MessageSquare size={15} color="#3b82f6" />
                <Typography variant="caption" sx={{ fontWeight: 700, color: '#93c5fd', textTransform: 'uppercase' }}>
                  Ganxo de Venda (Cold Outreach)
                </Typography>
              </Box>
              <Typography
                variant="body2"
                sx={{
                  fontStyle: 'italic',
                  color: '#bfdbfe',
                  mb: 1.5,
                  lineHeight: 1.5,
                }}
              >
                "{offer.analysis?.ganxo_venda || '-'}"
              </Typography>
              <Button
                variant="contained"
                size="small"
                onClick={handleCopyHook}
                startIcon={copied ? <Check size={14} /> : <Copy size={14} />}
                sx={{
                  bgcolor: copied ? '#10b981' : '#2563eb',
                  '&:hover': { bgcolor: copied ? '#059669' : '#1d4ed8' },
                  textTransform: 'none',
                  fontSize: '12px',
                  fontWeight: 600,
                  borderRadius: '6px',
                }}
              >
                {copied ? 'Copiat!' : 'Copiar Ganxo'}
              </Button>
            </Box>

            {/* Original Text */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase', mb: 1, display: 'block' }}>
                Text de l'Oferta Original
              </Typography>
              <Box
                sx={{
                  p: 1.5,
                  bgcolor: '#0b0f19',
                  borderRadius: '6px',
                  maxHeight: 140,
                  overflowY: 'auto',
                  fontSize: '12px',
                  color: '#9ca3af',
                  lineHeight: 1.5,
                }}
              >
                {offer.raw_text || 'Text no disponible'}
              </Box>
              <Box sx={{ mt: 1.5 }}>
                <Button
                  component="a"
                  href={offer.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  size="small"
                  endIcon={<ExternalLink size={13} />}
                  sx={{ color: '#60a5fa', textTransform: 'none', p: 0, fontSize: '12px' }}
                >
                  Veure oferta original a la font
                </Button>
              </Box>
            </Box>
          </Box>
        )}

        {currentTab === 1 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
            {/* Usage quota */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
                📊 Consum de l'API & Quota Diària
              </Typography>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 1, borderBottom: '1px solid #374151' }}>
                <Typography variant="body2" sx={{ color: '#9ca3af' }}>Senyals consumits avui:</Typography>
                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                  {systemStatus?.signals_today ?? 0} / {systemStatus?.daily_limit ?? 50}
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 1, borderBottom: '1px solid #374151' }}>
                <Typography variant="body2" sx={{ color: '#9ca3af' }}>Límit restant:</Typography>
                <Typography variant="body2" sx={{ fontWeight: 700, color: '#34d399' }}>
                  {Math.max(0, (systemStatus?.daily_limit ?? 50) - (systemStatus?.signals_today ?? 0))} senyals
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 1 }}>
                <Typography variant="body2" sx={{ color: '#9ca3af' }}>Cost d'operació:</Typography>
                <Typography variant="body2" sx={{ fontWeight: 700, color: '#34d399' }}>
                  {systemStatus?.cost_eur?.toFixed(2) ?? '0.00'} €
                </Typography>
              </Box>
            </Box>

            {/* Scraper Status */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
                🤖 Estat dels Rastrejadors (Scrapers)
              </Typography>
              {systemStatus?.scrapers?.map((scraper, idx) => (
                <Box key={idx} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1, borderBottom: idx < systemStatus.scrapers.length - 1 ? '1px solid #374151' : 'none' }}>
                  <Typography variant="body2" sx={{ color: '#9ca3af' }}>{scraper.source}:</Typography>
                  <Chip
                    label={`${scraper.status.toUpperCase()} (${scraper.items_found} ofertes)`}
                    size="small"
                    sx={{
                      bgcolor: scraper.status === 'ok' ? 'rgba(16, 185, 129, 0.15)' : 'rgba(239, 68, 68, 0.15)',
                      color: scraper.status === 'ok' ? '#34d399' : '#f87171',
                      fontWeight: 600,
                      fontSize: '11px',
                    }}
                  />
                </Box>
              ))}
            </Box>

            {/* Groq Connection */}
            <Box sx={{ p: 2, borderRadius: '10px', bgcolor: '#1f2937', border: '1px solid #374151' }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#9ca3af', textTransform: 'uppercase', mb: 1.5, display: 'block' }}>
                ⚡ Connexió Groq IA
              </Typography>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 1, borderBottom: '1px solid #374151' }}>
                <Typography variant="body2" sx={{ color: '#9ca3af' }}>Model actiu:</Typography>
                <Typography variant="body2" sx={{ color: '#cbd5e1' }}>{systemStatus?.llm_status?.model || 'llama-3.3-70b-versatile'}</Typography>
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 1 }}>
                <Typography variant="body2" sx={{ color: '#9ca3af' }}>Estat de l'API:</Typography>
                <Chip
                  label={systemStatus?.llm_status?.connected ? 'Operatiu' : 'Mode Rescat / Offline'}
                  size="small"
                  sx={{
                    bgcolor: systemStatus?.llm_status?.connected ? 'rgba(16, 185, 129, 0.15)' : 'rgba(245, 158, 11, 0.15)',
                    color: systemStatus?.llm_status?.connected ? '#34d399' : '#fbbf24',
                    fontWeight: 600,
                    fontSize: '11px',
                  }}
                />
              </Box>
            </Box>
          </Box>
        )}
      </Box>
    </Drawer>
  );
};
