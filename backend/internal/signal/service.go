package signal

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
	scrapers   []Scraper
	httpClient *http.Client
}

func NewService(repo *Repository) *Service {
	svc := &Service{
		repo:     repo,
		scrapers: make([]Scraper, 0),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	// Registre per defecte dels extractors inicials
	svc.RegisterScraper(NewFeinaActivaScraper(nil))
	svc.RegisterScraper(NewRSSScraper("Reddit SmallBusiness", "https://www.reddit.com/r/smallbusiness/.rss", "queixa_forum"))
	svc.RegisterScraper(NewRSSScraper("Reddit Entrepreneur", "https://www.reddit.com/r/Entrepreneur/.rss", "queixa_forum"))

	return svc
}

// RegisterScraper permet afegir nous extractors (patró Strategy) sense tocar el nucli
func (s *Service) RegisterScraper(scraper Scraper) {
	s.scrapers = append(s.scrapers, scraper)
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

func (s *Service) ListSignals(ctx context.Context, status string, minScore int, limit, offset int) (*SignalListResponse, error) {
	signals, total, err := s.repo.ListSignals(ctx, status, minScore, limit, offset)
	if err != nil {
		return nil, err
	}
	return &SignalListResponse{
		Total: total,
		Items: signals,
	}, nil
}

func (s *Service) GetSignalByID(ctx context.Context, id string) (*Signal, error) {
	return s.repo.GetSignalByID(ctx, id)
}

func (s *Service) DiscardSignal(ctx context.Context, id string) error {
	return s.repo.DiscardSignal(ctx, id)
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

// ProcessScrapers itera de forma seqüencial sobre els extractors registrats
func (s *Service) ProcessScrapers(ctx context.Context) (*ScraperRunResult, error) {
	signalsToday, err := s.repo.CountSignalsToday(ctx)
	if err != nil {
		return nil, err
	}

	dailyLimit := s.GetDailyLimit()
	if signalsToday >= dailyLimit {
		return nil, ErrDailyLimitReached
	}

	totalFound := 0
	totalAnalyzed := 0

	for _, scraper := range s.scrapers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Comprovar si ja hem arribat al límit
		if signalsToday+totalAnalyzed >= dailyLimit {
			break
		}

		log.Printf("Iniciant extracció amb scraper: %s", scraper.Name())
		rawSignals, err := scraper.Fetch(ctx)
		if err != nil {
			errMsg := err.Error()
			log.Printf("Error a l'extractor %s: %v", scraper.Name(), err)
			_ = s.repo.RecordScraperRun(ctx, scraper.Name(), "error", 0, &errMsg)
			continue
		}

		scraperFound := len(rawSignals)
		scraperAnalyzed := 0
		totalFound += scraperFound

		for _, rawSig := range rawSignals {
			if signalsToday+totalAnalyzed >= dailyLimit {
				break
			}

			// 1. Descartar si ja existeix a la base de dades
			exists, err := s.repo.ExistsByURL(ctx, rawSig.SourceURL)
			if err != nil || exists {
				continue
			}

			// 2. Desar el senyal a la BD
			signalID, err := s.repo.SaveRawSignal(ctx, &rawSig)
			if err != nil {
				log.Printf("Error desant senyal de %s: %v", scraper.Name(), err)
				continue
			}

			// 3. Processament amb Groq
			analysis, err := s.analyzeWithGroq(ctx, rawSig.Title, rawSig.RawText, rawSig.SignalType)
			if err != nil {
				log.Printf("Avis: Error a Groq, usant anàlisi de rescat: %v", err)
				analysis = s.fallbackAnalysis(rawSig.Title, rawSig.RawText)
			}

			analysis.SignalID = signalID
			if err := s.repo.SaveOpportunityAnalysis(ctx, analysis); err == nil {
				scraperAnalyzed++
				totalAnalyzed++
			}
		}

		_ = s.repo.RecordScraperRun(ctx, scraper.Name(), "ok", scraperFound, nil)
	}

	return &ScraperRunResult{
		Success:        true,
		NewOffersFound: totalFound,
		AnalyzedCount:  totalAnalyzed,
		Message:        fmt.Sprintf("Extracció completada. S'han trobat %d senyals i analitzat %d noves oportunitats.", totalFound, totalAnalyzed),
	}, nil
}

// IngestAndAnalyzeURL processa manualment una URL
func (s *Service) IngestAndAnalyzeURL(ctx context.Context, targetURL string) (*Signal, error) {
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

	title, company, rawText, err := s.extractWebText(targetURL)
	if err != nil {
		return nil, fmt.Errorf("error obtenint text de la URL: %w", err)
	}

	rawSig := RawSignal{
		SourceURL:   targetURL,
		CompanyName: company,
		Title:       title,
		RawText:     rawText,
		SignalType:  "oferta_feina",
		Source:      "manual",
	}

	signalID, err := s.repo.SaveRawSignal(ctx, &rawSig)
	if err != nil {
		return nil, fmt.Errorf("error desant senyal: %w", err)
	}

	analysis, err := s.analyzeWithGroq(ctx, rawSig.Title, rawSig.RawText, rawSig.SignalType)
	if err != nil {
		analysis = s.fallbackAnalysis(rawSig.Title, rawSig.RawText)
	}

	analysis.SignalID = signalID
	if err := s.repo.SaveOpportunityAnalysis(ctx, analysis); err != nil {
		return nil, fmt.Errorf("error desant anàlisi: %w", err)
	}

	return s.repo.GetSignalByID(ctx, signalID)
}

func (s *Service) extractWebText(rawURL string) (string, *string, string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", nil, "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, "", err
	}

	html := string(bodyBytes)
	titleRegex := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	titleMatch := titleRegex.FindStringSubmatch(html)
	title := "Senyal d'oportunitat"
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

	comp := "Empresa"
	return title, &comp, cleanText, nil
}

type GroqChatRequest struct {
	Model          string          `json:"model"`
	Messages       []GroqMessage   `json:"messages"`
	ResponseFormat *GroqRespFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
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

func (s *Service) analyzeWithGroq(ctx context.Context, title, text, signalType string) (*OpportunityAnalysis, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un analista d'oportunitats Micro-SaaS.
La teva missió és analitzar ofertes de feina i queixes en fòrums per detectar tasques manuals ineficients (Excel trencats, introducció de dades, gestió de torns, comunicació dispersa) que es puguin resoldre amb un Micro-SaaS d'una sola funció (<100€/mes).

Retorna ÚNICAMENT un objecte JSON amb aquests camps exactes:
{
  "ineficiencia_manual": "Què estan fent a mà o amb un procés trencat",
  "proposta_micro_saas": "Nom i descripció de la solució autònoma d'una sola funció (<100€/mes)",
  "viabilitat_plg_score": 1-5 (5 = 100% self-onboarding autònom; 1 = requereix integració complexa a mida),
  "decisor_compra": "Càrrec a qui li fa mal el problema (ex: Cap de planta, Gerent, Propietari)",
  "ganxo_venda": "Frase curta per a correu en fred destacant el dolor i la solució"
}`

	userPrompt := fmt.Sprintf("Tipus de senyal: %s\nTítol: %s\n\nText:\n%s", signalType, title, text)

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
		return nil, fmt.Errorf("groq status %d: %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("error parsejant JSON de Groq: %w", err)
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
	ineficiencia := "Tasca manual repetitiva gestionada amb fulls de càlcul o coordinació manual."
	proposta := "Micro-SaaS d'automatització directe d'un sol flux."
	decisor := "Gerent / Responsable d'Operacions"
	ganxo := "He vist la vostra publicació. Tenim una eina que automatitza aquest flux en 2 minuts."

	if strings.Contains(textLower, "quadrant") || strings.Contains(textLower, "torn") {
		score = 5
		ineficiencia = "Planificació de quadrants de torns rotatius en fulls Excel i avisos dispersos per WhatsApp."
		proposta = "QuadrantBot: Generació automàtica de torns mensuals amb notificació als treballadors."
		decisor = "Cap de Planta / Producció"
		ganxo = "Tenim una eina que calcula els quadrants automàticament sense passar per Excel."
	} else if strings.Contains(textLower, "albar") || strings.Contains(textLower, "transport") || strings.Contains(textLower, "ruta") {
		score = 5
		ineficiencia = "Picar dades d'albarans en paper a l'ordinador i control manual d'enviaments."
		proposta = "ScanAlbara: OCR d'albarans amb captura via foto i exportació automàtica a taula."
		decisor = "Cap de Trànsit / Logística"
		ganxo = "Voleu digitalitzar els albarans amb una sola foto des del mòbil?"
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
