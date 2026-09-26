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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type SearXNGScraper struct {
	client           *http.Client
	baseURL          string
	dorks            []string
	deepFetchEnabled bool
	mu               sync.RWMutex
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

// NewSearXNGScraper crea un extractor que fa consultes de dorks d'intenció a SearXNG
func NewSearXNGScraper() *SearXNGScraper {
	return NewSearXNGScraperWithConfig("", nil, false)
}

// NewSearXNGScraperWithClient permet injectar client HTTP i URL personalitzada
func NewSearXNGScraperWithClient(baseURL string, client *http.Client) *SearXNGScraper {
	return NewSearXNGScraperWithConfig(baseURL, client, false)
}

// NewSearXNGScraperWithConfig permet configurar baseURL, client HTTP i mode deep fetch
func NewSearXNGScraperWithConfig(baseURL string, client *http.Client, deepFetch bool) *SearXNGScraper {
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
		client:           client,
		baseURL:          baseURL,
		deepFetchEnabled: deepFetch,
		dorks: []string{
			`site:upwork.com/freelance-jobs OR site:workana.com/jobs "excel" OR "macro" "inventari" OR "torns" OR "cuadrantes"`,
			`site:es.trustpilot.com OR site:capterra.es "Factusol" OR "Sage" OR "A3" "complicat" OR "lent" "magatzem" OR "operaris"`,
			`site:linkedin.com/posts "algú em pot recomanar un programa per" OR "estic buscant un software que"`,
			`"software senzill" OR "busco programa" "gestió de torns" OR "quadrants"`,
			`"estic fart de l'excel" OR "perdem molt de temps" "inventari" OR "estocs"`,
			`"introducció de dades" "albarans" OR "comandes" "empresa"`,
			`"domini d'excel" "control d'estoc" "magatzem" catalunya`,
			`ext:xls OR ext:xlsx "control de presència" OR "quadrant de torns"`,
			`ext:xls OR ext:xlsx "manteniment preventiu" OR "revisió maquinària"`,
			`ext:xls OR ext:xlsx "full de ruta" OR "albarans" "transport"`,
		},
	}
}




func (s *SearXNGScraper) Name() string {
	return "SearXNG (Google Dorks)"
}

func (s *SearXNGScraper) SetDeepFetch(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deepFetchEnabled = enabled
}

func (s *SearXNGScraper) IsDeepFetch() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deepFetchEnabled
}

// randomDelay aplica un retard aleatori d'entre 3 i 7 segons entre peticions de navegació web profunda
func (s *SearXNGScraper) randomDelay(ctx context.Context) error {
	delaySec := 3 + rand.Intn(5) // 3..7 segons
	select {
	case <-time.After(time.Duration(delaySec) * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *SearXNGScraper) fetchDeepContent(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ca,es;q=0.9,en;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Netejar tags de codi i components auxiliars
	doc.Find("script, style, noscript, svg, nav, footer, header, iframe").Each(func(_ int, sel *goquery.Selection) {
		sel.Remove()
	})

	bodyText := doc.Find("body").Text()
	if bodyText == "" {
		bodyText = doc.Text()
	}

	spaceRegex := regexp.MustCompile(`\s+`)
	cleaned := strings.TrimSpace(spaceRegex.ReplaceAllString(bodyText, " "))
	if len(cleaned) < 30 {
		return "", fmt.Errorf("contingut extret massa curt")
	}

	if len(cleaned) > 3500 {
		cleaned = cleaned[:3500] + "..."
	}

	return cleaned, nil
}

func (s *SearXNGScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal

	// Escollim 2 o 3 dorks d'intenció a l'atzar
	dorkPool := make([]string, len(s.dorks))
	copy(dorkPool, s.dorks)
	rand.Shuffle(len(dorkPool), func(i, j int) {
		dorkPool[i], dorkPool[j] = dorkPool[j], dorkPool[i]
	})

	numDorks := 2 + rand.Intn(2) // 2..3 dorks
	if numDorks > len(dorkPool) {
		numDorks = len(dorkPool)
	}
	selectedDorks := dorkPool[:numDorks]

	deepMode := s.IsDeepFetch()
	log.Printf("[SearXNG Scraper] Executant %d Dorks d'intenció a %s (Deep Fetch: %t)...", len(selectedDorks), s.baseURL, deepMode)

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

			// Opció A: Utilitzar snippet directe com a RawText
			rawText := strings.TrimSpace(item.Content)
			if rawText == "" {
				rawText = item.Title
			}

			// Opció B: Deep Fetch (navegació web profunda amb goquery)
			if deepMode {
				_ = s.randomDelay(ctx)
				if ctx.Err() != nil {
					return signals, ctx.Err()
				}
				deepText, err := s.fetchDeepContent(ctx, item.URL)
				if err == nil && deepText != "" {
					rawText = deepText
					log.Printf("[SearXNG DeepFetch] ✅ Extret text complet (%d caràcters) per a %s", len(rawText), item.URL)
				} else {
					// Fallback silenciós al snippet del JSON
					log.Printf("[SearXNG DeepFetch] ℹ️ Fallback a snippet per a %s (motiu: %v)", item.URL, err)
				}
			}

			signals = append(signals, RawSignal{
				SourceURL:  item.URL,
				Title:      item.Title,
				RawText:    rawText,
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


