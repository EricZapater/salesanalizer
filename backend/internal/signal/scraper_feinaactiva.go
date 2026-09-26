package signal

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FeinaActivaScraper struct {
	client   *http.Client
	keywords []string
}

func NewFeinaActivaScraper(keywords []string) *FeinaActivaScraper {
	if len(keywords) == 0 {
		keywords = []string{
			"auxiliar administratiu",
			"control de planta",
			"gestió d'estocs",
			"quadrants",
			"introducció de dades",
			"gestió de rutes",
		}
	}
	return &FeinaActivaScraper{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		keywords: keywords,
	}
}

func (s *FeinaActivaScraper) Name() string {
	return "Feina Activa (SOC)"
}

// randomDelay aplica un retard aleatori entre peticions HTTP per evitar bloquejos
func (s *FeinaActivaScraper) randomDelay(ctx context.Context) error {
	delaySec := 3 + rand.Intn(6) // 3..8 segons
	select {
	case <-time.After(time.Duration(delaySec) * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *FeinaActivaScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal

	for i, kw := range s.keywords {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		// Aplicar retard entre peticions consecutives per evitar bloquejos
		if i > 0 {
			if err := s.randomDelay(ctx); err != nil {
				return signals, err
			}
		}

		searchURL := fmt.Sprintf("https://feinaactiva.gencat.cat/ofertes-de-feina?paraulaClau=%s", url.QueryEscape(kw))

		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept-Language", "ca,es;q=0.9,en;q=0.8")

		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Extreure nodes d'ofertes amb goquery
		doc.Find("a[href*='/oferta/'], a[href*='/ofertes-de-feina/'], .job-item, .card-oferta, article").Each(func(_ int, sel *goquery.Selection) {
			link, exists := sel.Attr("href")
			if !exists {
				linkNode := sel.Find("a")
				link, _ = linkNode.Attr("href")
			}

			if link == "" {
				return
			}

			fullURL := link
			if !strings.HasPrefix(fullURL, "http") {
				fullURL = "https://feinaactiva.gencat.cat" + link
			}

			title := strings.TrimSpace(sel.Find("h2, h3, .job-title, .title").First().Text())
			if title == "" {
				title = strings.TrimSpace(sel.Text())
				if len(title) > 80 {
					title = title[:80] + "..."
				}
			}

			companyText := strings.TrimSpace(sel.Find(".company, .empresa, .job-company").First().Text())
			var company *string
			if companyText != "" {
				company = &companyText
			}

			rawText := strings.TrimSpace(sel.Text())
			if len(rawText) < 20 {
				rawText = fmt.Sprintf("Oferta de feina per a %s al portal Feina Activa de la Generalitat de Catalunya.", title)
			}

			signals = append(signals, RawSignal{
				SourceURL:   fullURL,
				CompanyName: company,
				Title:       title,
				RawText:     rawText,
				SignalType:  "oferta_feina",
				Source:      "feina_activa",
			})
		})

		// Si el DOM és renderitzat dinàmicament per JS, generem un senyal amb la URL viva de la cerca
		if len(signals) == 0 {
			signals = append(signals, RawSignal{
				SourceURL:  searchURL,
				Title:      fmt.Sprintf("Cerca d'ofertes per: %s", kw),
				RawText:    fmt.Sprintf("Rastreig de llocs de treball per paraula clau '%s' a la petita indústria i serveis a Catalunya (Feina Activa SOC).", kw),
				SignalType: "oferta_feina",
				Source:     "feina_activa",
			})
		}
	}

	return signals, nil
}
