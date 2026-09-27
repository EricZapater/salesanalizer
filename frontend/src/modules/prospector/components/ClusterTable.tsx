import React from 'react';
import {
  Box,
  Button,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { Layers, Sparkles, Database, Building2 } from 'lucide-react';
import { PainCluster } from '../types';
import { useProspectorStore } from '../store';

interface ClusterTableProps {
  onSelectCluster: (cluster: PainCluster) => void;
}

export const ClusterTable: React.FC<ClusterTableProps> = ({ onSelectCluster }) => {
  const { clusters, selectedCluster, isLoading, synthesizeOpportunity, isSynthesizing } = useProspectorStore();

  const getStatusChip = (status: string) => {
    switch (status) {
      case 'consolidated':
        return <Chip label="Consolidat (≥3 evidències)" size="small" sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', color: '#34d399', fontWeight: 700, border: '1px solid #10b981' }} />;
      case 'validated':
        return <Chip label="Validat" size="small" sx={{ bgcolor: 'rgba(59, 130, 246, 0.2)', color: '#60a5fa', fontWeight: 700 }} />;
      case 'discarded':
        return <Chip label="Descartat" size="small" sx={{ bgcolor: 'rgba(107, 114, 128, 0.2)', color: '#9ca3af' }} />;
      default:
        return <Chip label="Emergent" size="small" sx={{ bgcolor: 'rgba(245, 158, 11, 0.2)', color: '#fbbf24', fontWeight: 600 }} />;
    }
  };

  if (isLoading && clusters.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af' }}>
        <Typography>Carregant clústers de dolor...</Typography>
      </Box>
    );
  }

  if (clusters.length === 0) {
    return (
      <Box sx={{ p: 6, textAlign: 'center', color: '#9ca3af', bgcolor: '#111827', borderRadius: '12px', border: '1px solid #374151' }}>
        <Layers size={40} style={{ margin: '0 auto 12px', color: '#6b7280' }} />
        <Typography variant="h6" sx={{ color: '#e5e7eb', mb: 1 }}>
          Cap clúster de dolor trobat
        </Typography>
        <Typography variant="body2">
          Executa un rastreig o afegeix URLs manuals per generar evidències i agrupar problemes.
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
              Estat & Procés Canònic
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }}>
              Dolor Operatiu Identificat
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }} align="center">
              Evidències
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }} align="center">
              Empreses
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }} align="center">
              Fonts
            </TableCell>
            <TableCell sx={{ color: '#9ca3af', fontWeight: 700, fontSize: '12px', textTransform: 'uppercase' }} align="right">
              Acció
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {clusters.map((c) => {
            const isSelected = selectedCluster?.id === c.id;
            return (
              <TableRow
                key={c.id}
                hover
                onClick={() => onSelectCluster(c)}
                sx={{
                  cursor: 'pointer',
                  bgcolor: isSelected ? 'rgba(59, 130, 246, 0.08)' : 'transparent',
                  borderLeft: isSelected ? '3px solid #3b82f6' : '3px solid transparent',
                  '&:hover': {
                    bgcolor: 'rgba(255, 255, 255, 0.03)',
                  },
                }}
              >
                <TableCell sx={{ color: '#f3f4f6', py: 2 }}>
                  <Box sx={{ mb: 1 }}>{getStatusChip(c.status)}</Box>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#60a5fa' }}>
                    {c.process_name || 'Procés Operatiu'}
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                    {c.category}
                  </Typography>
                </TableCell>

                <TableCell sx={{ color: '#f3f4f6', maxWidth: 360 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#fff', mb: 0.5 }}>
                    {c.title}
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#9ca3af', fontSize: '13px', lineHeight: 1.4 }}>
                    {c.summary}
                  </Typography>
                </TableCell>

                <TableCell align="center" sx={{ color: '#f3f4f6' }}>
                  <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, bgcolor: 'rgba(255,255,255,0.05)', px: 1.2, py: 0.4, borderRadius: '6px' }}>
                    <Layers size={14} color="#60a5fa" />
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#fff' }}>
                      {c.evidence_count}
                    </Typography>
                  </Box>
                </TableCell>

                <TableCell align="center" sx={{ color: '#f3f4f6' }}>
                  <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
                    <Building2 size={14} color="#a78bfa" />
                    <Typography variant="body2" sx={{ fontWeight: 600, color: '#e5e7eb' }}>
                      {c.company_count}
                    </Typography>
                  </Box>
                </TableCell>

                <TableCell align="center" sx={{ color: '#f3f4f6' }}>
                  <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
                    <Database size={14} color="#34d399" />
                    <Typography variant="body2" sx={{ fontWeight: 600, color: '#e5e7eb' }}>
                      {c.source_count}
                    </Typography>
                  </Box>
                </TableCell>

                <TableCell align="right" sx={{ color: '#f3f4f6' }}>
                  <Button
                    variant="contained"
                    size="small"
                    disabled={isSynthesizing}
                    onClick={(e) => {
                      e.stopPropagation();
                      synthesizeOpportunity(c.id);
                    }}
                    startIcon={<Sparkles size={14} />}
                    sx={{
                      background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
                      textTransform: 'none',
                      fontWeight: 700,
                      fontSize: '12px',
                      borderRadius: '6px',
                    }}
                  >
                    Sintetitzar Micro-SaaS
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
