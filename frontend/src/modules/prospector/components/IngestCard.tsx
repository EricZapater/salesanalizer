import React, { useState } from 'react';
import { Box, Button, Card, CircularProgress, TextField, Typography } from '@mui/material';
import { Sparkles, Search } from 'lucide-react';
import { useProspectorStore } from '../store';

export const IngestCard: React.FC = () => {
  const [url, setUrl] = useState('');
  const { analyzeURL, isAnalyzingUrl } = useProspectorStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim()) return;

    const ok = await analyzeURL(url.trim());
    if (ok) {
      setUrl('');
    }
  };

  return (
    <Card
      sx={{
        backgroundColor: '#111827',
        border: '1px solid #374151',
        borderRadius: '12px',
        p: '18px 22px',
        mb: 3,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
        <Search size={18} color="#60a5fa" />
        <Typography variant="subtitle2" sx={{ fontWeight: 600, color: '#f9fafb' }}>
          Ingestió Manual d'Oferta
        </Typography>
      </Box>

      <Box
        component="form"
        onSubmit={handleSubmit}
        sx={{ display: 'flex', gap: 2, alignItems: 'center' }}
      >
        <TextField
          fullWidth
          size="small"
          placeholder="Enganxa la URL d'una oferta de Feina Activa, Infofeina o qualsevol portal..."
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          disabled={isAnalyzingUrl}
          sx={{
            backgroundColor: '#090d16',
            borderRadius: '8px',
            '& .MuiOutlinedInput-root': {
              color: '#fff',
              '& fieldset': { borderColor: '#374151' },
              '&:hover fieldset': { borderColor: '#4b5563' },
              '&.Mui-focused fieldset': { borderColor: '#2563eb' },
            },
          }}
        />
        <Button
          type="submit"
          variant="contained"
          disabled={isAnalyzingUrl || !url.trim()}
          startIcon={isAnalyzingUrl ? <CircularProgress size={16} color="inherit" /> : <Sparkles size={16} />}
          sx={{
            bgcolor: '#2563eb',
            '&:hover': { bgcolor: '#1d4ed8' },
            textTransform: 'none',
            fontWeight: 600,
            whiteSpace: 'nowrap',
            px: 3,
            py: 1,
            borderRadius: '8px',
          }}
        >
          {isAnalyzingUrl ? 'Analitzant amb Groq...' : 'Analitzar amb Groq'}
        </Button>
      </Box>
    </Card>
  );
};
