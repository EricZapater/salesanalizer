package signal

import (
	"context"
	"log"
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

type customUserAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *customUserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("User-Agent", t.userAgent)
	reqCopy.Header.Set("Accept", "application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.8")
	return t.base.RoundTrip(reqCopy)
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
				URL:        "https://www.reddit.com/r/catalunya/search.rss?q=empresa+OR+feina+OR+gesti%C3%B3+OR+aut%C3%B2nom+OR+factura+OR+excel+OR+treball+OR+negoci&sort=new&restrict_sr=on",
				SignalType: "queixa_forum",
			},
			{
				Name:       "Reddit Barcelona (PIMEs & Gestió)",
				URL:        "https://www.reddit.com/r/barcelona/search.rss?q=business+OR+empresa+OR+gestoria+OR+autonomo+OR+freelance+OR+pime+OR+software+OR+work&sort=new&restrict_sr=on",
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
			{
				Name:       "Reddit Spain (Negocis & PIMEs)",
				URL:        "https://www.reddit.com/r/spain/search.rss?q=autonomo+OR+pyme+OR+empresa+OR+factura+OR+gestoria+OR+excel&sort=new&restrict_sr=on",
				SignalType: "queixa_forum",
			},
		}
	}

	httpClient := &http.Client{
		Timeout: 25 * time.Second,
		Transport: &customUserAgentTransport{
			base:      http.DefaultTransport,
			userAgent: "web:salesanalizer.prospector.bot:v1.0.0 (by /u/ericzapater)",
		},
	}

	fp := gofeed.NewParser()
	fp.Client = httpClient

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
		"empresa", "pime", "pyme", "autònom", "autonomo", "gestió", "gestio", "gestoria",
		"factura", "facturació", "albarà", "albaran", "torn", "horari", "quadrant", "excel",
		"client", "pressupost", "presupuesto", "proveïdor", "proveedor", "estoc", "stock", "inventari",
		"comanda", "pedido", "obra", "taller", "magatzem", "almacen", "ruta", "repartiment",
		"transport", "furgoneta", "contracte", "contrato", "iva", "impost", "botiga", "comerç", "comercio",
		"treballador", "empleat", "empleado", "nòmina", "nomina", "personal", "manual", "paper", "registre",
		"crm", "erp", "software", "eina", "herramienta", "automatització", "automatizar",
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

	log.Printf("[RSS Scraper] Iniciant extracció de %d feeds RSS configurats...", len(s.feeds))

	for _, feedConfig := range s.feeds {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		log.Printf("[RSS Scraper] Connectant a feed: %s (%s)", feedConfig.Name, feedConfig.URL)
		feed, err := s.parser.ParseURLWithContext(feedConfig.URL, ctx)
		if err != nil {
			log.Printf("[RSS Scraper] ⚠️ Avis al feed %s: %v", feedConfig.Name, err)
			continue
		}

		sigType := feedConfig.SignalType
		if sigType == "" {
			sigType = "queixa_forum"
		}

		feedMatched := 0
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

			feedMatched++
			signals = append(signals, RawSignal{
				SourceURL:   item.Link,
				CompanyName: authorCompany,
				Title:       item.Title,
				RawText:     cleanContent,
				SignalType:  sigType,
				Source:      feedConfig.Name,
			})
		}

		log.Printf("[RSS Scraper] ✅ Feed %s: %d posts analitzats, %d rellevants per a negoci", feedConfig.Name, len(feed.Items), feedMatched)
	}

	log.Printf("[RSS Scraper] Resum final: %d senyals recollits en total", len(signals))

	if len(signals) == 0 && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return signals, nil
}

