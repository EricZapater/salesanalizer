package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type SearXNGScraper struct {
	client  *http.Client
	baseURL string
	dorks   []string
}

type searxngResponse struct {
	Query           string `json:"query"`
	NumberOfResults int    `json:"number_of_results"`
	Results         []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Engine  string `json:"engine"`
	} `json:"results"`
}

// NewSearXNGScraper crea un extractor que fa consultes de Google Dorks a una instància interna de SearXNG
func NewSearXNGScraper() *SearXNGScraper {
	return NewSearXNGScraperWithClient("", nil)
}

// NewSearXNGScraperWithClient permet injectar client HTTP i URL personalitzada (útil per a tests i configuració)
func NewSearXNGScraperWithClient(baseURL string, client *http.Client) *SearXNGScraper {
	if baseURL == "" {
		baseURL = os.Getenv("SEARXNG_URL")
		if baseURL == "" {
			baseURL = "http://searxng:8080"
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if client == nil {
		client = &http.Client{
			Timeout: 20 * time.Second,
		}
	}

	return &SearXNGScraper{
		client:  client,
		baseURL: baseURL,
		dorks: []string{
			`ext:xls OR ext:xlsx "control de presència" OR "quadrant de torns" OR "control horari"`,
			`ext:xls OR ext:xlsx "manteniment preventiu" OR "revisió maquinària" OR "part de treball"`,
			`ext:xls OR ext:xlsx "control d'estoc" OR "inventari de material" OR "fitxa de magatzem"`,
			`ext:xls OR ext:xlsx "full de ruta" OR "albarans pendents" OR "repartiment transport"`,
			`filetype:pdf OR filetype:xls "sol·licitud de vacances" OR "petició dies d'assumptes propis" "empresa"`,
			`filetype:xls "comunicat d'incidències" OR "part d'avaries" "taller"`,
		},
	}
}

func (s *SearXNGScraper) Name() string {
	return "SearXNG (Google Dorks)"
}

func (s *SearXNGScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal

	// Escollim 3 dorks aleatoris a cada execució per rotar i optimitzar
	dorkPool := make([]string, len(s.dorks))
	copy(dorkPool, s.dorks)
	rand.Shuffle(len(dorkPool), func(i, j int) {
		dorkPool[i], dorkPool[j] = dorkPool[j], dorkPool[i]
	})

	selectedDorks := dorkPool
	if len(selectedDorks) > 3 {
		selectedDorks = selectedDorks[:3]
	}

	log.Printf("[SearXNG Scraper] Executant %d Dorks seleccionats a la instància %s...", len(selectedDorks), s.baseURL)

	for i, dork := range selectedDorks {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		searchEndpoint := fmt.Sprintf("%s/search?q=%s&format=json", s.baseURL, url.QueryEscape(dork))
		log.Printf("[SearXNG Scraper] [%d/%d] Querying dork: %q -> %s", i+1, len(selectedDorks), dork, searchEndpoint)

		req, err := http.NewRequestWithContext(ctx, "GET", searchEndpoint, nil)
		if err != nil {
			log.Printf("[SearXNG Scraper] ⚠️ Error creant petició per a dork %q: %v", dork, err)
			continue
		}
		req.Header.Set("User-Agent", "SalesAnalizer-Prospector/1.0")
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.Header.Set("X-Real-IP", "127.0.0.1")
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("[SearXNG Scraper] ⚠️ Instància SearXNG no accessible a %s (%v)", s.baseURL, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("[SearXNG Scraper] ⚠️ Codi d'estat no OK (%d) per a dork %q", resp.StatusCode, dork)
			continue
		}

		var res searxngResponse
		err = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if err != nil {
			log.Printf("[SearXNG Scraper] ⚠️ Error descodificant JSON per a dork %q: %v", dork, err)
			continue
		}

		dorkCount := 0
		for _, item := range res.Results {
			if item.URL == "" || item.Title == "" {
				continue
			}

			content := strings.TrimSpace(item.Content)
			if content == "" {
				content = item.Title
			}

			signals = append(signals, RawSignal{
				SourceURL:  item.URL,
				Title:      item.Title,
				RawText:    content,
				SignalType: "searxng_dork",
				Source:     "searxng",
			})
			dorkCount++
		}

		log.Printf("[SearXNG Scraper] ✅ Dork %q: trobats %d resultats (motor SearXNG)", dork, dorkCount)
	}

	log.Printf("[SearXNG Scraper] Resum final: %d senyals recollits en total", len(signals))

	return signals, nil
}

