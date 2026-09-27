package signal_test

import (
	"encoding/json"
	"salesanalizer/backend/internal/signal"
	"testing"
	"time"
)

func TestNormalization_Heuristics(t *testing.T) {
	svc := signal.NewService(nil)

	tests := []struct {
		name             string
		extractedProcess string
		taskDescription  string
		expectedCategory string
		expectedName     string
	}{
		{
			name:             "Quadrants i torns rotatius",
			extractedProcess: "Gestió de torns de fàbrica",
			taskDescription:  "Planificar el quadrant mensual dels 30 operaris",
			expectedCategory: "Planificació de Torns & RRHH",
			expectedName:     "Planificació de quadrants i torns de treball",
		},
		{
			name:             "Albarans i rutes de repartiment",
			extractedProcess: "Picar albarans de transport",
			taskDescription:  "Revisió dels fulls de ruta i albarans de lliurament signats",
			expectedCategory: "Logística & Repartiment",
			expectedName:     "Control de fulls de ruta i albarans de lliurament",
		},
		{
			name:             "Control d'estocs i magatzem",
			extractedProcess: "Inventari de recanvis",
			taskDescription:  "Recompte físic d'estoc al magatzem central",
			expectedCategory: "Magatzem & Inventari",
			expectedName:     "Control d'estocs i inventari de magatzem",
		},
		{
			name:             "Factures i cobraments",
			extractedProcess: "Conciliació de cobraments",
			taskDescription:  "Puntejar factures vençudes i trucades a clients morosos",
			expectedCategory: "Facturació & Tresoreria",
			expectedName:     "Conciliació de factures i cobraments pendents",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev := &signal.Evidence{
				ExtractedProcess: &tc.extractedProcess,
				TaskDescription:  &tc.taskDescription,
				ToolsMentioned:   []string{"Excel"},
			}

			norm := svc.FallbackProcessNormalization(ev)
			if norm == nil {
				t.Fatalf("FallbackProcessNormalization returned nil")
			}

			if norm.Category != tc.expectedCategory {
				t.Errorf("expected Category %q, got %q", tc.expectedCategory, norm.Category)
			}
			if norm.CanonicalProcessName != tc.expectedName {
				t.Errorf("expected CanonicalProcessName %q, got %q", tc.expectedName, norm.CanonicalProcessName)
			}
		})
	}
}

func TestClustering_DecisionHeuristics(t *testing.T) {
	svc := signal.NewService(nil)

	candidateClusters := []signal.PainCluster{
		{
			ID:      "c-1",
			Title:   "Caos en la planificació de quadrants de torns rotatius",
			Summary: "Massa temps dedicat a quadrar torns rotatius i gestionar canvis d'última hora amb Excel i WhatsApp.",
		},
		{
			ID:      "c-2",
			Title:   "Pèrdua d'albarans físics i re-entrada manual",
			Summary: "Els xofers porten albarans en paper que s'han de picar a mà cada tarda.",
		},
	}

	// Cas 1: Evidència similar al cluster c-1 (torns)
	evShifts := &signal.Evidence{
		ExtractedProcess: strPtr("Planificació de torns rotatius en fàbrica"),
		TaskDescription:  strPtr("El responsable dedica 4 hores setmanals a quadrar torns d'operaris amb fulls Excel."),
	}

	decShifts := svc.FallbackClusteringDecision(evShifts, candidateClusters)
	if decShifts.Action != "join_existing" || decShifts.TargetClusterID != "c-1" {
		t.Errorf("expected join_existing to c-1, got action=%s target=%s", decShifts.Action, decShifts.TargetClusterID)
	}

	// Cas 2: Evidència diferent (manteniment d'ascensors)
	evElevators := &signal.Evidence{
		ExtractedProcess: strPtr("Manteniment preventiu d'ascensors"),
		TaskDescription:  strPtr("Revisió tècnica de maquinària d'elevadors i signatures de conformitat."),
	}

	decElevators := svc.FallbackClusteringDecision(evElevators, candidateClusters)
	if decElevators.Action != "create_new" {
		t.Errorf("expected create_new, got action=%s", decElevators.Action)
	}
}

func TestPainClusterModels_JSONSerialization(t *testing.T) {
	now := time.Now()
	desc := "Procés de planificació operativa"

	proc := signal.ProcessNormalized{
		ID:            "pn-123",
		CanonicalName: "Planificació de quadrants i torns de treball",
		Category:      "Planificació de Torns & RRHH",
		TypicalTools:  []string{"Excel", "WhatsApp"},
		Description:   &desc,
		CreatedAt:     now,
	}

	cluster := signal.PainCluster{
		ID:             "pc-456",
		ProcessID:      proc.ID,
		ProcessName:    proc.CanonicalName,
		Category:       proc.Category,
		Title:          "Caos en quadrants rotatius",
		Summary:        "Ineficiència en torns i avisos",
		Status:         "consolidated",
		EvidenceCount:  5,
		CompanyCount:   4,
		SourceCount:    3,
		SectorBreadth:  2,
		LastEvidenceAt: &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	bytes, err := json.Marshal(cluster)
	if err != nil {
		t.Fatalf("error marshaling PainCluster: %v", err)
	}

	var parsed signal.PainCluster
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("error unmarshaling PainCluster: %v", err)
	}

	if parsed.Status != "consolidated" {
		t.Errorf("expected status 'consolidated', got %s", parsed.Status)
	}
	if parsed.EvidenceCount != 5 || parsed.SourceCount != 3 {
		t.Errorf("expected EvidenceCount=5, SourceCount=3, got %d, %d", parsed.EvidenceCount, parsed.SourceCount)
	}
}

func strPtr(s string) *string {
	return &s
}
