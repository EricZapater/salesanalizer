import { create } from 'zustand';
import { prospectorApi } from './api';
import {
  JobOffer,
  PainCluster,
  PainClusterWithDetails,
  Opportunity,
  ProgressEvent,
  SystemStatusResponse,
} from './types';

interface ProspectorState {
  activeView: 'clusters' | 'opportunities' | 'offers';
  offers: JobOffer[];
  totalOffers: number;
  selectedOffer: JobOffer | null;

  clusters: PainCluster[];
  totalClusters: number;
  selectedCluster: PainClusterWithDetails | null;

  opportunities: Opportunity[];
  totalOpportunities: number;
  selectedOpportunity: Opportunity | null;

  systemStatus: SystemStatusResponse | null;
  statusFilter: string;
  clusterFilter: string;
  opportunityFilter: string;
  isLoading: boolean;
  isAnalyzingUrl: boolean;
  isScraping: boolean;
  isSynthesizing: boolean;
  progressState: ProgressEvent | null;
  error: string | null;
  toastMessage: string | null;

  // Actions
  setActiveView: (view: 'clusters' | 'opportunities' | 'offers') => void;
  fetchOffers: (status?: string, minScore?: number) => Promise<void>;
  fetchClusters: (status?: string, minEvidence?: number) => Promise<void>;
  fetchOpportunities: (minScore?: number, tier?: string) => Promise<void>;
  setStatusFilter: (status: string) => Promise<void>;
  setClusterFilter: (status: string) => Promise<void>;
  setOpportunityFilter: (tier: string) => Promise<void>;
  selectOffer: (offer: JobOffer | null) => void;
  selectCluster: (cluster: PainCluster | null) => Promise<void>;
  selectOpportunity: (opp: Opportunity | null) => void;
  loadOfferDetail: (id: string) => Promise<void>;
  updateOfferStatus: (id: string, status: string) => Promise<void>;
  discardOffer: (id: string) => Promise<void>;
  synthesizeOpportunity: (clusterId: string) => Promise<Opportunity | null>;
  analyzeURL: (url: string) => Promise<boolean>;
  runScrapers: () => Promise<void>;
  fetchSystemStatus: () => Promise<void>;
  setToast: (msg: string | null) => void;
  clearProgress: () => void;
}

export const useProspectorStore = create<ProspectorState>((set, get) => ({
  activeView: 'clusters',
  offers: [],
  totalOffers: 0,
  selectedOffer: null,

  clusters: [],
  totalClusters: 0,
  selectedCluster: null,

  opportunities: [],
  totalOpportunities: 0,
  selectedOpportunity: null,

  systemStatus: null,
  statusFilter: 'active',
  clusterFilter: 'all',
  opportunityFilter: 'all',
  isLoading: false,
  isAnalyzingUrl: false,
  isScraping: false,
  isSynthesizing: false,
  progressState: null,
  error: null,
  toastMessage: null,

  setActiveView: (view) => {
    set({ activeView: view });
    if (view === 'clusters') {
      get().fetchClusters();
    } else if (view === 'opportunities') {
      get().fetchOpportunities();
    } else {
      get().fetchOffers();
    }
  },

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

  fetchClusters: async (status, minEvidence) => {
    const currentStatus = status !== undefined ? status : get().clusterFilter;
    set({ isLoading: true, error: null });
    try {
      const data = await prospectorApi.listClusters({ status: currentStatus, min_evidence: minEvidence });
      set({ clusters: data.items, totalClusters: data.total, isLoading: false });
      if (!get().selectedCluster && data.items.length > 0) {
        await get().selectCluster(data.items[0]);
      }
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant els clústers', isLoading: false });
    }
  },

  fetchOpportunities: async (minScore, tier) => {
    const currentTier = tier !== undefined ? tier : get().opportunityFilter;
    set({ isLoading: true, error: null });
    try {
      const data = await prospectorApi.listOpportunities({ min_score: minScore, tier: currentTier });
      set({ opportunities: data.items, totalOpportunities: data.total, isLoading: false });
      if (!get().selectedOpportunity && data.items.length > 0) {
        set({ selectedOpportunity: data.items[0] });
      }
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant oportunitats', isLoading: false });
    }
  },

  setStatusFilter: async (status: string) => {
    set({ statusFilter: status });
    await get().fetchOffers(status);
  },

  setClusterFilter: async (status: string) => {
    set({ clusterFilter: status });
    await get().fetchClusters(status);
  },

  setOpportunityFilter: async (tier: string) => {
    set({ opportunityFilter: tier });
    await get().fetchOpportunities(undefined, tier);
  },

  selectOffer: (offer) => {
    set({ selectedOffer: offer });
  },

  selectCluster: async (cluster) => {
    if (!cluster) {
      set({ selectedCluster: null });
      return;
    }
    try {
      const details = await prospectorApi.getClusterDetails(cluster.id);
      set({ selectedCluster: details });
    } catch (e) {
      set({ selectedCluster: { ...cluster, evidences: [] } });
    }
  },

  selectOpportunity: (opp) => {
    set({ selectedOpportunity: opp });
  },

  loadOfferDetail: async (id: string) => {
    try {
      const fullOffer = await prospectorApi.getOfferByID(id);
      set({ selectedOffer: fullOffer });
    } catch (err: any) {
      set({ error: err.response?.data?.message || 'Error carregant detall de l\'oferta' });
    }
  },

  synthesizeOpportunity: async (clusterId: string) => {
    set({ isSynthesizing: true });
    try {
      const opp = await prospectorApi.synthesizeOpportunity(clusterId);
      set((state) => ({
        opportunities: [opp, ...state.opportunities.filter((o) => o.id !== opp.id)],
        selectedOpportunity: opp,
        isSynthesizing: false,
        toastMessage: `Oportunitat ${opp.title} generada amb èxit!`,
      }));
      await get().fetchClusters();
      return opp;
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Error sintetitzant l\'oportunitat';
      set({ isSynthesizing: false, toastMessage: msg });
      return null;
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
      get().fetchClusters();
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
      await get().fetchClusters();
      await get().fetchOpportunities();
      await get().fetchOffers();
      await get().fetchSystemStatus();
    } catch (err: any) {
      try {
        const res = await prospectorApi.runScrapers();
        set({ isScraping: false, toastMessage: res.message });
        await get().fetchClusters();
        await get().fetchOpportunities();
        await get().fetchOffers();
        await get().fetchSystemStatus();
      } catch (innerErr: any) {
        const msg = innerErr.response?.data?.message || 'Error executant els scrapers';
        set({ isScraping: false, toastMessage: msg });
      }
    } finally {
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
      // ignore
    }
  },

  setToast: (msg) => {
    set({ toastMessage: msg });
  },

  clearProgress: () => {
    set({ progressState: null, isScraping: false });
  },
}));

