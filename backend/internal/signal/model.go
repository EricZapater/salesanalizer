package signal

import (
	"encoding/json"
	"time"
)

type Signal struct {
	ID         string               `json:"id"`
	Source     string               `json:"source"`
	SignalType string               `json:"signal_type"` // "oferta_feina" o "queixa_forum"
	ExternalID *string              `json:"external_id,omitempty"`
	Title      string               `json:"title"`
	Company    *string              `json:"company,omitempty"`
	Location   *string              `json:"location,omitempty"`
	URL        string               `json:"url"`
	RawText    string               `json:"raw_text,omitempty"`
	Status     string               `json:"status"` // "pendent", "enviada", "acceptada", "rebutjada", "descartada"
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Analysis   *OpportunityAnalysis `json:"analysis,omitempty"`
}

type OpportunityAnalysis struct {
	ID                 string          `json:"id,omitempty"`
	SignalID           string          `json:"signal_id,omitempty"`
	IneficienciaManual string          `json:"ineficiencia_manual"`
	PropostaMicroSaas  string          `json:"proposta_micro_saas"`
	ViabilitatPLGScore int             `json:"viabilitat_plg_score"`
	DecisorCompra      string          `json:"decisor_compra"`
	GanxoVenda         string          `json:"ganxo_venda"`
	RawLLMResponse     json.RawMessage `json:"raw_llm_response,omitempty"`
	AnalyzedAt         time.Time       `json:"analyzed_at"`
}

type ScraperRun struct {
	ID           int       `json:"id"`
	Source       string    `json:"source"`
	Status       string    `json:"status"`
	ItemsFound   int       `json:"items_found"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	RunAt        time.Time `json:"run_at"`
}

type SystemStatusResponse struct {
	SignalsToday int          `json:"signals_today"`
	DailyLimit   int          `json:"daily_limit"`
	CostEUR      float64      `json:"cost_eur"`
	LLMStatus    LLMStatus    `json:"llm_status"`
	Scrapers     []ScraperRun `json:"scrapers"`
}

type LLMStatus struct {
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
}

type SignalListResponse struct {
	Total int      `json:"total"`
	Items []Signal `json:"items"`
}

type ScraperRunResult struct {
	Success        bool   `json:"success"`
	NewOffersFound int    `json:"new_offers_found"`
	AnalyzedCount  int    `json:"analyzed_count"`
	Message        string `json:"message"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type ScraperSettingsResponse struct {
	DeepFetchEnabled bool `json:"deep_fetch_enabled"`
}

type UpdateScraperSettingsRequest struct {
	DeepFetchEnabled bool `json:"deep_fetch_enabled"`
}

type ProgressEvent struct {
	Type          string            `json:"type"` // "start", "progress", "fun_fact", "joke", "analyzing", "completed", "error"
	Step          string            `json:"step,omitempty"`
	Message       string            `json:"message"`
	Progress      int               `json:"progress"` // 0 - 100
	FunFact       string            `json:"fun_fact,omitempty"`
	Joke          string            `json:"joke,omitempty"`
	Result        *ScraperRunResult `json:"result,omitempty"`
	EstimatedSecs int               `json:"estimated_secs,omitempty"`
}

// Evidence representa un fet o senyal d'observació empírica extret de la web o ofertes de feina
type Evidence struct {
	ID                  string     `json:"id"`
	RawContent          string     `json:"raw_content"`
	NormalizedURL       string     `json:"normalized_url"`
	ContentHash         string     `json:"content_hash"`
	Source              string     `json:"source"`
	SourceID            *string    `json:"source_id,omitempty"`
	AuthorOrCompany     *string    `json:"author_or_company,omitempty"`
	ExtractedProcess    *string    `json:"extracted_process,omitempty"`
	TaskDescription     *string    `json:"task_description,omitempty"`
	Frequency           string     `json:"frequency"` // "diària", "setmanal", "mensual", "puntual", "unknown"
	ManualityScore      int        `json:"manuality_score"` // 0..3
	ToolsMentioned      []string   `json:"tools_mentioned"`
	Sector              *string    `json:"sector,omitempty"`
	EvidenceType        *string    `json:"evidence_type,omitempty"`
	SourceEvidenceQuote *string    `json:"source_evidence_quote,omitempty"`
	EvidenceConfidence  string     `json:"evidence_confidence"` // "alta", "mitja", "baixa"
	IsDuplicateOf       *string    `json:"is_duplicate_of,omitempty"`
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ProcessExtractionLLMResult representa el contracte JSON estricte del Prompt 5.1 (Etapa 1: Evidence Extraction)
type ProcessExtractionLLMResult struct {
	ExtractedProcess    string   `json:"extracted_process"`
	TaskDescription     string   `json:"task_description"`
	Frequency           string   `json:"frequency"`
	ManualityScore      int      `json:"manuality_score"`
	ToolsMentioned      []string `json:"tools_mentioned"`
	Sector              string   `json:"sector"`
	EvidenceType        string   `json:"evidence_type"`
	SourceEvidenceQuote string   `json:"source_evidence_quote"`
}

// ProcessNormalized representa un procés de negoci canònic (taxonomia estàndard)
type ProcessNormalized struct {
	ID            string    `json:"id"`
	CanonicalName string    `json:"canonical_name"`
	Category      string    `json:"category"`
	TypicalTools  []string  `json:"typical_tools"`
	Description   *string   `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// PainCluster representa una agrupació d'evidències sobre un mateix dolor operatiu
type PainCluster struct {
	ID             string     `json:"id"`
	ProcessID      string     `json:"process_id"`
	ProcessName    string     `json:"process_name,omitempty"`
	Category       string     `json:"category,omitempty"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	Status         string     `json:"status"` // "emerging", "consolidated", "validated", "discarded"
	EvidenceCount  int        `json:"evidence_count"`
	CompanyCount   int        `json:"company_count"`
	SourceCount    int        `json:"source_count"`
	SectorBreadth  int        `json:"sector_breadth"`
	LastEvidenceAt *time.Time `json:"last_evidence_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PainClusterEvidence representa l'enllaç N:M entre un clúster de dolor i una evidència observada
type PainClusterEvidence struct {
	ClusterID      string    `json:"cluster_id"`
	EvidenceID     string    `json:"evidence_id"`
	RelevanceScore float64   `json:"relevance_score"`
	AddedAt        time.Time `json:"added_at"`
}

// PainClusterWithDetails inclou el clúster amb el llistat d'evidències i les seves cites
type PainClusterWithDetails struct {
	PainCluster
	Evidences []Evidence `json:"evidences"`
}

// PainClusterListResponse per a les respostes de l'API
type PainClusterListResponse struct {
	Total int           `json:"total"`
	Items []PainCluster `json:"items"`
}

// NormalizationLLMResult representa el contracte JSON del Prompt 5.2 (Etapa 2: Process Normalization)
type NormalizationLLMResult struct {
	CanonicalProcessName string   `json:"canonical_process_name"`
	Category             string   `json:"category"`
	TypicalTools         []string `json:"typical_tools"`
	ProcessDescription   string   `json:"process_description"`
	Confidence           string   `json:"confidence"` // "alta", "mitja", "baixa"
}

// ClusteringDecisionLLMResult representa el contracte JSON del Prompt 5.3 (Etapa 3: Pain Clustering Confirmation)
type ClusteringDecisionLLMResult struct {
	Action         string  `json:"action"` // "join_existing" o "create_new"
	TargetClusterID string `json:"target_cluster_id,omitempty"`
	NewClusterTitle string `json:"new_cluster_title,omitempty"`
	NewClusterSummary string `json:"new_cluster_summary,omitempty"`
	RelevanceScore float64 `json:"relevance_score"`
	Reasoning      string  `json:"reasoning"`
}

// Opportunity representa una hipòtesi de producte Micro-SaaS validada sobre un clúster de dolor
type Opportunity struct {
	ID            string            `json:"id"`
	ClusterID     string            `json:"cluster_id"`
	ClusterTitle  string            `json:"cluster_title,omitempty"`
	ProcessName   string            `json:"process_name,omitempty"`
	Category      string            `json:"category,omitempty"`
	Title         string            `json:"title"`
	TargetUser    string            `json:"target_user"`
	BuyerPersona  string            `json:"buyer_persona"`
	CoreWorkflow  string            `json:"core_workflow"`
	ValueProp     string            `json:"value_prop"`
	PricingModel  string            `json:"pricing_model"`
	OutreachHook  string            `json:"outreach_hook"`
	Scores        ScoringDimensions `json:"scores"`
	GlobalScore   float64           `json:"global_score"`
	ViabilityTier string            `json:"viability_tier"` // "excel·lent", "prometedora", "feble", "descartada"
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// OpportunityListResponse per a l'API
type OpportunityListResponse struct {
	Total int           `json:"total"`
	Items []Opportunity `json:"items"`
}

// OpportunitySynthesisLLMResult representa el contracte JSON del Prompt 5.4 (Etapa 4: Opportunity Synthesis)
type OpportunitySynthesisLLMResult struct {
	Title        string `json:"title"`
	TargetUser   string `json:"target_user"`
	BuyerPersona string `json:"buyer_persona"`
	CoreWorkflow string `json:"core_workflow"`
	ValueProp    string `json:"value_prop"`
	PricingModel string `json:"pricing_model"`
	OutreachHook string `json:"outreach_hook"`
}

// MultidimensionalScoringLLMResult representa el contracte JSON del Prompt 5.5 (Etapa 5: Scoring 12 Factors)
type MultidimensionalScoringLLMResult struct {
	Scores ScoringDimensions `json:"scores"`
}




