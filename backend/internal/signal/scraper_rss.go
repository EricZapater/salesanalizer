package signal

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSScraper struct {
	name       string
	feedURL    string
	signalType string
	parser     *gofeed.Parser
}

func NewRSSScraper(name, feedURL, signalType string) *RSSScraper {
	if signalType == "" {
		signalType = "queixa_forum"
	}

	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Timeout: 15 * time.Second,
	}

	return &RSSScraper{
		name:       name,
		feedURL:    feedURL,
		signalType: signalType,
		parser:     fp,
	}
}

func (s *RSSScraper) Name() string {
	return s.name
}

func (s *RSSScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	feed, err := s.parser.ParseURLWithContext(s.feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("error analitzant feed RSS (%s): %w", s.feedURL, err)
	}

	var signals []RawSignal
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)

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
			SignalType:  s.signalType,
			Source:      s.name,
		})
	}

	return signals, nil
}
