export type OfferSource = string;
export type OfferStatus = 'pendent' | 'enviada' | 'acceptada' | 'rebutjada' | 'descartada' | 'analyzed' | 'pending_analysis' | 'discarded';

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
  signal_type?: string;
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

// Nous models V2: Evidències, Clústers de Dolor i Oportunitats Micro-SaaS
export interface Evidence {
  id: string;
  raw_content: string;
  normalized_url: string;
  content_hash: string;
  source: string;
  source_id?: string;
  author_or_company?: string;
  extracted_process?: string;
  task_description?: string;
  frequency: string;
  manuality_score: number;
  tools_mentioned: string[];
  sector?: string;
  evidence_type?: string;
  source_evidence_quote?: string;
  evidence_confidence: 'alta' | 'mitja' | 'baixa';
  is_duplicate_of?: string;
  created_at: string;
}

export interface PainCluster {
  id: string;
  process_id: string;
  process_name?: string;
  category?: string;
  title: string;
  summary: string;
  status: 'emerging' | 'consolidated' | 'validated' | 'discarded';
  evidence_count: number;
  company_count: number;
  source_count: number;
  sector_breadth: number;
  last_evidence_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PainClusterWithDetails extends PainCluster {
  evidences: Evidence[];
}

export interface FactorScore {
  score: number;
  justification: string;
}

export interface ScoringDimensions {
  // Grup 1: Dolor i Demanda (35%)
  pain_intensity: FactorScore;
  urgency: FactorScore;
  budget_discretion: FactorScore;
  market_reach: FactorScore;

  // Grup 2: Viabilitat Tècnica & PLG (40%)
  implementation_simplicity: FactorScore;
  self_onboarding: FactorScore;
  zero_integrations: FactorScore;
  no_ocr_no_hardware: FactorScore;

  // Grup 3: Negoci i Competició (25%)
  retention_stickiness: FactorScore;
  niche_competition: FactorScore;
  willingness_to_pay: FactorScore;
  scalability_reach: FactorScore;
}

export interface Opportunity {
  id: string;
  cluster_id: string;
  cluster_title?: string;
  process_name?: string;
  category?: string;
  title: string;
  target_user: string;
  buyer_persona: string;
  core_workflow: string;
  value_prop: string;
  pricing_model: string;
  outreach_hook: string;
  scores: ScoringDimensions;
  global_score: number;
  viability_tier: 'excel·lent' | 'prometedora' | 'feble' | 'descartada';
  created_at: string;
  updated_at: string;
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

export interface ProgressEvent {
  type: 'start' | 'progress' | 'fun_fact' | 'joke' | 'analyzing' | 'completed' | 'error';
  step?: string;
  message: string;
  progress: number;
  fun_fact?: string;
  joke?: string;
  result?: ScraperRunResult;
  estimated_secs?: number;
}

