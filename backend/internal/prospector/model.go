package prospector

import (
	"encoding/json"
	"time"
)

type JobOffer struct {
	ID         string               `json:"id"`
	Source     string               `json:"source"`
	ExternalID *string              `json:"external_id,omitempty"`
	Title      string               `json:"title"`
	Company    *string              `json:"company,omitempty"`
	Location   *string              `json:"location,omitempty"`
	URL        string               `json:"url"`
	RawText    string               `json:"raw_text,omitempty"`
	Status     string               `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	Analysis   *OpportunityAnalysis `json:"analysis,omitempty"`
}

type OpportunityAnalysis struct {
	ID                 string          `json:"id,omitempty"`
	JobOfferID         string          `json:"job_offer_id,omitempty"`
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
	SignalsToday int            `json:"signals_today"`
	DailyLimit   int            `json:"daily_limit"`
	CostEUR      float64        `json:"cost_eur"`
	LLMStatus    LLMStatus      `json:"llm_status"`
	Scrapers     []ScraperRun   `json:"scrapers"`
}

type LLMStatus struct {
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
}

type AnalyzeURLRequest struct {
	URL string `json:"url" binding:"required"`
}

type OfferListResponse struct {
	Total int        `json:"total"`
	Items []JobOffer `json:"items"`
}

type ScraperRunResult struct {
	Success        bool   `json:"success"`
	NewOffersFound int    `json:"new_offers_found"`
	AnalyzedCount  int    `json:"analyzed_count"`
	Message        string `json:"message"`
}
