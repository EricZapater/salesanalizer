package signal_test

import (
	"context"
	"salesanalizer/backend/internal/signal"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
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

func TestRSSParser_Logic(t *testing.T) {
	sampleRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Reddit SmallBusiness Sample</title>
    <link>https://reddit.com/r/smallbusiness</link>
    <description>Sample feed</description>
    <item>
      <title>Com calculeu els torns dels empleats?</title>
      <link>https://reddit.com/r/smallbusiness/comments/123/torns</link>
      <description>&lt;p&gt;Estic fart de fer servir fulls Excel i avisar la gent per WhatsApp a última hora.&lt;/p&gt;</description>
      <author>gerent_pime</author>
    </item>
  </channel>
</rss>`

	fp := gofeed.NewParser()
	feed, err := fp.Parse(strings.NewReader(sampleRSS))
	if err != nil {
		t.Fatalf("error parsejant feed RSS: %v", err)
	}

	if len(feed.Items) != 1 {
		t.Fatalf("esperava 1 element al feed, obtinguts %d", len(feed.Items))
	}

	item := feed.Items[0]
	if item.Title != "Com calculeu els torns dels empleats?" {
		t.Errorf("títol incorrecte: %s", item.Title)
	}
	if item.Link != "https://reddit.com/r/smallbusiness/comments/123/torns" {
		t.Errorf("link incorrecte: %s", item.Link)
	}
}

func TestGoogleSearchScraper_Initialization(t *testing.T) {
	scraper := signal.NewGoogleSearchScraper()
	if scraper.Name() != "Google Custom Search (Dorks)" {
		t.Errorf("nom esperat 'Google Custom Search (Dorks)', obtingut: %s", scraper.Name())
	}

	// Without API keys, Fetch should return nil gracefully without error
	signals, err := scraper.Fetch(context.Background())
	if err != nil {
		t.Errorf("Fetch sense claus no hauria de fallar: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("esperava 0 senyals sense claus d'API, obtinguts %d", len(signals))
	}
}
