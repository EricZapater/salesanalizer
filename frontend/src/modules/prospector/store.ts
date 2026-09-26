import { create } from 'zustand';
import { prospectorApi } from './api';
import { JobOffer, SystemStatusResponse } from './types';

interface ProspectorState {
  offers: JobOffer[];
  totalOffers: number;
  selectedOffer: JobOffer | null;
  systemStatus: SystemStatusResponse | null;
  isLoading: boolean;
  isAnalyzingUrl: boolean;
  isScraping: boolean;
  error: string | null;
  toastMessage: string | null;

  // Actions
  fetchOffers: (status?: string, minScore?: number) => Promise<void>;
  selectOffer: (offer: JobOffer | null) => void;
  loadOfferDetail: (id: string) => Promise<void>;
  discardOffer: (id: string) => Promise<void>;
  analyzeURL: (url: string) => Promise<boolean>;
  runScrapers: () => Promise<void>;
  fetchSystemStatus: () => Promise<void>;
  setToast: (msg: string | null) => void;
}

export const useProspectorStore = create<ProspectorState>((set, get) => ({
  offers: [],
  totalOffers: 0,
  selectedOffer: null,
  systemStatus: null,
  isLoading: false,
  isAnalyzingUrl: false,
  isScraping: false,
  error: null,
  toastMessage: null,

  fetchOffers: async (status = 'active', minScore) => {
    set({ isLoading: true, error: null });
    try {
      const data = await prospectorApi.listOffers({ status, min_score: minScore });
      set({ offers: data.items, totalOffers: data.total, isLoading: false });
      if (!get().selectedOffer && data.items.length > 0) {
        set({ selectedOffer: data.items[0] });
      }
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant les ofertes', isLoading: false });
    }
  },

  selectOffer: (offer) => {
    set({ selectedOffer: offer });
  },

  loadOfferDetail: async (id: string) => {
    try {
      const fullOffer = await prospectorApi.getOfferByID(id);
      set({ selectedOffer: fullOffer });
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant detall de l\'oferta' });
    }
  },

  discardOffer: async (id: string) => {
    try {
      await prospectorApi.discardOffer(id);
      set((state) => {
        const remaining = state.offers.filter((o) => o.id !== id);
        const nextSelected = state.selectedOffer?.id === id ? (remaining[0] || null) : state.selectedOffer;
        return {
          offers: remaining,
          totalOffers: Math.max(0, state.totalOffers - 1),
          selectedOffer: nextSelected,
          toastMessage: 'Oferta descartada / arxivada',
        };
      });
    } catch (err: any) {
      set({ toastMessage: 'Error descartant l\'oferta' });
    }
  },

  analyzeURL: async (url: string) => {
    set({ isAnalyzingUrl: true, error: null });
    try {
      const newOffer = await prospectorApi.analyzeURL(url);
      set((state) => ({
        offers: [newOffer, ...state.offers.filter((o) => o.id !== newOffer.id)],
        totalOffers: state.totalOffers + 1,
        selectedOffer: newOffer,
        isAnalyzingUrl: false,
        toastMessage: 'Oferta analitzada i registrada amb èxit!',
      }));
      get().fetchSystemStatus();
      return true;
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Error analitzant la URL';
      set({ isAnalyzingUrl: false, toastMessage: msg, error: msg });
      return false;
    }
  },

  runScrapers: async () => {
    set({ isScraping: true });
    try {
      const res = await prospectorApi.runScrapers();
      set({ isScraping: false, toastMessage: res.message });
      await get().fetchOffers();
      await get().fetchSystemStatus();
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Error executant els scrapers';
      set({ isScraping: false, toastMessage: msg });
    }
  },

  fetchSystemStatus: async () => {
    try {
      const status = await prospectorApi.getSystemStatus();
      set({ systemStatus: status });
    } catch (err) {
      // ignore non-critical status error
    }
  },

  setToast: (msg) => {
    set({ toastMessage: msg });
  },
}));
