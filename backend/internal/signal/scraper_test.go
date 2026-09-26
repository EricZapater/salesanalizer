package signal_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
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

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestSearXNGScraper_FetchWithMockTransport(t *testing.T) {
	mockJSON := `{
		"query": "ext:xls control de presencia",
		"number_of_results": 1,
		"results": [
			{
				"url": "https://example.cat/plantilla-torns.xls",
				"title": "Plantilla Excel de control horari i torns per a operaris",
				"content": "Full de càlcul per al registre diari de presència i quadrants de torns rotatius a fàbrica.",
				"engine": "google"
			}
		]
	}`

	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(mockJSON)),
			}
		}),
	}

	scraper := signal.NewSearXNGScraperWithClient("http://searxng.internal", mockClient)
	if scraper.Name() != "SearXNG (Google Dorks)" {
		t.Errorf("nom esperat 'SearXNG (Google Dorks)', obtingut: %s", scraper.Name())
	}

	signals, err := scraper.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch a SearXNG ha fallat: %v", err)
	}

	if len(signals) == 0 {
		t.Fatalf("esperava com a mínim 1 senyal, obtinguts 0")
	}

	first := signals[0]
	if first.SignalType != "searxng_dork" {
		t.Errorf("signal_type esperat 'searxng_dork', obtingut '%s'", first.SignalType)
	}
	if first.Source != "searxng" {
		t.Errorf("source esperada 'searxng', obtingut '%s'", first.Source)
	}
	if first.SourceURL != "https://example.cat/plantilla-torns.xls" {
		t.Errorf("URL incorrecta: %s", first.SourceURL)
	}
}
