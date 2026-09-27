package signal

import (
	"math"
)

// FactorScore representa la puntuació d'una dimensió individual amb la seva justificació
type FactorScore struct {
	Score         int    `json:"score"`         // 0..5
	Justification string `json:"justification"` // Raonament factual
}

// ScoringDimensions agrupa els 12 factors d'avaluació objectiva d'oportunitats Micro-SaaS
type ScoringDimensions struct {
	// Grup 1: Dolor i Demanda (Pes: 35%)
	PainIntensity    FactorScore `json:"pain_intensity"`
	Urgency          FactorScore `json:"urgency"`
	BudgetDiscretion FactorScore `json:"budget_discretion"`
	MarketReach      FactorScore `json:"market_reach"`

	// Grup 2: Viabilitat Tècnica & PLG (Pes: 40%)
	ImplementationSimplicity FactorScore `json:"implementation_simplicity"`
	SelfOnboarding           FactorScore `json:"self_onboarding"`
	ZeroIntegrations         FactorScore `json:"zero_integrations"`
	NoOCRNoHardware          FactorScore `json:"no_ocr_no_hardware"`

	// Grup 3: Negoci i Competició (Pes: 25%)
	RetentionStickiness FactorScore `json:"retention_stickiness"`
	NicheCompetition    FactorScore `json:"niche_competition"`
	WillingnessToPay    FactorScore `json:"willingness_to_pay"`
	ScalabilityReach    FactorScore `json:"scalability_reach"`
}

// CalculateGlobalScore aplica la fórmula ponderada dels 12 factors (escala 0.0 - 5.0)
func CalculateGlobalScore(s *ScoringDimensions) (float64, string) {
	if s == nil {
		return 0.0, "descartada"
	}

	clamp := func(val int) float64 {
		if val < 0 {
			return 0.0
		}
		if val > 5 {
			return 5.0
		}
		return float64(val)
	}

	// Grup 1: Dolor i Demanda (35%)
	g1 := (clamp(s.PainIntensity.Score) + clamp(s.Urgency.Score) + clamp(s.BudgetDiscretion.Score) + clamp(s.MarketReach.Score)) / 4.0

	// Grup 2: Viabilitat Tècnica & PLG (40%)
	g2 := (clamp(s.ImplementationSimplicity.Score) + clamp(s.SelfOnboarding.Score) + clamp(s.ZeroIntegrations.Score) + clamp(s.NoOCRNoHardware.Score)) / 4.0

	// Grup 3: Negoci i Competició (25%)
	g3 := (clamp(s.RetentionStickiness.Score) + clamp(s.NicheCompetition.Score) + clamp(s.WillingnessToPay.Score) + clamp(s.ScalabilityReach.Score)) / 4.0

	global := (0.35 * g1) + (0.40 * g2) + (0.25 * g3)
	// Arrodonir a 2 decimals
	global = math.Round(global*100) / 100

	var tier string
	if global >= 4.0 {
		tier = "excel·lent"
	} else if global >= 3.0 {
		tier = "prometedora"
	} else if global >= 2.0 {
		tier = "feble"
	} else {
		tier = "descartada"
	}

	return global, tier
}
