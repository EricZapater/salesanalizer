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

// NewRSSScraper crea un extractor RSS focalitzat en fòrums de Catalunya i gestió de PIMEs
func NewRSSScraper(name string, feedURLs []RSSFeedConfig) *RSSScraper {
	if len(feedURLs) == 0 {
		feedURLs = []RSSFeedConfig{
			{
				Name:       "Reddit Catalunya (Negocis & Feina)",
				URL:        "https://www.reddit.com/r/catalunya/search.rss?q=empresa+OR+feina+OR+gesti%C3%B3+OR+aut%C3%B2nom+OR+factura+OR+excel+OR+treball&sort=new&restrict_sr=on",
				SignalType: "queixa_forum",
			},
			{
				Name:       "Reddit Barcelona (PIMEs & Gestió)",
				URL:        "https://www.reddit.com/r/barcelona/search.rss?q=business+OR+empresa+OR+gestoria+OR+autonomo+OR+freelance+OR+pime&sort=new&restrict_sr=on",
				SignalType: "queixa_forum",
			},
			{
				Name:       "Reddit Autònoms & PIMEs",
				URL:        "https://www.reddit.com/r/autonomos/.rss",
				SignalType: "queixa_forum",
			},
			{
				Name:       "Reddit Emprendedores Local",
				URL:        "https://www.reddit.com/r/emprendedor/.rss",
				SignalType: "queixa_forum",
			},
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

// isBusinessRelevant filtra publicacions que no tinguin relació amb activitat laboral, comercial o gestió
func isBusinessRelevant(title, content string) bool {
	combined := strings.ToLower(title + " " + content)
	keywords := []string{
		"empresa", "pime", "autònom", "autonomo", "gestió", "gestio", "gestoria",
		"factura", "albarà", "albaran", "torn", "horari", "quadrant", "excel",
		"client", "pressupost", "proveïdor", "proveedor", "estoc", "inventari",
		"comanda", "pedido", "obra", "taller", "magatzem", "almacen", "ruta",
		"transport", "furgoneta", "contracte", "iva", "impost", "botiga", "comerç",
		"treballador", "empleat", "nòmina", "personal", "manual", "paper", "registre",
	}

	for _, kw := range keywords {
		if strings.Contains(combined, kw) {
			return true
		}
	}
	return false
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

			// Filtrar només publicacions rellevants per a negocis i ineficiències
			if !isBusinessRelevant(item.Title, cleanContent) {
				continue
			}

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
