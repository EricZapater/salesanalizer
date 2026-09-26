package signal_test

import (
	"context"
	"encoding/json"
	"os"
	"salesanalizer/backend/internal/signal"
	"testing"
	"time"
)



func TestSignalModel_JSONSerialization(t *testing.T) {
	extID := "ext-123"
	company := "Metal·lúrgica BCN SL"
	loc := "Granollers"

	sig := signal.Signal{
		ID:         "550e8400-e29b-41d4-a716-446655440000",
		Source:     "feina_activa",
		SignalType: "oferta_feina",
		ExternalID: &extID,
		Title:      "Auxiliar de magatzem i albarans",
		Company:    &company,
		Location:   &loc,
		URL:        "https://feinaactiva.gencat.cat/ofertes-de-feina/123",
		Status:     "analyzed",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Analysis: &signal.OpportunityAnalysis{
			ID:                 "660e8400-e29b-41d4-a716-446655440000",
			SignalID:           "550e8400-e29b-41d4-a716-446655440000",
			IneficienciaManual: "Picar albarans a mà",
			PropostaMicroSaas:  "ScanAlbara OCR",
			ViabilitatPLGScore: 5,
			DecisorCompra:      "Cap de Trànsit",
			GanxoVenda:         "Digitalitza albarans amb 1 foto.",
			AnalyzedAt:         time.Now(),
		},
	}

	bytes, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("error serialitzant a JSON: %v", err)
	}

	var parsed signal.Signal
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("error deserialitzant de JSON: %v", err)
	}

	if parsed.SignalType != "oferta_feina" {
		t.Errorf("esperava signal_type 'oferta_feina', obtingut '%s'", parsed.SignalType)
	}
	if parsed.Analysis == nil || parsed.Analysis.ViabilitatPLGScore != 5 {
		t.Errorf("esperava score 5, obtingut %+v", parsed.Analysis)
	}
}

func TestService_GroqAnalysis(t *testing.T) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		t.Skip("GROQ_API_KEY no configurada a l'entorn, ometent test d'integració amb Groq")
	}

	svc := signal.NewService(nil)

	// Cas 1: Negoci B2B amb dolor clar (Excel per quadrants)
	analysisB2B, err := svc.AnalyzeSignal(context.Background(), "Responsable de torns", "Necessitem algú per gestionar quadrants de 40 operaris en torns rotatius amb fulls Excel i avisar per WhatsApp.", "oferta_feina")
	if err != nil {
		t.Fatalf("error a analyzeWithGroq: %v", err)
	}

	if analysisB2B == nil {
		t.Fatalf("esperava anàlisi no nul·la")
	}

	if analysisB2B.ViabilitatPLGScore < 3 {
		t.Errorf("esperava score >= 3 per a dolor B2B clar, obtingut: %d", analysisB2B.ViabilitatPLGScore)
	}
	if analysisB2B.PropostaMicroSaas == "" {
		t.Errorf("esperava proposta micro-saas generada per IA")
	}

	// Cas 2: Notícia general / debat no B2B -> ha de ser descartat amb Score 1
	analysisNonB2B, err := svc.AnalyzeSignal(context.Background(), "Cronos: pel·lícules en català", "Aquest cap de setmana fan 36 pel·lícules en català als cinemes de Catalunya.", "queixa_forum")
	if err != nil {
		t.Fatalf("error a analyzeWithGroq no-b2b: %v", err)
	}

	if analysisNonB2B.ViabilitatPLGScore > 2 {
		t.Errorf("esperava score <= 2 per a notícia de cinema no B2B, obtingut: %d", analysisNonB2B.ViabilitatPLGScore)
	}
}



