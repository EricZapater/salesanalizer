import { apiClient } from '../../api/client';
import { JobOffer, OfferListResponse, ScraperRunResult, SystemStatusResponse } from './types';

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

  getSystemStatus: async (): Promise<SystemStatusResponse> => {
    const res = await apiClient.get('/system/status');
    return res.data;
  },
};
