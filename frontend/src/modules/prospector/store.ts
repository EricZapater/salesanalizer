import { create } from 'zustand';
import { prospectorApi } from './api';
import { JobOffer, ProgressEvent, SystemStatusResponse } from './types';

interface ProspectorState {
  offers: JobOffer[];
  totalOffers: number;
  selectedOffer: JobOffer | null;
  systemStatus: SystemStatusResponse | null;
  statusFilter: string;
  isLoading: boolean;
  isAnalyzingUrl: boolean;
  isScraping: boolean;
  progressState: ProgressEvent | null;
  error: string | null;
  toastMessage: string | null;

  // Actions
  fetchOffers: (status?: string, minScore?: number) => Promise<void>;
  setStatusFilter: (status: string) => Promise<void>;
  selectOffer: (offer: JobOffer | null) => void;
  loadOfferDetail: (id: string) => Promise<void>;
  updateOfferStatus: (id: string, status: string) => Promise<void>;
  discardOffer: (id: string) => Promise<void>;
  analyzeURL: (url: string) => Promise<boolean>;
  runScrapers: () => Promise<void>;
  fetchSystemStatus: () => Promise<void>;
  setToast: (msg: string | null) => void;
  clearProgress: () => void;
}

export const useProspectorStore = create<ProspectorState>((set, get) => ({
  offers: [],
  totalOffers: 0,
  selectedOffer: null,
  systemStatus: null,
  statusFilter: 'active',
  isLoading: false,
  isAnalyzingUrl: false,
  isScraping: false,
  progressState: null,
  error: null,
  toastMessage: null,

  fetchOffers: async (status, minScore) => {
    const currentStatus = status !== undefined ? status : get().statusFilter;
    set({ isLoading: true, error: null });
    try {
      const data = await prospectorApi.listOffers({ status: currentStatus, min_score: minScore });
      set({ offers: data.items, totalOffers: data.total, isLoading: false });
      if (!get().selectedOffer && data.items.length > 0) {
        set({ selectedOffer: data.items[0] });
      }
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant les ofertes', isLoading: false });
    }
  },

  setStatusFilter: async (status: string) => {
    set({ statusFilter: status });
    await get().fetchOffers(status);
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

  updateOfferStatus: async (id: string, status: string) => {
    try {
      await prospectorApi.updateOfferStatus(id, status);
      set((state) => {
        const updatedOffers = state.offers.map((o) => (o.id === id ? { ...o, status: status as any } : o));
        const updatedSelected = state.selectedOffer?.id === id ? { ...state.selectedOffer, status: status as any } : state.selectedOffer;
        return {
          offers: updatedOffers,
          selectedOffer: updatedSelected,
          toastMessage: `Estat canviat a: ${status}`,
        };
      });
      // Si estem filtrant per un estat específic i l'oferta ja no hi pertany, recarreguem
      const currentFilter = get().statusFilter;
      if (currentFilter !== 'all' && currentFilter !== 'active') {
        await get().fetchOffers();
      }
    } catch (err: any) {
      set({ toastMessage: 'Error actualitzant l\'estat' });
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
          toastMessage: 'Oportunitat descartada',
        };
      });
    } catch (err: any) {
      set({ toastMessage: 'Error descartant l\'oportunitat' });
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
    set({
      isScraping: true,
      progressState: {
        type: 'start',
        message: '🚀 Iniciant rastreig en paral·lel...',
        progress: 10,
        estimated_secs: 20,
        fun_fact: '💡 Preparant extractors multicanal...',
      },
    });

    try {
      const res = await prospectorApi.streamScrapers((event) => {
        set({ progressState: event });
      });
      set({
        isScraping: false,
        toastMessage: res.message || 'Rastreig completat amb èxit!',
      });
      await get().fetchOffers();
      await get().fetchSystemStatus();
    } catch (err: any) {
      // Fallback a crida normal si SSE fallés
      try {
        const res = await prospectorApi.runScrapers();
        set({ isScraping: false, toastMessage: res.message });
        await get().fetchOffers();
        await get().fetchSystemStatus();
      } catch (innerErr: any) {
        const msg = innerErr.response?.data?.message || 'Error executant els scrapers';
        set({ isScraping: false, toastMessage: msg });
      }
    } finally {
      // Deixar el progrés visible 2 segons després de completar
      setTimeout(() => {
        set({ progressState: null });
      }, 3000);
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

  clearProgress: () => {
    set({ progressState: null, isScraping: false });
  },
}));
