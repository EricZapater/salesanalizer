package signal_test

import (
	"encoding/json"
	"salesanalizer/backend/internal/signal"
	"testing"
	"time"
)

func TestCalculateGlobalScore_FormulaAndTiers(t *testing.T) {
	// Cas 1: Puntuació màxima (tot 5) -> Global 5.00, Tier "excel·lent"
	perfectScores := &signal.ScoringDimensions{
		PainIntensity:            signal.FactorScore{Score: 5, Justification: "Dolor crític"},
		Urgency:                  signal.FactorScore{Score: 5, Justification: "Urgent"},
		BudgetDiscretion:         signal.FactorScore{Score: 5, Justification: "Decisor directe"},
		MarketReach:              signal.FactorScore{Score: 5, Justification: "Gran mercat"},
		ImplementationSimplicity: signal.FactorScore{Score: 5, Justification: "Molt fàcil"},
		SelfOnboarding:           signal.FactorScore{Score: 5, Justification: "Directe"},
		ZeroIntegrations:         signal.FactorScore{Score: 5, Justification: "0 integracions"},
		NoOCRNoHardware:          signal.FactorScore{Score: 5, Justification: "100% digital"},
		RetentionStickiness:      signal.FactorScore{Score: 5, Justification: "Diari"},
		NicheCompetition:         signal.FactorScore{Score: 5, Justification: "Sense competidors"},
		WillingnessToPay:         signal.FactorScore{Score: 5, Justification: "Clar ROI"},
		ScalabilityReach:         signal.FactorScore{Score: 5, Justification: "Molt escalable"},
	}

	score, tier := signal.CalculateGlobalScore(perfectScores)
	if score != 5.00 {
		t.Errorf("expected score 5.00, got %.2f", score)
	}
	if tier != "excel·lent" {
		t.Errorf("expected tier 'excel·lent', got %s", tier)
	}

	// Cas 2: Puntuació equilibrada bona -> G1=4, G2=4, G3=4 -> Global 4.00, Tier "excel·lent"
	goodScores := &signal.ScoringDimensions{
		PainIntensity:            signal.FactorScore{Score: 4, Justification: "G1"},
		Urgency:                  signal.FactorScore{Score: 4, Justification: "G1"},
		BudgetDiscretion:         signal.FactorScore{Score: 4, Justification: "G1"},
		MarketReach:              signal.FactorScore{Score: 4, Justification: "G1"},
		ImplementationSimplicity: signal.FactorScore{Score: 4, Justification: "G2"},
		SelfOnboarding:           signal.FactorScore{Score: 4, Justification: "G2"},
		ZeroIntegrations:         signal.FactorScore{Score: 4, Justification: "G2"},
		NoOCRNoHardware:          signal.FactorScore{Score: 4, Justification: "G2"},
		RetentionStickiness:      signal.FactorScore{Score: 4, Justification: "G3"},
		NicheCompetition:         signal.FactorScore{Score: 4, Justification: "G3"},
		WillingnessToPay:         signal.FactorScore{Score: 4, Justification: "G3"},
		ScalabilityReach:         signal.FactorScore{Score: 4, Justification: "G3"},
	}

	scoreGood, tierGood := signal.CalculateGlobalScore(goodScores)
	if scoreGood != 4.00 {
		t.Errorf("expected score 4.00, got %.2f", scoreGood)
	}
	if tierGood != "excel·lent" {
		t.Errorf("expected tier 'excel·lent', got %s", tierGood)
	}

	// Cas 3: Puntuació mitjana -> G1=3, G2=3, G3=3 -> Global 3.00, Tier "prometedora"
	promisingScores := &signal.ScoringDimensions{
		PainIntensity:            signal.FactorScore{Score: 3, Justification: "G1"},
		Urgency:                  signal.FactorScore{Score: 3, Justification: "G1"},
		BudgetDiscretion:         signal.FactorScore{Score: 3, Justification: "G1"},
		MarketReach:              signal.FactorScore{Score: 3, Justification: "G1"},
		ImplementationSimplicity: signal.FactorScore{Score: 3, Justification: "G2"},
		SelfOnboarding:           signal.FactorScore{Score: 3, Justification: "G2"},
		ZeroIntegrations:         signal.FactorScore{Score: 3, Justification: "G2"},
		NoOCRNoHardware:          signal.FactorScore{Score: 3, Justification: "G2"},
		RetentionStickiness:      signal.FactorScore{Score: 3, Justification: "G3"},
		NicheCompetition:         signal.FactorScore{Score: 3, Justification: "G3"},
		WillingnessToPay:         signal.FactorScore{Score: 3, Justification: "G3"},
		ScalabilityReach:         signal.FactorScore{Score: 3, Justification: "G3"},
	}

	scoreProm, tierProm := signal.CalculateGlobalScore(promisingScores)
	if scoreProm != 3.00 {
		t.Errorf("expected score 3.00, got %.2f", scoreProm)
	}
	if tierProm != "prometedora" {
		t.Errorf("expected tier 'prometedora', got %s", tierProm)
	}

	// Cas 4: Puntuació feble -> G1=2, G2=2, G3=2 -> Global 2.00, Tier "feble"
	weakScores := &signal.ScoringDimensions{
		PainIntensity:            signal.FactorScore{Score: 2, Justification: "G1"},
		Urgency:                  signal.FactorScore{Score: 2, Justification: "G1"},
		BudgetDiscretion:         signal.FactorScore{Score: 2, Justification: "G1"},
		MarketReach:              signal.FactorScore{Score: 2, Justification: "G1"},
		ImplementationSimplicity: signal.FactorScore{Score: 2, Justification: "G2"},
		SelfOnboarding:           signal.FactorScore{Score: 2, Justification: "G2"},
		ZeroIntegrations:         signal.FactorScore{Score: 2, Justification: "G2"},
		NoOCRNoHardware:          signal.FactorScore{Score: 2, Justification: "G2"},
		RetentionStickiness:      signal.FactorScore{Score: 2, Justification: "G3"},
		NicheCompetition:         signal.FactorScore{Score: 2, Justification: "G3"},
		WillingnessToPay:         signal.FactorScore{Score: 2, Justification: "G3"},
		ScalabilityReach:         signal.FactorScore{Score: 2, Justification: "G3"},
	}

	scoreWeak, tierWeak := signal.CalculateGlobalScore(weakScores)
	if scoreWeak != 2.00 {
		t.Errorf("expected score 2.00, got %.2f", scoreWeak)
	}
	if tierWeak != "feble" {
		t.Errorf("expected tier 'feble', got %s", tierWeak)
	}

	// Cas 5: Puntuació inviable / descartada -> G1=1, G2=1, G3=1 -> Global 1.00, Tier "descartada"
	discardedScores := &signal.ScoringDimensions{
		PainIntensity:            signal.FactorScore{Score: 1, Justification: "G1"},
		Urgency:                  signal.FactorScore{Score: 1, Justification: "G1"},
		BudgetDiscretion:         signal.FactorScore{Score: 1, Justification: "G1"},
		MarketReach:              signal.FactorScore{Score: 1, Justification: "G1"},
		ImplementationSimplicity: signal.FactorScore{Score: 1, Justification: "G2"},
		SelfOnboarding:           signal.FactorScore{Score: 1, Justification: "G2"},
		ZeroIntegrations:         signal.FactorScore{Score: 1, Justification: "G2"},
		NoOCRNoHardware:          signal.FactorScore{Score: 1, Justification: "G2"},
		RetentionStickiness:      signal.FactorScore{Score: 1, Justification: "G3"},
		NicheCompetition:         signal.FactorScore{Score: 1, Justification: "G3"},
		WillingnessToPay:         signal.FactorScore{Score: 1, Justification: "G3"},
		ScalabilityReach:         signal.FactorScore{Score: 1, Justification: "G3"},
	}

	scoreDisc, tierDisc := signal.CalculateGlobalScore(discardedScores)
	if scoreDisc != 1.00 {
		t.Errorf("expected score 1.00, got %.2f", scoreDisc)
	}
	if tierDisc != "descartada" {
		t.Errorf("expected tier 'descartada', got %s", tierDisc)
	}
}

func TestOpportunityModel_JSONSerialization(t *testing.T) {
	now := time.Now()
	opp := signal.Opportunity{
		ID:           "opp-789",
		ClusterID:    "pc-456",
		ClusterTitle: "Dolor en quadrants rotatius",
		ProcessName:  "Planificació de quadrants i torns de treball",
		Category:     "Planificació de Torns & RRHH",
		Title:        "TornsDirect",
		TargetUser:   "Cap de torn / Responsable d'operacions",
		BuyerPersona: "Gerent de planta",
		CoreWorkflow: "Formulari web mòbil per assignar torns i confirmar canvis automàticament.",
		ValueProp:    "Elimina el caos d'Excel i estalvia 6 hores setmanals de gestió.",
		PricingModel: "39€/mes per empresa",
		OutreachHook: "Com esteu gestionant les baixes d'última hora als torns dels 30 operaris?",
		Scores: signal.ScoringDimensions{
			PainIntensity:            signal.FactorScore{Score: 4, Justification: "Dolor diari"},
			Urgency:                  signal.FactorScore{Score: 4, Justification: "Molts canvis"},
			BudgetDiscretion:         signal.FactorScore{Score: 5, Justification: "Pressupost petit"},
			MarketReach:              signal.FactorScore{Score: 4, Justification: "Totes les fàbriques"},
			ImplementationSimplicity: signal.FactorScore{Score: 5, Justification: "1 taula i 1 vista"},
			SelfOnboarding:           signal.FactorScore{Score: 4, Justification: "Ràpid"},
			ZeroIntegrations:         signal.FactorScore{Score: 5, Justification: "Autònom"},
			NoOCRNoHardware:          signal.FactorScore{Score: 5, Justification: "Web pura"},
			RetentionStickiness:      signal.FactorScore{Score: 4, Justification: "Ús setmanal"},
			NicheCompetition:         signal.FactorScore{Score: 4, Justification: "Buit al mercat PIME"},
			WillingnessToPay:         signal.FactorScore{Score: 4, Justification: "39€ és marginal"},
			ScalabilityReach:         signal.FactorScore{Score: 4, Justification: "Indústries i hospitals"},
		},
		GlobalScore:   4.35,
		ViabilityTier: "excel·lent",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	bytes, err := json.Marshal(opp)
	if err != nil {
		t.Fatalf("error serialitzant Opportunity a JSON: %v", err)
	}

	var parsed signal.Opportunity
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("error deserialitzant Opportunity de JSON: %v", err)
	}

	if parsed.Title != "TornsDirect" {
		t.Errorf("expected Title 'TornsDirect', got %s", parsed.Title)
	}
	if parsed.GlobalScore != 4.35 {
		t.Errorf("expected GlobalScore 4.35, got %.2f", parsed.GlobalScore)
	}
	if parsed.Scores.ImplementationSimplicity.Score != 5 {
		t.Errorf("expected ImplementationSimplicity=5, got %d", parsed.Scores.ImplementationSimplicity.Score)
	}
}

func TestFallbackOpportunitySynthesisAndScoring(t *testing.T) {
	svc := signal.NewService(nil)

	cluster := &signal.PainClusterWithDetails{
		PainCluster: signal.PainCluster{
			ID:            "pc-test",
			Title:         "Gestió manual de fulls de ruta i albarans",
			ProcessName:   "Control de fulls de ruta i albarans de lliurament",
			Category:      "Logística & Repartiment",
			Summary:       "Xofers que porten albarans físics i s'han de repicar a l'oficina.",
			EvidenceCount: 3,
			CompanyCount:  3,
			SourceCount:   2,
		},
		Evidences: []signal.Evidence{
			{
				RawContent:          "Empresa de transports cerca administratiu per albarans en paper.",
				Source:              "feina_activa",
				ExtractedProcess:    strPtr("Control d'albarans en paper"),
				SourceEvidenceQuote: strPtr("albarans en paper"),
			},
		},
	}

	synth := svc.FallbackOpportunitySynthesis(cluster)
	if synth == nil || synth.Title == "" {
		t.Fatalf("FallbackOpportunitySynthesis ha retornat nil o buit")
	}

	scores := svc.FallbackOpportunityScoring(synth, cluster)
	if scores == nil {
		t.Fatalf("FallbackOpportunityScoring ha retornat nil")
	}

	globalScore, tier := signal.CalculateGlobalScore(scores)
	if globalScore < 3.5 {
		t.Errorf("expected high score for good fallback candidate, got %.2f", globalScore)
	}
	if tier != "excel·lent" && tier != "prometedora" {
		t.Errorf("expected tier 'excel·lent' or 'prometedora', got %s", tier)
	}
}
