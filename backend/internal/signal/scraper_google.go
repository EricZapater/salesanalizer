package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"
)

var dorkIndex uint64

type GoogleCustomSearchScraper struct {
	client *http.Client
	dorks  []string
}

type googleSearchResult struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func NewGoogleSearchScraper() *GoogleCustomSearchScraper {
	return &GoogleCustomSearchScraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		dorks: []string{
			`site:es ext:xls OR ext:pdf "control de presència" OR "quadrants de torns"`,
			`site:es ext:xls OR ext:pdf "manteniment preventiu" OR "checklist maquinària"`,
			`site:es ext:xls "control d'estoc" OR "inventari de material"`,
			`site:es ext:xls "full de rutes" OR "comunicat de treball" "operaris"`,
			`site:es ext:xls OR ext:pdf "plantilla torns" OR "registre horari"`,
			`site:es ext:xls "control d'albarans" OR "comandes pendents"`,
		},
	}
}

func (s *GoogleCustomSearchScraper) Name() string {
	return "Google Custom Search (Dorks)"
}

func (s *GoogleCustomSearchScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	cx := os.Getenv("GOOGLE_SEARCH_CX")
	if cx == "" {
		cx = os.Getenv("GOOGLE_CSE_CX")
	}

	if apiKey == "" || cx == "" {
		log.Println("Google Custom Search no configurat (manca GOOGLE_SEARCH_API_KEY o GOOGLE_SEARCH_CX). Ometent extractor.")
		return nil, nil
	}

	var signals []RawSignal
	maxCalls := 5
	callsCount := 0

	totalDorks := len(s.dorks)
	startIdx := int(atomic.AddUint64(&dorkIndex, 1) % uint64(totalDorks))

	for i := 0; i < totalDorks && callsCount < maxCalls; i++ {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		currentDork := s.dorks[(startIdx+i)%totalDorks]
		searchEndpoint := fmt.Sprintf(
			"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=10",
			url.QueryEscape(apiKey),
			url.QueryEscape(cx),
			url.QueryEscape(currentDork),
		)

		req, err := http.NewRequestWithContext(ctx, "GET", searchEndpoint, nil)
		if err != nil {
			continue
		}

		resp, err := s.client.Do(req)
		callsCount++
		if err != nil {
			log.Printf("Avis Google Search HTTP error per dork %q: %v", currentDork, err)
			continue
		}

		if resp.StatusCode == http.StatusForbidden {
			var errRes googleSearchResult
			_ = json.NewDecoder(resp.Body).Decode(&errRes)
			resp.Body.Close()
			log.Println("Avis Google Search (403 Forbidden): L'API Custom Search JSON no està habilitada al projecte de Google Cloud o la clau no té permís. Ometent extractor de Google Search.")
			break
		}

		if resp.StatusCode != http.StatusOK {
			var errRes googleSearchResult
			_ = json.NewDecoder(resp.Body).Decode(&errRes)
			resp.Body.Close()
			if errRes.Error != nil {
				log.Printf("Avis Google Search API error (%d): %s", errRes.Error.Code, errRes.Error.Message)
			}
			continue
		}

		var searchRes googleSearchResult
		err = json.NewDecoder(resp.Body).Decode(&searchRes)
		resp.Body.Close()
		if err != nil {
			continue
		}

		for _, item := range searchRes.Items {
			if item.Link == "" || item.Title == "" {
				continue
			}

			rawText := item.Snippet
			if rawText == "" {
				rawText = item.Title
			}

			signals = append(signals, RawSignal{
				SourceURL:  item.Link,
				Title:      item.Title,
				RawText:    rawText,
				SignalType: "queixa_forum",
				Source:     "google_search",
			})
		}
	}

	return signals, nil
}
