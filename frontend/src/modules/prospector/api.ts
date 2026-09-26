import { apiClient } from '../../api/client';
import { JobOffer, OfferListResponse, ProgressEvent, ScraperRunResult, SystemStatusResponse } from './types';

export const prospectorApi = {
  login: async (secret: string): Promise<{ token: string; message: string }> => {
    const res = await apiClient.post('/auth/login', { secret });
    return res.data;
  },

  listOffers: async (params?: { status?: string; min_score?: number; limit?: number; offset?: number }): Promise<OfferListResponse> => {
    const res = await apiClient.get('/offers', { params });
    return res.data;
  },

  getOfferByID: async (id: string): Promise<JobOffer> => {
    const res = await apiClient.get(`/offers/${id}`);
    return res.data;
  },

  updateOfferStatus: async (id: string, status: string): Promise<{ success: boolean; message: string; status: string }> => {
    const res = await apiClient.patch(`/offers/${id}/status`, { status });
    return res.data;
  },

  discardOffer: async (id: string): Promise<{ success: boolean; message: string }> => {
    const res = await apiClient.delete(`/offers/${id}`);
    return res.data;
  },

  analyzeURL: async (url: string): Promise<JobOffer> => {
    const res = await apiClient.post('/offers/analyze', { url });
    return res.data;
  },

  runScrapers: async (): Promise<ScraperRunResult> => {
    const res = await apiClient.post('/scrapers/run');
    return res.data;
  },

  streamScrapers: async (onProgress: (event: ProgressEvent) => void): Promise<ScraperRunResult> => {
    const token = localStorage.getItem('salesanalizer_token');
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api';
    const response = await fetch(`${baseUrl}/scrapers/stream`, {
      method: 'GET',
      headers: {
        'Accept': 'text/event-stream',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error ${response.status}`);
    }

    const reader = response.body?.getReader();
    const decoder = new TextDecoder();
    let finalResult: ScraperRunResult = {
      success: true,
      new_offers_found: 0,
      analyzed_count: 0,
      message: 'Rastreig completat.',
    };

    if (reader) {
      let buffer = '';
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmed = line.trim();
          if (trimmed.startsWith('data:')) {
            const jsonStr = trimmed.slice(5).trim();
            if (jsonStr) {
              try {
                const event: ProgressEvent = JSON.parse(jsonStr);
                onProgress(event);
                if (event.result) {
                  finalResult = event.result;
                }
              } catch (e) {
                console.error('Error parsing SSE event:', e);
              }
            }
          }
        }
      }
    }

    return finalResult;
  },

  getSystemStatus: async (): Promise<SystemStatusResponse> => {
    const res = await apiClient.get('/system/status');
    return res.data;
  },
};
