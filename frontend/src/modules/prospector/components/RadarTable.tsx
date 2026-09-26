import React from 'react';
import {
  Box,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Chip,
  Tooltip,
} from '@mui/material';
import { Trash2 } from 'lucide-react';
import { useProspectorStore } from '../store';
import { JobOffer } from '../types';

interface RadarTableProps {
  onSelectOffer: (offer: JobOffer) => void;
}

export const RadarTable: React.FC<RadarTableProps> = ({ onSelectOffer }) => {
  const { offers, selectedOffer, discardOffer, isLoading } = useProspectorStore();

  const getScoreColor = (score?: number) => {
    switch (score) {
      case 5:
        return { bg: 'rgba(16, 185, 129, 0.15)', text: '#34d399', border: 'rgba(16, 185, 129, 0.4)' };
      case 4:
        return { bg: 'rgba(59, 130, 246, 0.15)', text: '#60a5fa', border: 'rgba(59, 130, 246, 0.4)' };
      case 3:
        return { bg: 'rgba(245, 158, 11, 0.15)', text: '#fbbf24', border: 'rgba(245, 158, 11, 0.4)' };
      case 2:
        return { bg: 'rgba(249, 115, 22, 0.15)', text: '#fb923c', border: 'rgba(249, 115, 22, 0.4)' };
      default:
        return { bg: 'rgba(239, 68, 68, 0.15)', text: '#f87171', border: 'rgba(239, 68, 68, 0.4)' };
    }
  };

  const getSourceBadge = (source: string) => {
    const s = (source || '').toLowerCase();
    if (s.includes('soc') || s.includes('feina_activa') || s.includes('feina activa')) {
      return <Chip label="SOC" size="small" sx={{ bgcolor: '#1e3a8a', color: '#93c5fd', fontWeight: 600, fontSize: 11 }} />;
    }
    if (s.includes('reddit')) {
      return <Chip label="Reddit" size="small" sx={{ bgcolor: '#7c2d12', color: '#fdba74', fontWeight: 600, fontSize: 11 }} />;
    }
    if (s.includes('searxng') || s.includes('dork')) {
      return <Chip label="SearXNG Dork" size="small" sx={{ bgcolor: '#0f2942', color: '#38bdf8', fontWeight: 600, fontSize: 11, border: '1px solid #0284c7' }} />;
    }
    if (s.includes('google')) {
      return <Chip label="Google Dork" size="small" sx={{ bgcolor: '#1e293b', color: '#38bdf8', fontWeight: 600, fontSize: 11, border: '1px solid #0284c7' }} />;
    }
    if (s.includes('infofeina')) {
      return <Chip label="Infofeina" size="small" sx={{ bgcolor: '#581c87', color: '#d8b4fe', fontWeight: 600, fontSize: 11 }} />;
    }
    return <Chip label={source || 'Manual'} size="small" sx={{ bgcolor: '#065f46', color: '#6ee7b7', fontWeight: 600, fontSize: 11 }} />;
  };

  const getStatusBadge = (status: string) => {
    const s = (status || 'pendent').toLowerCase();
    switch (s) {
      case 'acceptada':
        return <Chip label="Acceptada" size="small" sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', color: '#34d399', fontWeight: 700, fontSize: 11, border: '1px solid rgba(16, 185, 129, 0.4)' }} />;
      case 'enviada':
        return <Chip label="Enviada" size="small" sx={{ bgcolor: 'rgba(147, 51, 234, 0.2)', color: '#c084fc', fontWeight: 700, fontSize: 11, border: '1px solid rgba(147, 51, 234, 0.4)' }} />;
      case 'rebutjada':
        return <Chip label="Rebutjada" size="small" sx={{ bgcolor: 'rgba(239, 68, 68, 0.2)', color: '#f87171', fontWeight: 700, fontSize: 11, border: '1px solid rgba(239, 68, 68, 0.4)' }} />;
      case 'descartada':
      case 'discarded':
        return <Chip label="Descartada" size="small" sx={{ bgcolor: 'rgba(107, 114, 128, 0.2)', color: '#9ca3af', fontWeight: 700, fontSize: 11, border: '1px solid rgba(107, 114, 128, 0.4)' }} />;
      default:
        return <Chip label="Pendent" size="small" sx={{ bgcolor: 'rgba(59, 130, 246, 0.2)', color: '#60a5fa', fontWeight: 700, fontSize: 11, border: '1px solid rgba(59, 130, 246, 0.4)' }} />;
    }
  };

  if (isLoading && offers.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af' }}>
        <Typography>Carregant el Radar d'oportunitats...</Typography>
      </Box>
    );
  }

  if (offers.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af', backgroundColor: '#111827', borderRadius: 2 }}>
        <Typography variant="h6" sx={{ color: '#e5e7eb', mb: 1 }}>No s'han trobat senyals per al filtre seleccionat</Typography>
        <Typography variant="body2">
          Utilitza el botó "Rastrejar portals ara" o selecciona un altre estat per veure les oportunitats.
        </Typography>
      </Box>
    );
  }

  return (
    <TableContainer
      sx={{
        backgroundColor: '#111827',
        border: '1px solid #374151',
        borderRadius: '12px',
        overflow: 'hidden',
      }}
    >
      <Table>
        <TableHead sx={{ backgroundColor: '#151d2c' }}>
          <TableRow>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px', width: 70, textAlign: 'center' }}>
              SCORE
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>OFERTA & EMPRESA</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>FONT</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>ESTAT</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>INEFICIÈNCIA MANUAL</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>PROPOSTA MICRO-SAAS</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px' }}>DECISOR</TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 600, fontSize: '12px', textAlign: 'right' }}>ACCIONS</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {offers.map((offer) => {
            const scoreStyle = getScoreColor(offer.analysis?.viabilitat_plg_score);
            const isSelected = selectedOffer?.id === offer.id;

            return (
              <TableRow
                key={offer.id}
                onClick={() => onSelectOffer(offer)}
                hover
                sx={{
                  cursor: 'pointer',
                  backgroundColor: isSelected ? 'rgba(37, 99, 235, 0.12)' : 'inherit',
                  borderLeft: isSelected ? '3px solid #2563eb' : '3px solid transparent',
                  '&:hover': {
                    backgroundColor: isSelected ? 'rgba(37, 99, 235, 0.2)' : '#1f2937 !important',
                  },
                }}
              >
                <TableCell sx={{ textAlign: 'center', verticalAlign: 'middle' }}>
                  <Box
                    sx={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      width: 34,
                      height: 34,
                      borderRadius: '8px',
                      fontWeight: 800,
                      fontSize: '14px',
                      backgroundColor: scoreStyle.bg,
                      color: scoreStyle.text,
                      border: `1px solid ${scoreStyle.border}`,
                    }}
                  >
                    {offer.analysis?.viabilitat_plg_score ?? '-'}
                  </Box>
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle', maxWidth: 200 }}>
                  <Typography sx={{ fontWeight: 600, color: '#f9fafb', fontSize: '14px', lineHeight: 1.3 }}>
                    {offer.title}
                  </Typography>
                  <Typography sx={{ color: '#9ca3af', fontSize: '12px', mt: 0.5 }}>
                    {offer.company || 'Empresa confidencial'} {offer.location ? `• ${offer.location}` : ''}
                  </Typography>
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle' }}>
                  {getSourceBadge(offer.source)}
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle' }}>
                  {getStatusBadge(offer.status)}
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle', maxWidth: 240 }}>
                  <Typography sx={{ color: '#cbd5e1', fontSize: '13px', lineHeight: 1.4 }}>
                    {offer.analysis?.ineficiencia_manual || 'Pendent d\'anàlisi'}
                  </Typography>
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle', maxWidth: 240 }}>
                  <Typography sx={{ color: '#e2e8f0', fontWeight: 500, fontSize: '13px', lineHeight: 1.4 }}>
                    {offer.analysis?.proposta_micro_saas || '-'}
                  </Typography>
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle', whiteSpace: 'nowrap' }}>
                  <Typography sx={{ color: '#93c5fd', fontSize: '13px' }}>
                    {offer.analysis?.decisor_compra || '-'}
                  </Typography>
                </TableCell>

                <TableCell sx={{ verticalAlign: 'middle', textAlign: 'right' }} onClick={(e) => e.stopPropagation()}>
                  <Tooltip title="Descartar / Arxivar">
                    <IconButton
                      size="small"
                      onClick={() => discardOffer(offer.id)}
                      sx={{
                        color: '#9ca3af',
                        '&:hover': { color: '#ef4444', bgcolor: 'rgba(239, 68, 68, 0.15)' },
                      }}
                    >
                      <Trash2 size={16} />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
