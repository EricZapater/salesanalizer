package signal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrDailyLimitReached = errors.New("s'ha assolit el límit diari de 50 senyals")
	ErrInvalidURL        = errors.New("URL no vàlida o no accessible")

	activeGroqModel = "llama-3.3-70b-versatile"
	groqModelMutex  sync.RWMutex
)

func getCandidateGroqModels() []string {
	groqModelMutex.RLock()
	current := activeGroqModel
	groqModelMutex.RUnlock()

	custom := os.Getenv("GROQ_MODEL")
	models := []string{}
	if custom != "" {
		models = append(models, custom)
	}
	if current != "" && current != custom {
		models = append(models, current)
	}

	defaults := []string{
		"llama-3.3-70b-versatile",
		"llama-3.1-70b-versatile",
		"llama3-70b-8192",
		"llama-3.1-8b-instant",
		"llama3-8b-8192",
		"mixtral-8x7b-32768",
	}
	for _, d := range defaults {
		found := false
		for _, m := range models {
			if m == d {
				found = true
				break
			}
		}
		if !found {
			models = append(models, d)
		}
	}
	return models
}

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
			Timeout: 20 * time.Second,
		},
	}

	// Registre d'extractors per defecte aplicant el patró Strategy
	svc.RegisterScraper(NewFeinaActivaScraper(nil))
	svc.RegisterScraper(NewRSSScraper("Fòrums Gestió & PIMEs (RSS)", nil))
	svc.RegisterScraper(NewSearXNGScraper())

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

func (s *Service) UpdateSignalStatus(ctx context.Context, id, status string) error {
	return s.repo.UpdateSignalStatus(ctx, id, status)
}

func (s *Service) GetScraperSettings(ctx context.Context) (bool, error) {
	if s.repo == nil {
		return false, nil
	}
	return s.repo.GetScraperSettings(ctx)
}

func (s *Service) UpdateScraperSettings(ctx context.Context, deepFetch bool) error {
	for _, sc := range s.scrapers {
		if searx, ok := sc.(*SearXNGScraper); ok {
			searx.SetDeepFetch(deepFetch)
		}
	}
	if s.repo == nil {
		return nil
	}
	return s.repo.UpdateScraperSettings(ctx, deepFetch)
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

	groqModelMutex.RLock()
	currentModel := activeGroqModel
	groqModelMutex.RUnlock()

	return &SystemStatusResponse{
		SignalsToday: signalsToday,
		DailyLimit:   s.GetDailyLimit(),
		CostEUR:      0.0,
		LLMStatus: LLMStatus{
			Model:     currentModel,
			Provider:  "Groq (Free Tier)",
			Connected: connected,
		},
		Scrapers: scrapers,
	}, nil
}

type scraperResult struct {
	scraperName string
	signals     []RawSignal
	err         error
}

var entertainmentPool = []struct {
	kind string
	text string
}{
	{"joke", "🤖 Acudit IA: Un full Excel entra a un bar i demana un tallat. El cambrer li diu: 'Són 1,50€'. L'Excel respon: 'D'acord, t'ho guardo com a 01/05/1900'."},
	{"fun_fact", "💡 Sabies que el 68% de les PIMEs a Catalunya encara gestionen el quadrant de torns rotatius amb un Excel que només sap tocar una persona a l'empresa?"},
	{"joke", "🤖 Acudit PIME: — Quants consultors calen per automatitzar un procés manual? — Cap, primer farem 4 reunions per documentar el problema en un PDF de 30 pàgines."},
	{"fun_fact", "💡 La paraula 'Albarà' ve de l'àrab 'al-barā'ah' (comprovant). Té 800 anys d'història... i encara n'hi ha qui els re-pica a mà a l'ordinador cada tarda!"},
	{"fun_fact", "💡 Les millors oportunitats Micro-SaaS (Score 5) no intenten substituir un ERP sencer, sinó resoldre un sol flux concret (com sol·licitud de vacances o fitxatges d'obra) en menys de 3 clics."},
	{"joke", "🤖 Acudit Sysadmin: Hi ha dos tipus d'empreses: les que tenen alertes automàtiques i les que tenen un fitxer anomenat 'HORARIS_DEFINITIU_v2_aquest_si.xlsx'."},
}

// ProcessScrapers executa els scrapers en paral·lel
func (s *Service) ProcessScrapers(ctx context.Context) (*ScraperRunResult, error) {
	return s.ProcessScrapersWithProgress(ctx, nil)
}

// ProcessScrapersWithProgress executa tots els scrapers concurrentment i emet esdeveniments de progrés en temps real
func (s *Service) ProcessScrapersWithProgress(ctx context.Context, onProgress func(ProgressEvent)) (*ScraperRunResult, error) {
	emit := func(e ProgressEvent) {
		if onProgress != nil {
			onProgress(e)
		}
	}

	signalsToday, err := s.repo.CountSignalsToday(ctx)
	if err != nil {
		return nil, err
	}

	dailyLimit := s.GetDailyLimit()
	if signalsToday >= dailyLimit {
		return nil, ErrDailyLimitReached
	}

	// Carregar i sincronitzar paràmetre Deep Fetch
	deepFetch, _ := s.GetScraperSettings(ctx)
	for _, sc := range s.scrapers {
		if searx, ok := sc.(*SearXNGScraper); ok {
			searx.SetDeepFetch(deepFetch)
		}
	}

	emit(ProgressEvent{
		Type:          "start",
		Message:       "🚀 Llançant tots els extractors multicanal en paral·lel...",
		Progress:      10,
		EstimatedSecs: 20,
		FunFact:       entertainmentPool[rand.Intn(len(entertainmentPool))].text,
	})

	log.Printf("[Pipeline] === INICI D'EXECUCIÓ DE SCRAPERS ===")
	log.Printf("[Pipeline] Extractors registrats: %d | Deep Fetch: %t | Límit diari: %d (avui portem %d)", len(s.scrapers), deepFetch, dailyLimit, signalsToday)

	// 1. Execució PARAL·LELA de tots els scrapers registrats
	resChan := make(chan scraperResult, len(s.scrapers))
	var wg sync.WaitGroup

	for _, sc := range s.scrapers {
		wg.Add(1)
		go func(scraper Scraper) {
			defer wg.Done()
			log.Printf("[Pipeline Worker] 🚀 Llançant extractor: %s", scraper.Name())
			sigs, fetchErr := scraper.Fetch(ctx)
			resChan <- scraperResult{
				scraperName: scraper.Name(),
				signals:     sigs,
				err:         fetchErr,
			}
		}(sc)
	}


	// Tancar canal quan tots els scrapers acabin
	go func() {
		wg.Wait()
		close(resChan)
	}()

	var allRawSignals []RawSignal
	totalFound := 0
	scraperCompleted := 0

	for res := range resChan {
		scraperCompleted++
		currentProg := 10 + int((float64(scraperCompleted)/float64(len(s.scrapers)))*40.0) // 10% .. 50%

		if res.err != nil {
			errMsg := res.err.Error()
			log.Printf("[Pipeline Worker] ⚠️ Avis extractor %s: %v", res.scraperName, res.err)
			_ = s.repo.RecordScraperRun(ctx, res.scraperName, "error", 0, &errMsg)
			emit(ProgressEvent{
				Type:     "progress",
				Step:     res.scraperName,
				Message:  fmt.Sprintf("⚠️ Extractor %s ha finalitzat amb avis.", res.scraperName),
				Progress: currentProg,
			})
		} else {
			items := len(res.signals)
			totalFound += items
			allRawSignals = append(allRawSignals, res.signals...)
			log.Printf("[Pipeline Worker] ✅ Extractor %s finalitzat: %d senyals bruts recollits", res.scraperName, items)
			_ = s.repo.RecordScraperRun(ctx, res.scraperName, "ok", items, nil)
			emit(ProgressEvent{
				Type:     "progress",
				Step:     res.scraperName,
				Message:  fmt.Sprintf("✅ %s: %d senyals recollits", res.scraperName, items),
				Progress: currentProg,
				Joke:     entertainmentPool[rand.Intn(len(entertainmentPool))].text,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// 2. Desduplicació i processament IA amb Groq
	emit(ProgressEvent{
		Type:          "progress",
		Message:       fmt.Sprintf("🔍 Desduplicant i filtrant %d senyals bruts...", len(allRawSignals)),
		Progress:      55,
		EstimatedSecs: 10,
	})

	var pendingAnalysis []RawSignal
	alreadyExisted := 0
	for _, raw := range allRawSignals {
		if raw.SourceURL == "" {
			continue
		}
		exists, err := s.repo.ExistsByURL(ctx, raw.SourceURL)
		if err == nil && !exists {
			pendingAnalysis = append(pendingAnalysis, raw)
		} else {
			alreadyExisted++
		}
	}

	log.Printf("[Pipeline] Resum desduplicació: %d totals, %d ja existents a BD, %d nous candidats per analitzar", len(allRawSignals), alreadyExisted, len(pendingAnalysis))

	totalAnalyzed := 0
	totalToAnalyze := len(pendingAnalysis)
	if totalToAnalyze > (dailyLimit - signalsToday) {
		totalToAnalyze = dailyLimit - signalsToday
		log.Printf("[Pipeline] Ajustat per límit diari restant: analitzarem %d senyals", totalToAnalyze)
	}

	for i := 0; i < totalToAnalyze; i++ {
		if ctx.Err() != nil {
			break
		}
		rawSig := pendingAnalysis[i]

		// Desar el senyal a la BD
		signalID, err := s.repo.SaveRawSignal(ctx, &rawSig)
		if err != nil {
			log.Printf("[Pipeline] ⚠️ Error desant senyal a BD: %v", err)
			continue
		}

		log.Printf("[Pipeline IA %d/%d] 🧠 Analitzant senyal [ID=%s] [%s]: %q (Font: %s)", i+1, totalToAnalyze, signalID, rawSig.SignalType, rawSig.Title, rawSig.Source)

		analyzingProgress := 55 + int((float64(i+1)/float64(totalToAnalyze))*40.0) // 55% .. 95%
		emit(ProgressEvent{
			Type:          "analyzing",
			Message:       fmt.Sprintf("🧠 Analitzant amb IA Groq (%d de %d): %s", i+1, totalToAnalyze, truncateStr(rawSig.Title, 45)),
			Progress:      analyzingProgress,
			EstimatedSecs: (totalToAnalyze - i) * 2,
			FunFact:       entertainmentPool[rand.Intn(len(entertainmentPool))].text,
		})

		// Anàlisi IA amb fallback automàtic de models
		analysis, err := s.analyzeWithGroq(ctx, rawSig.Title, rawSig.RawText, rawSig.SignalType)
		if err != nil {
			log.Printf("[Pipeline IA %d/%d] ⚠️ Avis Groq: %v. Usant anàlisi de suport heurístic.", i+1, totalToAnalyze, err)
			analysis = s.fallbackAnalysis(rawSig.Title, rawSig.RawText)
		}

		log.Printf("[Pipeline IA %d/%d] 💡 Resultat: Score=%d/5 | Dolor=%q | Micro-SaaS=%q | Decisor=%q",
			i+1, totalToAnalyze, analysis.ViabilitatPLGScore, analysis.IneficienciaManual, analysis.PropostaMicroSaas, analysis.DecisorCompra)

		analysis.SignalID = signalID
		if err := s.repo.SaveOpportunityAnalysis(ctx, analysis); err != nil {
			log.Printf("[Pipeline IA %d/%d] ⚠️ Error desant anàlisi a BD: %v", i+1, totalToAnalyze, err)
		} else {
			totalAnalyzed++
		}
	}

	result := &ScraperRunResult{
		Success:        true,
		NewOffersFound: totalFound,
		AnalyzedCount:  totalAnalyzed,
		Message:        fmt.Sprintf("Rastreig completat. S'han trobat %d senyals i generat %d noves oportunitats Micro-SaaS.", totalFound, totalAnalyzed),
	}

	log.Printf("[Pipeline] === FI D'EXECUCIÓ: %s ===", result.Message)

	emit(ProgressEvent{
		Type:     "completed",
		Message:  result.Message,
		Progress: 100,
		Result:   result,
	})

	return result, nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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

	systemPrompt := `Ets un analista d'oportunitats de negoci Micro-SaaS expert en el teixit empresarial de Catalunya (PIMEs, tallers, comerços, logística i autònoms).
La teva missió és analitzar ofertes de feina i debats de fòrums per detectar ineficiències de gestió reals (Excel caòtics, gestió de torns, comunicació dispersa per WhatsApp, control d'estocs, fulls de ruta o tasques repetitives) que representin una autèntica OPORTUNITAT DE NEGOCI B2B resoluble amb un Micro-SaaS d'una sola funció (<100€/mes).

FILTRE DE RELEVÀNCIA DE NEGOCI:
- Si el text és una queixa domèstica d'un particular (turisme, transport públic com Renfe/Metro, política, etc.) o no té cap relació amb la gestió d'un negoci/PIME, descarta'l assignant viabilitat_plg_score = 1 i ineficiencia_manual = "No és una oportunitat de negoci B2B".
- Enfoca la solució a resoldre el dolor operatiu del negoci de forma 100% digital i autònoma.

PROHIBICIÓ ESTRICTA: No proposis MAI cap solució basada en OCR (Reconeixement Òptic de Caràcters), escaneig de documents físics o processament automàtic de factures en paper. Busca exclusivament solucions de software basades en formularis digitals purs, portals de dades o micro-SaaS on l'usuari teclegi o seleccioni la informació directament des de zero.

Retorna ÚNICAMENT un objecte JSON amb aquests camps exactes:
{
  "ineficiencia_manual": "Quina ineficiència de gestió o procés manual pateix el negoci",
  "proposta_micro_saas": "Nom i descripció de la solució autònoma d'una sola funció (<100€/mes)",
  "viabilitat_plg_score": 1-5 (5 = oportunitat clara 100% self-onboarding autònom; 1 = no és negoci o requereix integració complexa),
  "decisor_compra": "Càrrec a qui li fa mal el problema (ex: Cap de planta, Gerent, Propietari de negoci, Cap de taller)",
  "ganxo_venda": "Frase curta i directa per a contactar el negoci destacant el dolor i la solució"
}`

	userPrompt := fmt.Sprintf("Tipus de senyal: %s\nTítol: %s\n\nText:\n%s", signalType, title, text)
	candidateModels := getCandidateGroqModels()
	var lastErr error

	for _, modelName := range candidateModels {
		payload := GroqChatRequest{
			Model: modelName,
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
			lastErr = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("model %s no disponible a Groq (404), provant següent", modelName)
			log.Printf("Avis Groq: %v", lastErr)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
			log.Printf("Avis Groq: %v", lastErr)
			continue
		}

		var chatResp GroqChatResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			lastErr = err
			continue
		}

		if len(chatResp.Choices) == 0 {
			lastErr = errors.New("resposta buida de Groq")
			continue
		}

		rawJSON := chatResp.Choices[0].Message.Content
		var analysis OpportunityAnalysis
		if err := json.Unmarshal([]byte(rawJSON), &analysis); err != nil {
			lastErr = fmt.Errorf("error parsejant JSON de Groq: %w", err)
			continue
		}

		if analysis.ViabilitatPLGScore < 1 {
			analysis.ViabilitatPLGScore = 1
		} else if analysis.ViabilitatPLGScore > 5 {
			analysis.ViabilitatPLGScore = 5
		}

		analysis.RawLLMResponse = json.RawMessage(rawJSON)
		analysis.AnalyzedAt = time.Now()

		// Record working model
		groqModelMutex.Lock()
		activeGroqModel = modelName
		groqModelMutex.Unlock()

		return &analysis, nil
	}

	return nil, fmt.Errorf("no s'ha pogut obtenir resposta de cap model Groq: %w", lastErr)
}

func (s *Service) fallbackAnalysis(title, text string) *OpportunityAnalysis {
	textLower := strings.ToLower(title + " " + text)

	score := 4
	ineficiencia := "Tasca manual repetitiva gestionada amb fulls de càlcul o coordinació manual."
	proposta := "Micro-SaaS d'automatització directa mitjançant formulari web d'un sol flux."
	decisor := "Gerent / Responsable d'Operacions"
	ganxo := "He vist la vostra publicació. Tenim una eina web que estandarditza aquest flux en 2 minuts des de zero."

	if strings.Contains(textLower, "quadrant") || strings.Contains(textLower, "torn") {
		score = 5
		ineficiencia = "Planificació de quadrants de torns rotatius en fulls Excel i avisos dispersos per WhatsApp."
		proposta = "QuadrantBot: Portal web per a generació i selecció interactiva de torns mensuals amb notificació als treballadors."
		decisor = "Cap de Planta / Producció"
		ganxo = "Tenim una eina web directa per configurar els quadrants dels operaris sense obrir cap Excel."
	} else if strings.Contains(textLower, "albar") || strings.Contains(textLower, "transport") || strings.Contains(textLower, "ruta") {
		score = 5
		ineficiencia = "Gestió manual de fulls de ruta i albarans en paper amb re-entrada de dades a l'oficina."
		proposta = "RutaDirect: Formulari mòbil web per a xofers per registrar entregues i signatures en temps real des de zero."
		decisor = "Cap de Trànsit / Logística"
		ganxo = "Voleu que els xofers introdueixin les dades d'entrega directament des del mòbil eliminant el paper a l'oficina?"
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
