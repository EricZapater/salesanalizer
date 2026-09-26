export type OfferSource = 'soc' | 'infofeina' | 'manual';
export type OfferStatus = 'pending_analysis' | 'analyzed' | 'discarded';

export interface OpportunityAnalysis {
  id?: string;
  job_offer_id?: string;
  ineficiencia_manual: string;
  proposta_micro_saas: string;
  viabilitat_plg_score: number; // 1-5
  decisor_compra: string;
  ganxo_venda: string;
  analyzed_at?: string;
}

export interface JobOffer {
  id: string;
  source: OfferSource;
  external_id?: string;
  title: string;
  company?: string;
  location?: string;
  url: string;
  raw_text?: string;
  status: OfferStatus;
  analysis?: OpportunityAnalysis;
  created_at: string;
  updated_at?: string;
}

export interface OfferListResponse {
  total: number;
  items: JobOffer[];
}

export interface ScraperStatus {
  source: string;
  status: 'ok' | 'error' | 'idle';
  items_found: number;
  last_run_at: string;
  error_message?: string;
}

export interface SystemStatusResponse {
  signals_today: number;
  daily_limit: number;
  cost_eur: number;
  llm_status: {
    model: string;
    provider: string;
    connected: boolean;
  };
  scrapers: ScraperStatus[];
}

export interface ScraperRunResult {
  success: boolean;
  new_offers_found: number;
  analyzed_count: number;
  message: string;
}
