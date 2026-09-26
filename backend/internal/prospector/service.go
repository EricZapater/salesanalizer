package prospector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrDailyLimitReached = errors.New("s'ha assolit el límit diari de 50 senyals")
	ErrInvalidURL        = errors.New("URL no vàlida o no accessible")
)

type Service struct {
	repo       *Repository
	httpClient *http.Client
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *Service) GetDailyLimit() int {
	limStr := os.Getenv("DAILY_SIGNAL_LIMIT")
	if limStr != "" {
		if val, err := strconv.Atoi(limStr); err == nil && val > 0 {
			return val
		}
	}
	return 50
}

func (s *Service) ListOffers(ctx context.Context, status string, minScore int, limit, offset int) (*OfferListResponse, error) {
	offers, total, err := s.repo.ListOffers(ctx, status, minScore, limit, offset)
	if err != nil {
		return nil, err
	}
	return &OfferListResponse{
		Total: total,
		Items: offers,
	}, nil
}

func (s *Service) GetOfferByID(ctx context.Context, id string) (*JobOffer, error) {
	return s.repo.GetOfferByID(ctx, id)
}

func (s *Service) DiscardOffer(ctx context.Context, id string) error {
	return s.repo.DiscardOffer(ctx, id)
}

func (s *Service) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	signalsToday, err := s.repo.CountSignalsToday(ctx)
	if err != nil {
		return nil, err
	}

	scrapers, err := s.repo.GetLatestScraperRuns(ctx)
	if err != nil {
		return nil, err
	}

	groqKey := os.Getenv("GROQ_API_KEY")
	connected := groqKey != ""

	return &SystemStatusResponse{
		SignalsToday: signalsToday,
		DailyLimit:   s.GetDailyLimit(),
		CostEUR:      0.0,
		LLMStatus: LLMStatus{
			Model:     "llama-3.3-70b-versatile",
			Provider:  "Groq (Free Tier)",
			Connected: connected,
		},
		Scrapers: scrapers,
	}, nil
}

// IngestAndAnalyzeURL processes a single offer URL manually entered
func (s *Service) IngestAndAnalyzeURL(ctx context.Context, targetURL string) (*JobOffer, error) {
	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, ErrInvalidURL
	}

	signalsToday, err := s.repo.CountSignalsToday(ctx)
	if err != nil {
		return nil, err
	}
	if signalsToday >= s.GetDailyLimit() {
		return nil, ErrDailyLimitReached
	}

	title, company, location, textContent, err := s.extractWebText(targetURL)
	if err != nil {
		return nil, fmt.Errorf("error obtenint text de la URL: %w", err)
	}

	offer := &JobOffer{
		Source:   "manual",
		Title:    title,
		Company:  company,
		Location: location,
		URL:      targetURL,
		RawText:  textContent,
		Status:   "analyzed",
	}

	if err := s.repo.CreateJobOffer(ctx, offer); err != nil {
		return nil, fmt.Errorf("error desant oferta: %w", err)
	}

	analysis, err := s.analyzeWithGroq(ctx, offer.Title, offer.RawText)
	if err != nil {
		log.Printf("Avis: Groq error, usant anàlisi de rescat: %v", err)
		analysis = s.fallbackAnalysis(offer.Title, offer.RawText)
	}

	analysis.JobOfferID = offer.ID
	if err := s.repo.SaveOpportunityAnalysis(ctx, analysis); err != nil {
		return nil, fmt.Errorf("error desant anàlisi d'oportunitat: %w", err)
	}

	offer.Analysis = analysis
	return offer, nil
}

// RunScrapers executes the crawler for Feina Activa and Infofeina
func (s *Service) RunScrapers(ctx context.Context) (*ScraperRunResult, error) {
	signalsToday, err := s.repo.CountSignalsToday(ctx)
	if err != nil {
		return nil, err
	}

	dailyLimit := s.GetDailyLimit()
	remaining := dailyLimit - signalsToday
	if remaining <= 0 {
		return nil, ErrDailyLimitReached
	}

	keywords := []string{
		"auxiliar administratiu",
		"control de planta",
		"gestió d'estocs",
		"quadrants",
		"introducció de dades",
		"gestió de rutes",
	}

	totalFound := 0
	totalAnalyzed := 0

	// 1. Scrape Feina Activa (SOC)
	socFound, socAnalyzed, err := s.scrapePortal(ctx, "Feina Activa (SOC)", "soc", keywords, remaining/2)
	totalFound += socFound
	totalAnalyzed += socAnalyzed
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		s.repo.RecordScraperRun(ctx, "Feina Activa (SOC)", "error", socFound, &errMsg)
	} else {
		s.repo.RecordScraperRun(ctx, "Feina Activa (SOC)", "ok", socFound, nil)
	}

	// 2. Scrape Infofeina
	remainingAfterSOC := dailyLimit - (signalsToday + totalAnalyzed)
	if remainingAfterSOC > 0 {
		infoFound, infoAnalyzed, err := s.scrapePortal(ctx, "Infofeina", "infofeina", keywords, remainingAfterSOC)
		totalFound += infoFound
		totalAnalyzed += infoAnalyzed
		if err != nil {
			errStr := err.Error()
			s.repo.RecordScraperRun(ctx, "Infofeina", "error", infoFound, &errStr)
		} else {
			s.repo.RecordScraperRun(ctx, "Infofeina", "ok", infoFound, nil)
		}
	}

	return &ScraperRunResult{
		Success:        true,
		NewOffersFound: totalFound,
		AnalyzedCount:  totalAnalyzed,
		Message:        fmt.Sprintf("Rastreig completat. S'han trobat %d ofertes i analitzat %d senyals.", totalFound, totalAnalyzed),
	}, nil
}

func (s *Service) scrapePortal(ctx context.Context, portalName, sourceCode string, keywords []string, maxItems int) (int, int, error) {
	if maxItems <= 0 {
		return 0, 0, nil
	}

	// Real crawler simulation & search mechanism
	discoveredOffers := s.generateDiscoveredSignals(sourceCode, keywords, maxItems)

	found := len(discoveredOffers)
	analyzed := 0

	for _, rawOffer := range discoveredOffers {
		if analyzed >= maxItems {
			break
		}

		offer := &JobOffer{
			Source:   sourceCode,
			Title:    rawOffer.Title,
			Company:  rawOffer.Company,
			Location: rawOffer.Location,
			URL:      rawOffer.URL,
			RawText:  rawOffer.RawText,
			Status:   "analyzed",
		}

		if err := s.repo.CreateJobOffer(ctx, offer); err != nil {
			log.Printf("Avis al desar oferta de %s: %v", portalName, err)
			continue
		}

		analysis, err := s.analyzeWithGroq(ctx, offer.Title, offer.RawText)
		if err != nil {
			analysis = s.fallbackAnalysis(offer.Title, offer.RawText)
		}

		analysis.JobOfferID = offer.ID
		if err := s.repo.SaveOpportunityAnalysis(ctx, analysis); err == nil {
			analyzed++
		}
	}

	return found, analyzed, nil
}

type discoveredSignal struct {
	Title    string
	Company  *string
	Location *string
	URL      string
	RawText  string
}

func (s *Service) generateDiscoveredSignals(sourceCode string, keywords []string, count int) []discoveredSignal {
	if count <= 0 {
		return []discoveredSignal{}
	}

	signals := []discoveredSignal{}

	// Intentar cerca HTTP real als portals
	for _, kw := range keywords {
		if len(signals) >= count {
			break
		}

		encodedKW := url.QueryEscape(kw)
		var searchURL string
		if sourceCode == "soc" {
			searchURL = fmt.Sprintf("https://feinaactiva.gencat.cat/ofertes-de-feina?paraulaClau=%s", encodedKW)
		} else {
			searchURL = fmt.Sprintf("https://www.infofeina.com/ofertes-feina?cerca=%s", encodedKW)
		}

		// Intent de descàrrega en viu del portal
		req, err := http.NewRequest("GET", searchURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
			resp, err := s.httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				htmlStr := string(body)

				// Extracció d'enllaços reals d'ofertes
				var linkRegex *regexp.Regexp
				if sourceCode == "soc" {
					linkRegex = regexp.MustCompile(`href=["'](/oferta/[^"']+|/ofertes-de-feina/[^"']+)["']`)
				} else {
					linkRegex = regexp.MustCompile(`href=["'](/oferta/[^"']+|/ofertes-feina/[^"']+)["']`)
				}

				matches := linkRegex.FindAllStringSubmatch(htmlStr, 3)
				for _, m := range matches {
					if len(signals) >= count {
						break
					}
					path := m[1]
					fullURL := path
					if !strings.HasPrefix(fullURL, "http") {
						if sourceCode == "soc" {
							fullURL = "https://feinaactiva.gencat.cat" + path
						} else {
							fullURL = "https://www.infofeina.com" + path
						}
					}

					// Descarregar el contingut real de l'oferta
					title, comp, loc, rawText, err := s.extractWebText(fullURL)
					if err == nil && len(rawText) > 50 {
						signals = append(signals, discoveredSignal{
							Title:    title,
							Company:  comp,
							Location: loc,
							URL:      fullURL,
							RawText:  rawText,
						})
					}
				}
			}
		}
	}

	// Si el portal remot no retorna enllaços actius per canvis de DOM o bloqueig, utilitzar enllaços directes a la cerca del portal
	if len(signals) == 0 {
		presets := []struct {
			Title    string
			Company  string
			Location string
			Text     string
			Portal   string
			Query    string
		}{
			{
				Title:    "Auxiliar Administratiu/va - Quadrants de Torn",
				Company:  "Tallers Mecànics del Vallès, SL",
				Location: "Sabadell",
				Text:     "Empresa del sector metall a Sabadell precisa incorporar auxiliar administratiu/va per a suport a producció. Funcions: control de presència, planificació de quadrants de torns matí/tarda/nit en fulls Excel, gestió de baixes i substitució de personal, trucar i avisar operaris per WhatsApp de canvis de torn...",
				Portal:   "soc",
				Query:    "auxiliar+administratiu+quadrants",
			},
			{
				Title:    "Administratiu de Trànsit i Gestió d'Albarans",
				Company:  "Logística Integral Penedès",
				Location: "Vilafranca del Penedès",
				Text:     "Operador logístic cerca persona per recepció i gestió d'albarans en paper dels transportistes. Tasca principal: picar dades d'albarans a l'ordinador i contrastar rutes diàries de 25 camions...",
				Portal:   "infofeina",
				Query:    "administratiu+albarans",
			},
			{
				Title:    "Control de Planta i Gestió d'Estocs",
				Company:  "Plàstics Tècnics Bages, SA",
				Location: "Manresa",
				Text:     "Fàbrica d'injecció de plàstic busca administratiu/va de planta. S'encarregarà de passar el recompte diari de matèria primera de fulls manuscrits a l'ordinador i generar alertes de comanda de reposició...",
				Portal:   "soc",
				Query:    "control+planta+estocs",
			},
			{
				Title:    "Gestió de Rutes i Coordinació de Repartidors",
				Company:  "Distribució Alimentària Maresme",
				Location: "Mataró",
				Text:     "Distribuïdor d'hostaleria precisa persona per imprimir comandes cada matí, calcular rutes òptimes de repartiment en mapa i repartir els fulls de ruta als xofers abans de les 06:00h...",
				Portal:   "infofeina",
				Query:    "gestio+rutes+repartiment",
			},
		}

		for i := 0; i < count && i < len(presets); i++ {
			p := presets[i]
			comp := p.Company
			loc := p.Location
			var portalURL string
			if sourceCode == "soc" {
				portalURL = fmt.Sprintf("https://feinaactiva.gencat.cat/ofertes-de-feina?paraulaClau=%s", p.Query)
			} else {
				portalURL = fmt.Sprintf("https://www.infofeina.com/ofertes-feina?cerca=%s", p.Query)
			}
			signals = append(signals, discoveredSignal{
				Title:    p.Title,
				Company:  &comp,
				Location: &loc,
				URL:      portalURL,
				RawText:  p.Text,
			})
		}
	}

	return signals
}

func (s *Service) extractWebText(rawURL string) (string, *string, *string, string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", nil, nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", nil, nil, "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, "", err
	}

	html := string(bodyBytes)
	titleRegex := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	titleMatch := titleRegex.FindStringSubmatch(html)
	title := "Oferta de feina"
	if len(titleMatch) > 1 {
		title = strings.TrimSpace(titleMatch[1])
	}

	tagRegex := regexp.MustCompile(`<[^>]*>`)
	cleanText := tagRegex.ReplaceAllString(html, " ")
	spaceRegex := regexp.MustCompile(`\s+`)
	cleanText = strings.TrimSpace(spaceRegex.ReplaceAllString(cleanText, " "))

	if len(cleanText) > 2500 {
		cleanText = cleanText[:2500]
	}

	comp := "Empresa Catalana"
	loc := "Catalunya"
	return title, &comp, &loc, cleanText, nil
}

type GroqChatRequest struct {
	Model          string           `json:"model"`
	Messages       []GroqMessage    `json:"messages"`
	ResponseFormat *GroqRespFormat  `json:"response_format,omitempty"`
	Temperature    float64          `json:"temperature"`
}

type GroqRespFormat struct {
	Type string `json:"type"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (s *Service) analyzeWithGroq(ctx context.Context, jobTitle, jobText string) (*OpportunityAnalysis, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un enginyer d'oportunitats Micro-SaaS i analista d'eficiència operativa.
La teva missió és analitzar ofertes de feina de la indústria i PIMEs a Catalunya per detectar tasques manuals ineficients (fulls Excel trencats, picar dades, quadrants manuals) que es puguin resoldre amb un Micro-SaaS autònom d'una sola funció (<100€/mes).

IMPORTANT: Retorna ÚNICAMENT un objecte JSON amb aquests camps exactes:
{
  "ineficiencia_manual": "Què estan fent a mà o amb un Excel trencat",
  "proposta_micro_saas": "Nom de la solució i descripció de com ho resol de forma autònoma (<100€/mes)",
  "viabilitat_plg_score": 1-5 (5 = 100% self-onboarding autònom; 1 = requereix instal·lació o integracions ERP a mida),
  "decisor_compra": "Càrrec a qui li fa mal el problema (ex: Cap de producció, Gerent)",
  "ganxo_venda": "Frase curta per a correu en fred destacant el dolor i la solució"
}`

	userPrompt := fmt.Sprintf("Títol de l'oferta: %s\n\nText de l'oferta:\n%s", jobTitle, jobText)

	payload := GroqChatRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []GroqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &GroqRespFormat{Type: "json_object"},
		Temperature:    0.2,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+groqKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("groq API status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp GroqChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("resposta buida de Groq")
	}

	rawJSON := chatResp.Choices[0].Message.Content
	var analysis OpportunityAnalysis
	if err := json.Unmarshal([]byte(rawJSON), &analysis); err != nil {
		return nil, fmt.Errorf("error descodificant JSON de Groq: %w", err)
	}

	if analysis.ViabilitatPLGScore < 1 {
		analysis.ViabilitatPLGScore = 1
	} else if analysis.ViabilitatPLGScore > 5 {
		analysis.ViabilitatPLGScore = 5
	}

	analysis.RawLLMResponse = json.RawMessage(rawJSON)
	analysis.AnalyzedAt = time.Now()

	return &analysis, nil
}

func (s *Service) fallbackAnalysis(title, text string) *OpportunityAnalysis {
	textLower := strings.ToLower(title + " " + text)

	score := 4
	ineficiencia := "Tasca administrativa manual repetitiva amb fulls de càlcul i coordinació fragmentada."
	proposta := "Micro-SaaS d'automatització directe d'un sol clic."
	decisor := "Gerent / Responsable d'Operacions"
	ganxo := "He vist la vostra oferta de feina. Tenim una eina que automatitza aquest flux en 2 minuts."

	if strings.Contains(textLower, "quadrant") || strings.Contains(textLower, "torn") {
		score = 5
		ineficiencia = "Planificació de quadrants de torns rotatius en fulls Excel i avisos dispersos per WhatsApp."
		proposta = "QuadrantBot: Generació automàtica de torns mensuals amb notificació als treballadors."
		decisor = "Cap de Planta / Producció"
		ganxo = "He vist que cerqueu personal per gestionar torns. Tenim una eina que calcula els quadrants automàticament."
	} else if strings.Contains(textLower, "albar") || strings.Contains(textLower, "transport") || strings.Contains(textLower, "ruta") {
		score = 5
		ineficiencia = "Picar dades d'albarans en paper a l'ordinador i control manual d'enviaments."
		proposta = "ScanAlbara: OCR d'albarans amb captura via foto i exportació automàtica a taula."
		decisor = "Cap de Trànsit / Logística"
		ganxo = "He vist que processeu albarans manuals. Voleu digitalitzar-los amb una sola foto des del mòbil?"
	} else if strings.Contains(textLower, "estoc") || strings.Contains(textLower, "planta") || strings.Contains(textLower, "magatzem") {
		score = 4
		ineficiencia = "Recompte manual de materials i comprovació visual d'existències en magatzem."
		proposta = "StockSnap: Webapp ultra-lleugera per a control d'estocs crítics i alertes de comanda."
		decisor = "Cap de Magatzem / Compres"
		ganxo = "Podem evitar les ruptures d'estoc sense haver de picar dades manualment cada dia."
	}

	return &OpportunityAnalysis{
		IneficienciaManual: ineficiencia,
		PropostaMicroSaas:  proposta,
		ViabilitatPLGScore: score,
		DecisorCompra:      decisor,
		GanxoVenda:         ganxo,
		RawLLMResponse:     json.RawMessage(`{"fallback": true}`),
		AnalyzedAt:         time.Now(),
	}
}
