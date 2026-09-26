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
				Name:       "Contractació Pública Catalunya (PSCP)",
				URL:        "https://contractaciopublica.cat/ca/rss/licitacions",
				SignalType: "public_tender",
			},
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

// isBusinessRelevant filtra publicacions que no tinguin relació directa amb gestió de negocis, PIMEs o software operatiu
func isBusinessRelevant(title, content, sigType string) bool {
	if sigType == "public_tender" {
		return true
	}

	combined := strings.ToLower(title + " " + content)

	// 1. Filtre d'exclusió de notícies generals, debats culturals, política, cinema i queixes ciutadanes
	exclusions := []string{
		"pel·lícula", "película", "cinema", "actor", "actriu", "cançó", "cantant", "música",
		"futbol", "barça", "partit polític", "eleccions", "govern", "parlament", "generalitat prepara",
		"jutjat", "presó", "policia", "mossos", "llengua catalana", "nivell c de català",
		"migrants", "menes", "manifestació", "independentisme", "vox", "erc", "cup", "psoe", "pp",
		"rodalies", "estudiar", "universitat", "quin grau", "turisme", "viatge",
		"detectiu conan", "medalla d’or", "medalla d'or", "joc de paraules",
	}

	for _, ex := range exclusions {
		if strings.Contains(combined, ex) {
			return false
		}
	}

	// 2. Patrons forts de necessitat de negoci, gestió de PIMEs o dolor de processos manuals
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(quadrant|quadrants)\b.*\b(torns?|empleats?|treballadors?|horari|excel)\b`),
		regexp.MustCompile(`(?i)\b(torns?\s+rotatius?|gestió\s+de\s+torns|control\s+horari|fitxatge)\b`),
		regexp.MustCompile(`(?i)\b(fulls?\s+d['e]\s*excel|fulls?\s+de\s+càlcul)\b.*\b(gestió|facturació|estoc|clients?|comandes?|automati)\b`),
		regexp.MustCompile(`(?i)\b(gestoria|autònom\s+societari|pimes?|pymes?|autònom|autonomo)\b.*\b(facturació|impost|gestió|software|programa|eina|rebut)\b`),
		regexp.MustCompile(`(?i)\b(albarans?|full\s+de\s+ruta|repartiment|magatzem)\b.*\b(clients?|paper|transport|gestió|signatura)\b`),
		regexp.MustCompile(`(?i)\b(control\s+d['e]\s*estoc|inventari\s+de\s+material|gestió\s+d['e]\s*estocs?)\b`),
		regexp.MustCompile(`(?i)\b(busco\s+programa|software\s+senzill|alguna\s+app|eina\s+per|quin\s+programa)\b.*\b(gestionar|factures|clients|torns|horaris|estocs)\b`),
		regexp.MustCompile(`(?i)\b(estic\s+fart\s+de|perdem\s+molt\s+de\s+temps|procés\s+manual)\b.*\b(excel|paper|whatsapp|factures|manualment)\b`),
		regexp.MustCompile(`(?i)\b(manteniment\s+preventiu|revisió\s+maquinària|part\s+de\s+treball)\b`),
	}

	for _, pat := range patterns {
		if pat.MatchString(combined) {
			return true
		}
	}

	return false
}

func (s *RSSScraper) Fetch(ctx context.Context) ([]RawSignal, error) {
	var signals []RawSignal
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)

	log.Printf("[RSS Scraper] Iniciant extracció de %d feeds RSS configurats...", len(s.feeds))

	for i, feedConfig := range s.feeds {
		if ctx.Err() != nil {
			return signals, ctx.Err()
		}

		// Retard de 2.5s entre feeds per respectar el rate limit de Reddit (evitar 429)
		if i > 0 {
			select {
			case <-time.After(2500 * time.Millisecond):
			case <-ctx.Done():
				return signals, ctx.Err()
			}
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
			if !isBusinessRelevant(item.Title, cleanContent, sigType) {
				continue
			}

			var authorCompany *string
			if item.Author != nil && item.Author.Name != "" {
				authorCompany = &item.Author.Name
			} else if sigType == "public_tender" {
				defaultOrg := "Organisme Públic (PSCP)"
				authorCompany = &defaultOrg
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

		log.Printf("[RSS Scraper] ✅ Feed %s: %d items analitzats, %d rellevants", feedConfig.Name, len(feed.Items), feedMatched)
	}

	log.Printf("[RSS Scraper] Resum final: %d senyals recollits en total", len(signals))

	if len(signals) == 0 && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return signals, nil
}


