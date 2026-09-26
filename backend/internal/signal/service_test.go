package signal_test

import (
	"encoding/json"
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
