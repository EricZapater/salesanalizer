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

func TestEvidenceModel_JSONSerialization(t *testing.T) {
	company := "Transports i Logística Girona SL"
	sector := "Logística"
	proc := "Control de rutes i albarans"
	task := "Revisió d'albarans físics i actualització de fulls Excel"
	quote := "gestió diària d'albarans en paper i fulls Excel"

	ev := signal.Evidence{
		ID:                  "770e8400-e29b-41d4-a716-446655440000",
		RawContent:          "Empresa de transports cerca administratiu per a la gestió diària d'albarans en paper i fulls Excel.",
		NormalizedURL:       "https://feinaactiva.gencat.cat/oferta/1234",
		ContentHash:         "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Source:              "feina_activa",
		AuthorOrCompany:     &company,
		ExtractedProcess:    &proc,
		TaskDescription:     &task,
		Frequency:           "diària",
		ManualityScore:      3,
		ToolsMentioned:      []string{"Excel", "Albarans en paper"},
		Sector:              &sector,
		EvidenceConfidence:  "alta",
		SourceEvidenceQuote: &quote,
		CreatedAt:           time.Now(),
	}

	bytes, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("error serialitzant Evidence a JSON: %v", err)
	}

	var parsed signal.Evidence
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("error deserialitzant Evidence de JSON: %v", err)
	}

	if parsed.ManualityScore != 3 {
		t.Errorf("esperava manuality_score 3, obtingut %d", parsed.ManualityScore)
	}
	if parsed.EvidenceConfidence != "alta" {
		t.Errorf("esperava confidence 'alta', obtingut '%s'", parsed.EvidenceConfidence)
	}
	if len(parsed.ToolsMentioned) != 2 {
		t.Errorf("esperava 2 eines, obtingut %d", len(parsed.ToolsMentioned))
	}
}

func TestFallbackProcessExtraction(t *testing.T) {
	svc := signal.NewService(nil)

	content := "Cercador de dades: tasques administratives de revisió de quadrants de torns amb fulls Excel i avisos per WhatsApp als xofers."
	title := "Administratiu de Trànsit"

	res := svc.FallbackProcessExtraction(content, "oferta_feina", title)
	if res == nil {
		t.Fatal("esperava resultat d'extracció no nul")
	}

	if res.ManualityScore < 2 {
		t.Errorf("esperava manuality score >= 2 per a contingut amb Excel i WhatsApp, obtingut %d", res.ManualityScore)
	}

	foundExcel := false
	for _, tool := range res.ToolsMentioned {
		if tool == "Excel" {
			foundExcel = true
			break
		}
	}
	if !foundExcel {
		t.Errorf("esperava trobar Excel a ToolsMentioned, obtingut %v", res.ToolsMentioned)
	}

	// Comprovar que la cita generada és vàlida en el contingut
	if !signal.VerifyEvidenceQuote(content, res.SourceEvidenceQuote) {
		t.Errorf("la cita de fallback no és present al contingut original: %q", res.SourceEvidenceQuote)
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



