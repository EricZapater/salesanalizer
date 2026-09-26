package signal

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSFeedConfig struct {
	Name       string
	URL        string
	SignalType string
}

type RSSScraper struct {
	name   string
	feeds  []RSSFeedConfig
	parser *gofeed.Parser
}

// NewRSSScraper crea un extractor RSS compatible amb múltiples feeds de fòrums
func NewRSSScraper(name string, feedURLs []RSSFeedConfig) *RSSScraper {
	if len(feedURLs) == 0 {
		feedURLs = []RSSFeedConfig{
			{Name: "Reddit SmallBusiness", URL: "https://www.reddit.com/r/smallbusiness/.rss", SignalType: "queixa_forum"},
			{Name: "Reddit Entrepreneur", URL: "https://www.reddit.com/r/Entrepreneur/.rss", SignalType: "queixa_forum"},
			{Name: "Reddit Manufacturing", URL: "https://www.reddit.com/r/manufacturing/.rss", SignalType: "queixa_forum"},
			{Name: "Reddit Logistics", URL: "https://www.reddit.com/r/logistics/.rss", SignalType: "queixa_forum"},
		}
	}

	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Timeout: 20 * time.Second,
	}

	return &RSSScraper{
		name:   name,
		feeds:  feedURLs,
		parser: fp,
	}
}

func (s *RSSScraper) Name() string {
	return s.name
}

func (s *RSSScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)

	for _, feedConfig := range s.feeds {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		feed, err := s.parser.ParseURLWithContext(feedConfig.URL, ctx)
		if err != nil {
			// Si un feed específic falla o fa rate-limiting, continuem amb els altres
			continue
		}

		sigType := feedConfig.SignalType
		if sigType == "" {
			sigType = "queixa_forum"
		}

		for _, item := range feed.Items {
			if ctx.Err() != nil {
				return signals, ctx.Err()
			}

			cleanContent := item.Content
			if cleanContent == "" {
				cleanContent = item.Description
			}

			cleanContent = htmlTagRegex.ReplaceAllString(cleanContent, " ")
			cleanContent = strings.TrimSpace(cleanContent)

			var authorCompany *string
			if item.Author != nil && item.Author.Name != "" {
				authorCompany = &item.Author.Name
			}

			signals = append(signals, RawSignal{
				SourceURL:   item.Link,
				CompanyName: authorCompany,
				Title:       item.Title,
				RawText:     cleanContent,
				SignalType:  sigType,
				Source:      feedConfig.Name,
			})
		}
	}

	if len(signals) == 0 && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return signals, nil
}
