import React, { useState } from 'react';
import {
  Box,
  Button,
  Chip,
  Drawer,
  IconButton,
  LinearProgress,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import {
  X,
  Copy,
  ExternalLink,
  Sparkles,
  Quote,
  Award,
  Check,
  AlertCircle,
} from 'lucide-react';
import { useProspectorStore } from '../store';
import { FactorScore, PainClusterWithDetails, Opportunity, JobOffer } from '../types';

interface LateralDrawerProps {
  open: boolean;
  onClose: () => void;
  cluster: PainClusterWithDetails | null;
  opportunity: Opportunity | null;
  offer: JobOffer | null;
}

export const LateralDrawer: React.FC<LateralDrawerProps> = ({
  open,
  onClose,
  cluster,
  opportunity,
  offer,
}) => {
  const [currentTab, setCurrentTab] = useState(0);
  const [copied, setCopied] = useState(false);
  const { setToast, synthesizeOpportunity, isSynthesizing } = useProspectorStore();

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setToast(`${label} copiat al portapapers!`);
    setTimeout(() => setCopied(false), 2000);
  };

  const renderFactorRow = (label: string, factor?: FactorScore) => {
    if (!factor) return null;
    const scorePct = (factor.score / 5) * 100;
    return (
      <Box sx={{ mb: 2, pb: 1.5, borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, color: '#e2e8f0', fontSize: '13px' }}>
            {label}
          </Typography>
          <Chip
            label={`${factor.score} / 5`}
            size="small"
            sx={{
              height: 20,
              fontSize: '11px',
              fontWeight: 700,
              bgcolor: factor.score >= 4 ? 'rgba(16, 185, 129, 0.2)' : factor.score >= 3 ? 'rgba(59, 130, 246, 0.2)' : 'rgba(239, 68, 68, 0.2)',
              color: factor.score >= 4 ? '#34d399' : factor.score >= 3 ? '#93c5fd' : '#f87171',
            }}
          />
        </Box>
        <LinearProgress
          variant="determinate"
          value={scorePct}
          sx={{
            height: 4,
            borderRadius: 2,
            bgcolor: 'rgba(255,255,255,0.08)',
            '& .MuiLinearProgress-bar': {
              bgcolor: factor.score >= 4 ? '#10b981' : factor.score >= 3 ? '#3b82f6' : '#ef4444',
            },
            mb: 0.8,
          }}
        />
        <Typography variant="caption" sx={{ color: '#9ca3af', fontStyle: 'italic', display: 'block', lineHeight: 1.3 }}>
          "{factor.justification}"
        </Typography>
      </Box>
    );
  };

  const title = opportunity?.title || cluster?.title || offer?.title || 'Detall';
  const subtitle = opportunity ? `Oportunitat Micro-SaaS (${opportunity.viability_tier})` : cluster ? `Clúster de Dolor (${cluster.status})` : 'Senyal';

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      variant="temporary"
      PaperProps={{
        sx: {
          width: { xs: '100%', sm: 540 },
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
          p: 2.5,
          bgcolor: '#151d2c',
          borderBottom: '1px solid #374151',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Box sx={{ maxWidth: '85%' }}>
          <Typography variant="caption" sx={{ textTransform: 'uppercase', color: '#93c5fd', fontWeight: 700, letterSpacing: '0.05em' }}>
            {subtitle}
          </Typography>
          <Typography variant="h6" sx={{ fontWeight: 800, color: '#fff', lineHeight: 1.3 }}>
            {title}
          </Typography>
        </Box>
        <IconButton onClick={onClose} size="small" sx={{ color: '#9ca3af', '&:hover': { color: '#fff' } }}>
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
        }}
      >
        <Tab label="💡 Visió General" />
        <Tab label={`💬 "Per què ho creiem?" (${cluster?.evidences?.length || 0})`} />
        {opportunity && <Tab label="📊 Avaluació 12 Factors" />}
      </Tabs>

      {/* Content */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 3 }}>
        {/* Tab 0: Visió General */}
        {currentTab === 0 && (
          <Box>
            {opportunity ? (
              <Box>
                {/* Micro-SaaS Value Prop Card */}
                <Box sx={{ p: 2.5, bgcolor: '#1a2234', borderRadius: '10px', border: '1px solid #3b82f6', mb: 3 }}>
                  <Typography variant="caption" sx={{ color: '#93c5fd', fontWeight: 700, textTransform: 'uppercase' }}>
                    Proposta de Valor
                  </Typography>
                  <Typography variant="h6" sx={{ fontWeight: 800, color: '#fff', my: 0.5 }}>
                    {opportunity.title}
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#e2e8f0', lineHeight: 1.5, mb: 2 }}>
                    {opportunity.value_prop}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
                    <Chip label={`Preu: ${opportunity.pricing_model}`} size="small" sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', color: '#34d399', fontWeight: 700 }} />
                    <Chip label={`Score: ${opportunity.global_score.toFixed(2)}/5.0`} size="small" sx={{ bgcolor: 'rgba(59, 130, 246, 0.2)', color: '#93c5fd', fontWeight: 700 }} />
                  </Box>
                </Box>

                {/* Workflow Card */}
                <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', border: '1px solid #374151', mb: 2.5 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#fff', mb: 0.5 }}>
                    Workflow Central (1 sol ús)
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#9ca3af', lineHeight: 1.5 }}>
                    {opportunity.core_workflow}
                  </Typography>
                </Box>

                {/* Target & Buyer */}
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 2.5 }}>
                  <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', border: '1px solid #374151' }}>
                    <Typography variant="caption" sx={{ color: '#9ca3af', fontWeight: 700 }}>
                      Decisor de Compra
                    </Typography>
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#fff', mt: 0.5 }}>
                      {opportunity.buyer_persona}
                    </Typography>
                  </Box>
                  <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', border: '1px solid #374151' }}>
                    <Typography variant="caption" sx={{ color: '#9ca3af', fontWeight: 700 }}>
                      Usuari Diari
                    </Typography>
                    <Typography variant="body2" sx={{ fontWeight: 700, color: '#fff', mt: 0.5 }}>
                      {opportunity.target_user}
                    </Typography>
                  </Box>
                </Box>

                {/* Outreach Hook */}
                <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', border: '1px solid #374151' }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#fff' }}>
                      Ganxo de Contacte Directe
                    </Typography>
                    <Button
                      size="small"
                      startIcon={copied ? <Check size={14} /> : <Copy size={14} />}
                      onClick={() => handleCopy(opportunity.outreach_hook, 'Ganxo')}
                      sx={{ textTransform: 'none', color: '#60a5fa', fontSize: '12px' }}
                    >
                      {copied ? 'Copiat!' : 'Copiar'}
                    </Button>
                  </Box>
                  <Typography variant="body2" sx={{ color: '#cbd5e1', fontStyle: 'italic', bgcolor: '#0b1120', p: 1.5, borderRadius: '6px' }}>
                    "{opportunity.outreach_hook}"
                  </Typography>
                </Box>
              </Box>
            ) : cluster ? (
              <Box>
                <Box sx={{ p: 2.5, bgcolor: '#151d2c', borderRadius: '10px', border: '1px solid #374151', mb: 3 }}>
                  <Typography variant="caption" sx={{ color: '#60a5fa', fontWeight: 700, textTransform: 'uppercase' }}>
                    {cluster.category}
                  </Typography>
                  <Typography variant="h6" sx={{ fontWeight: 800, color: '#fff', my: 0.5 }}>
                    {cluster.process_name}
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#9ca3af', lineHeight: 1.5, mb: 2 }}>
                    {cluster.summary}
                  </Typography>
                  <Button
                    variant="contained"
                    fullWidth
                    disabled={isSynthesizing}
                    onClick={() => synthesizeOpportunity(cluster.id)}
                    startIcon={<Sparkles size={16} />}
                    sx={{
                      background: 'linear-gradient(135deg, #2563eb, #7c3aed)',
                      textTransform: 'none',
                      fontWeight: 700,
                      py: 1,
                    }}
                  >
                    {isSynthesizing ? 'Sintetitzant amb IA...' : 'Sintetitzar i Auditar Oportunitat'}
                  </Button>
                </Box>

                {/* Metrics */}
                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 1.5, mb: 3 }}>
                  <Box sx={{ p: 1.5, bgcolor: '#151d2c', borderRadius: '8px', textAlign: 'center' }}>
                    <Typography variant="caption" sx={{ color: '#9ca3af' }}>Evidències</Typography>
                    <Typography variant="h6" sx={{ fontWeight: 800, color: '#60a5fa' }}>{cluster.evidence_count}</Typography>
                  </Box>
                  <Box sx={{ p: 1.5, bgcolor: '#151d2c', borderRadius: '8px', textAlign: 'center' }}>
                    <Typography variant="caption" sx={{ color: '#9ca3af' }}>Empreses</Typography>
                    <Typography variant="h6" sx={{ fontWeight: 800, color: '#a78bfa' }}>{cluster.company_count}</Typography>
                  </Box>
                  <Box sx={{ p: 1.5, bgcolor: '#151d2c', borderRadius: '8px', textAlign: 'center' }}>
                    <Typography variant="caption" sx={{ color: '#9ca3af' }}>Fonts</Typography>
                    <Typography variant="h6" sx={{ fontWeight: 800, color: '#34d399' }}>{cluster.source_count}</Typography>
                  </Box>
                </Box>
              </Box>
            ) : null}
          </Box>
        )}

        {/* Tab 1: "Per què ho creiem?" Cites Literals */}
        {currentTab === 1 && (
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#fff', mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Quote size={16} color="#38bdf8" /> Cites textuals extretes de les fonts
            </Typography>

            {(!cluster?.evidences || cluster.evidences.length === 0) ? (
              <Box sx={{ p: 4, textAlign: 'center', color: '#9ca3af', bgcolor: '#151d2c', borderRadius: '8px' }}>
                <AlertCircle size={28} style={{ margin: '0 auto 8px', color: '#6b7280' }} />
                <Typography variant="body2">No s'han trobat evidències desglossades per a aquest clúster.</Typography>
              </Box>
            ) : (
              cluster.evidences.map((ev, idx) => (
                <Box
                  key={ev.id || idx}
                  sx={{
                    p: 2,
                    mb: 2,
                    bgcolor: '#151d2c',
                    borderRadius: '8px',
                    border: '1px solid #374151',
                  }}
                >
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Chip label={ev.source} size="small" sx={{ bgcolor: 'rgba(59, 130, 246, 0.15)', color: '#93c5fd', fontSize: '11px', height: 20 }} />
                      <Chip
                        label={`Confiança ${ev.evidence_confidence}`}
                        size="small"
                        sx={{
                          height: 20,
                          fontSize: '11px',
                          bgcolor: ev.evidence_confidence === 'alta' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(245, 158, 11, 0.2)',
                          color: ev.evidence_confidence === 'alta' ? '#34d399' : '#fbbf24',
                        }}
                      />
                    </Box>
                    {ev.normalized_url && (
                      <IconButton size="small" component="a" href={ev.normalized_url} target="_blank" sx={{ color: '#9ca3af' }}>
                        <ExternalLink size={14} />
                      </IconButton>
                    )}
                  </Box>

                  {ev.source_evidence_quote ? (
                    <Box sx={{ p: 1.5, bgcolor: '#0b1120', borderRadius: '6px', borderLeft: '3px solid #38bdf8', mb: 1.5 }}>
                      <Typography variant="body2" sx={{ color: '#e2e8f0', fontStyle: 'italic', fontSize: '13px' }}>
                        "{ev.source_evidence_quote}"
                      </Typography>
                    </Box>
                  ) : null}

                  <Typography variant="caption" sx={{ color: '#9ca3af', display: 'block', mb: 0.5 }}>
                    <strong>Procés:</strong> {ev.extracted_process}
                  </Typography>

                  {ev.tools_mentioned && ev.tools_mentioned.length > 0 && (
                    <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 1 }}>
                      {ev.tools_mentioned.map((t, i) => (
                        <Chip key={i} label={t} size="small" sx={{ bgcolor: 'rgba(255,255,255,0.06)', color: '#cbd5e1', fontSize: '10px', height: 18 }} />
                      ))}
                    </Box>
                  )}
                </Box>
              ))
            )}
          </Box>
        )}

        {/* Tab 2: Avaluació 12 Factors (només quan hi ha oportunitat) */}
        {currentTab === 2 && opportunity && (
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 800, color: '#fff', mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Award size={18} color="#10b981" /> Matriu de Viabilitat dels 12 Factors
            </Typography>

            {/* Grup 1: Dolor & Demanda (35%) */}
            <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', mb: 2.5, border: '1px solid #374151' }}>
              <Typography variant="subtitle2" sx={{ color: '#93c5fd', fontWeight: 700, mb: 1.5 }}>
                1. Dolor i Demanda (Pes: 35%)
              </Typography>
              {renderFactorRow('Intensitat del Dolor', opportunity.scores?.pain_intensity)}
              {renderFactorRow('Urgència Operativa', opportunity.scores?.urgency)}
              {renderFactorRow('Poder de Decisió Pressupostària', opportunity.scores?.budget_discretion)}
              {renderFactorRow('Abast del Mercat PIME', opportunity.scores?.market_reach)}
            </Box>

            {/* Grup 2: Viabilitat Tècnica & PLG (40%) */}
            <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', mb: 2.5, border: '1px solid #374151' }}>
              <Typography variant="subtitle2" sx={{ color: '#34d399', fontWeight: 700, mb: 1.5 }}>
                2. Viabilitat Tècnica & PLG (Pes: 40%)
              </Typography>
              {renderFactorRow('Simplicitat d\'Implementació (<2 setm.)', opportunity.scores?.implementation_simplicity)}
              {renderFactorRow('Self-Onboarding Autònom (<5 min.)', opportunity.scores?.self_onboarding)}
              {renderFactorRow('0 Integracions Inicials', opportunity.scores?.zero_integrations)}
              {renderFactorRow('100% Digital (0 OCR / Paper físic)', opportunity.scores?.no_ocr_no_hardware)}
            </Box>

            {/* Grup 3: Negoci & Competició (25%) */}
            <Box sx={{ p: 2, bgcolor: '#151d2c', borderRadius: '8px', mb: 2.5, border: '1px solid #374151' }}>
              <Typography variant="subtitle2" sx={{ color: '#c084fc', fontWeight: 700, mb: 1.5 }}>
                3. Negoci i Competició (Pes: 25%)
              </Typography>
              {renderFactorRow('Retenció & Ús Recurrent', opportunity.scores?.retention_stickiness)}
              {renderFactorRow('Buit de Competidors Específics', opportunity.scores?.niche_competition)}
              {renderFactorRow('Disposició a Pagar & ROI Clar', opportunity.scores?.willingness_to_pay)}
              {renderFactorRow('Escalabilitat a Altres Sectors', opportunity.scores?.scalability_reach)}
            </Box>
          </Box>
        )}
      </Box>
    </Drawer>
  );
};
