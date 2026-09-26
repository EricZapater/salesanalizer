package signal_test

import (
	"context"
	"os"
	"salesanalizer/backend/internal/db"
	"salesanalizer/backend/internal/signal"
	"testing"
	"time"
)

func TestRepository_CRUD(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL / DATABASE_URL no configurada, ometent test d'integració de repository")
	}

	database, err := db.Connect(dbURL)
	if err != nil {
		t.Fatalf("error connectant a BD de test: %v", err)
	}

	// Executar migracions
	if err := database.RunAutoMigrations(); err != nil {
		t.Fatalf("error executant migracions: %v", err)
	}

	repo := signal.NewRepository(database)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testURL := "https://example.com/test-opp-" + time.Now().Format("20060102150405")
	comp := "Empresa Test SL"
	rawSig := signal.RawSignal{
		SourceURL:   testURL,
		CompanyName: &comp,
		Title:       "Test Oferta Integració",
		RawText:     "Descripció de test per a comprovació del repositori",
		SignalType:  "oferta_feina",
		Source:      "test",
	}

	// 1. SaveRawSignal
	id, err := repo.SaveRawSignal(ctx, &rawSig)
	if err != nil {
		t.Fatalf("error a SaveRawSignal: %v", err)
	}
	if id == "" {
		t.Fatalf("esperava ID no buit")
	}

	// 2. ExistsByURL
	exists, err := repo.ExistsByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("error a ExistsByURL: %v", err)
	}
	if !exists {
		t.Fatalf("esperava exists = true per a URL %s", testURL)
	}

	// 3. SaveOpportunityAnalysis
	analysis := signal.OpportunityAnalysis{
		SignalID:           id,
		IneficienciaManual: "Ineficiència detectada de prova",
		PropostaMicroSaas:  "MicroSaaS Test",
		ViabilitatPLGScore: 4,
		DecisorCompra:      "Gerent",
		GanxoVenda:         "Missatge de venda de prova",
		AnalyzedAt:         time.Now(),
	}
	if err := repo.SaveOpportunityAnalysis(ctx, &analysis); err != nil {
		t.Fatalf("error a SaveOpportunityAnalysis: %v", err)
	}

	// 4. GetSignalByID
	sig, err := repo.GetSignalByID(ctx, id)
	if err != nil {
		t.Fatalf("error a GetSignalByID: %v", err)
	}
	if sig == nil {
		t.Fatalf("esperava senyal trobat")
	}
	if sig.SignalType != "oferta_feina" {
		t.Errorf("esperava signal_type 'oferta_feina', obtingut '%s'", sig.SignalType)
	}
	if sig.Analysis == nil || sig.Analysis.ViabilitatPLGScore != 4 {
		t.Errorf("anàlisi invàlida al senyal obtingut")
	}

	// 5. ListSignals
	res, total, err := repo.ListSignals(ctx, "active", 0, 10, 0)
	if err != nil {
		t.Fatalf("error a ListSignals: %v", err)
	}
	if total == 0 || len(res) == 0 {
		t.Errorf("esperava almenys 1 senyal al llistat")
	}

	// 6. DiscardSignal
	if err := repo.DiscardSignal(ctx, id); err != nil {
		t.Fatalf("error a DiscardSignal: %v", err)
	}
}
