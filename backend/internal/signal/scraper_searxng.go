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
			`ext:xls OR ext:pdf "control de presència" OR "quadrants de torns"`,
			`ext:xls OR ext:pdf "manteniment preventiu" OR "checklist maquinària"`,
			`ext:xls "control d'estoc" OR "inventari de material"`,
			`ext:xls "full de rutes" OR "comunicat de treball" "operaris"`,
		},
	}
}

func (s *SearXNGScraper) Name() string {
	return "SearXNG (Google Dorks)"
}

func (s *SearXNGScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal

	// Escollim 2 dorks aleatoris a cada execució per rotar i optimitzar
	dorkPool := make([]string, len(s.dorks))
	copy(dorkPool, s.dorks)
	rand.Shuffle(len(dorkPool), func(i, j int) {
		dorkPool[i], dorkPool[j] = dorkPool[j], dorkPool[i]
	})

	selectedDorks := dorkPool
	if len(selectedDorks) > 2 {
		selectedDorks = selectedDorks[:2]
	}

	for _, dork := range selectedDorks {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		searchEndpoint := fmt.Sprintf("%s/search?q=%s&format=json", s.baseURL, url.QueryEscape(dork))

		req, err := http.NewRequestWithContext(ctx, "GET", searchEndpoint, nil)
		if err != nil {
			log.Printf("Avis SearXNG: error creant petició per a dork %q: %v", dork, err)
			continue
		}
		req.Header.Set("User-Agent", "SalesAnalizer-Bot/1.0")

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("Avis SearXNG: instància no accessible a %s (%v)", s.baseURL, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("Avis SearXNG: codi d'estat no OK (%d) per a %s", resp.StatusCode, searchEndpoint)
			continue
		}

		var res searxngResponse
		err = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if err != nil {
			log.Printf("Avis SearXNG: error descodificant JSON: %v", err)
			continue
		}

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
		}
	}

	return signals, nil
}
