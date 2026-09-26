package signal

import (
	"context"
	"fmt"
	"log"
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
			"full de rutes",
			"albarans",
			"manteniment preventiu",
		}
	}
	return &FeinaActivaScraper{
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
		keywords: keywords,
	}
}

func (s *FeinaActivaScraper) Name() string {
	return "Feina Activa (SOC)"
}

// randomDelay aplica un retard aleatori d'entre 3 i 7 segons entre peticions HTTP per evitar bloquejos
func (s *FeinaActivaScraper) randomDelay(ctx context.Context) error {
	delaySec := 3 + rand.Intn(5) // 3..7 segons
	select {
	case <-time.After(time.Duration(delaySec) * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *FeinaActivaScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal

	log.Printf("[Feina Activa SOC] Iniciant rastreig per a %d paraules clau...", len(s.keywords))

	for i, kw := range s.keywords {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		if i > 0 {
			if err := s.randomDelay(ctx); err != nil {
				return signals, err
			}
		}

		searchURL := fmt.Sprintf("https://feinaactiva.gencat.cat/ofertes-de-feina?paraulaClau=%s", url.QueryEscape(kw))
		log.Printf("[Feina Activa SOC] [%d/%d] Cercant paraula clau: %q -> %s", i+1, len(s.keywords), kw, searchURL)

		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			log.Printf("[Feina Activa SOC] ⚠️ Error preparant petició: %v", err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept-Language", "ca,es;q=0.9,en;q=0.8")

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("[Feina Activa SOC] ⚠️ Error HTTP consultant %q: %v", kw, err)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[Feina Activa SOC] ⚠️ Error parsejant HTML per a %q: %v", kw, err)
			continue
		}

		kwSignalsCount := 0
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
			kwSignalsCount++
		})

		// Si el DOM és renderitzat dinàmicament per JS i no hem trobat elements individuals, guardem la cerca d'ofertes com a senyal
		if kwSignalsCount == 0 {
			signals = append(signals, RawSignal{
				SourceURL:  searchURL,
				Title:      fmt.Sprintf("Cerca d'ofertes per: %s", kw),
				RawText:    fmt.Sprintf("Rastreig de llocs de treball per paraula clau '%s' a la petita indústria i serveis a Catalunya (Feina Activa SOC).", kw),
				SignalType: "oferta_feina",
				Source:     "feina_activa",
			})
			log.Printf("[Feina Activa SOC] Afegit senyal de cerca global per a %q (renderitzat JS)", kw)
		} else {
			log.Printf("[Feina Activa SOC] ✅ Trobats %d llocs de treball per a %q", kwSignalsCount, kw)
		}
	}

	log.Printf("[Feina Activa SOC] Resum final: %d senyals recollits en total", len(signals))

	return signals, nil
}

