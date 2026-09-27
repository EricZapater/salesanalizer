import React from 'react';
import {
  Box,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { Sparkles, Users, DollarSign, Award, ArrowRight } from 'lucide-react';
import { Opportunity } from '../types';
import { useProspectorStore } from '../store';

interface OpportunityTableProps {
  onSelectOpportunity: (opp: Opportunity) => void;
}

export const OpportunityTable: React.FC<OpportunityTableProps> = ({ onSelectOpportunity }) => {
  const { opportunities, selectedOpportunity, isLoading } = useProspectorStore();

  const getTierChip = (tier: string, score: number) => {
    switch (tier) {
      case 'excel·lent':
        return (
          <Chip
            icon={<Award size={14} color="#10b981" />}
            label={`${score.toFixed(2)} / 5.0 — Excel·lent`}
            size="small"
            sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', color: '#34d399', fontWeight: 800, border: '1px solid #10b981' }}
          />
        );
      case 'prometedora':
        return (
          <Chip
            icon={<Sparkles size={14} color="#60a5fa" />}
            label={`${score.toFixed(2)} / 5.0 — Prometedora`}
            size="small"
            sx={{ bgcolor: 'rgba(59, 130, 246, 0.2)', color: '#93c5fd', fontWeight: 700, border: '1px solid #3b82f6' }}
          />
        );
      case 'feble':
        return (
          <Chip
            label={`${score.toFixed(2)} / 5.0 — Feble`}
            size="small"
            sx={{ bgcolor: 'rgba(245, 158, 11, 0.2)', color: '#fbbf24', fontWeight: 600 }}
          />
        );
      default:
        return (
          <Chip
            label={`${score.toFixed(2)} / 5.0 — Descartada`}
            size="small"
            sx={{ bgcolor: 'rgba(107, 114, 128, 0.2)', color: '#9ca3af' }}
          />
        );
    }
  };

  if (isLoading && opportunities.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af' }}>
        <Typography>Carregant oportunitats Micro-SaaS...</Typography>
      </Box>
    );
  }

  if (opportunities.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af', bgcolor: '#111827', borderRadius: '12px', border: '1px solid #374151' }}>
        <Sparkles size={40} style={{ margin: '0 auto 12px', color: '#6b7280' }} />
        <Typography variant="h6" sx={{ color: '#e5e7eb', mb: 1 }}>
          Cap oportunitat sintetitzada encara
        </Typography>
        <Typography variant="body2">
          Ves a la pestanya de Clústers i fes clic a "Sintetitzar Micro-SaaS" en qualsevol clúster de dolor.
        </Typography>
      </Box>
    );
  }

  return (
    <TableContainer
      sx={{
        bgcolor: '#111827',
        borderRadius: '12px',
        border: '1px solid #374151',
        overflow: 'hidden',
      }}
    >
      <Table sx={{ minWidth: 800 }}>
        <TableHead sx={{ bgcolor: '#151d2c' }}>
          <TableRow>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }}>
              Puntuació & Viabilitat
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }}>
              Solució Micro-SaaS & Proposta
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }}>
              Decisor & Usuari
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }}>
              Model de Preu
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }} align="right">
              Detall
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {opportunities.map((opp) => {
            const isSelected = selectedOpportunity?.id === opp.id;
            return (
              <TableRow
                key={opp.id}
                hover
                onClick={() => onSelectOpportunity(opp)}
                sx={{
                  cursor: 'pointer',
                  bgcolor: isSelected ? 'rgba(16, 185, 129, 0.08)' : 'transparent',
                  borderLeft: isSelected ? '3px solid #10b981' : '3px solid transparent',
                  '&:hover': {
                    bgcolor: 'rgba(255, 255, 255, 0.03)',
                  },
                }}
              >
                <TableCell sx={{ color: '#f3f4f6', py: 2 }}>
                  <Box sx={{ mb: 1 }}>{getTierChip(opp.viability_tier, opp.global_score)}</Box>
                  <Typography variant="caption" sx={{ color: '#9ca3af', display: 'block' }}>
                    Procés: <strong style={{ color: '#e5e7eb' }}>{opp.process_name}</strong>
                  </Typography>
                </TableCell>

                <TableCell sx={{ color: '#f3f4f6', maxWidth: 360 }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 800, color: '#38bdf8', mb: 0.5 }}>
                    {opp.title}
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#e5e7eb', fontSize: '13px', lineHeight: 1.4, mb: 0.5 }}>
                    {opp.value_prop}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#9ca3af', fontStyle: 'italic' }}>
                    Flux: {opp.core_workflow}
                  </Typography>
                </TableCell>

                <TableCell sx={{ color: '#f3f4f6' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8, mb: 0.5 }}>
                    <Users size={14} color="#a78bfa" />
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#fff' }}>
                      {opp.buyer_persona}
                    </Typography>
                  </Box>
                  <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                    Usuari diari: {opp.target_user}
                  </Typography>
                </TableCell>

                <TableCell sx={{ color: '#f3f4f6' }}>
                  <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, bgcolor: 'rgba(255,255,255,0.05)', px: 1.2, py: 0.5, borderRadius: '6px' }}>
                    <DollarSign size={14} color="#34d399" />
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#34d399' }}>
                      {opp.pricing_model}
                    </Typography>
                  </Box>
                </TableCell>

                <TableCell align="right" sx={{ color: '#9ca3af' }}>
                  <ArrowRight size={18} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
