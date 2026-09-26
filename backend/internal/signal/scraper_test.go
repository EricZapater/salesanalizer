package signal_test

import (
	"context"
	"salesanalizer/backend/internal/signal"
	"testing"
)

type MockScraper struct {
	name string
}

func (m *MockScraper) Name() string {
	return m.name
}

func (m *MockScraper) Fetch(ctx context.Context) ([]signal.RawSignal, error) {
	return []signal.RawSignal{
		{
			SourceURL:  "https://example.com/test-signal",
			Title:      "Auxiliar de gestió de torns",
			RawText:    "Tasca manual de fer quadrants en fulls Excel cada setmana.",
			SignalType: "oferta_feina",
			Source:     m.name,
		},
	}, nil
}

func TestStrategyPattern_RegisterScraper(t *testing.T) {
	svc := signal.NewService(nil)
	initialLimit := svc.GetDailyLimit()
	if initialLimit != 50 {
		t.Errorf("esperava límit diari de 50, obtingut %d", initialLimit)
	}

	mock := &MockScraper{name: "TestForum"}
	svc.RegisterScraper(mock)

	// Fetch from mock
	signals, err := mock.Fetch(context.Background())
	if err != nil {
		t.Fatalf("error a fetch del mock: %v", err)
	}
	if len(signals) != 1 {
		t.Errorf("esperava 1 senyal, obtinguts %d", len(signals))
	}
	if signals[0].SignalType != "oferta_feina" {
		t.Errorf("esperava tipus oferta_feina, obtingut %s", signals[0].SignalType)
	}
}
