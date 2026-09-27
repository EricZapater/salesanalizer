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

	activeGroqModel = "openai/gpt-oss-120b"
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
		"openai/gpt-oss-120b",
		"openai/gpt-oss-20b",
		"qwen/qwen3.8-27b",
		"allam-2-7b",
		"llama-3.3-70b-versatile",
		"llama-3.1-8b-instant",
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

func (s *Service) ListEvidences(ctx context.Context, limit, offset int) ([]Evidence, int, error) {
	if s.repo == nil {
		return []Evidence{}, 0, nil
	}
	return s.repo.ListEvidences(ctx, limit, offset)
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

		// Flux V2: Ingestió d'evidència, normalització de procés i clustering de dolor
		ev, err := s.IngestEvidence(ctx, &rawSig)
		if err != nil {
			log.Printf("[Pipeline V2 %d/%d] ⚠️ Error processant evidència: %v", i+1, totalToAnalyze, err)
		} else if ev != nil && ev.IsDuplicateOf == nil {
			cluster, cErr := s.ProcessEvidenceIntoCluster(ctx, ev)
			if cErr != nil {
				log.Printf("[Pipeline V2 %d/%d] ⚠️ Error assignant a clúster: %v", i+1, totalToAnalyze, cErr)
			} else if cluster != nil && cluster.Status == "consolidated" {
				// Síntesi automàtica d'oportunitat si el clúster s'ha consolidat
				_, _ = s.SynthesizeOpportunityForCluster(ctx, cluster.ID)
			}
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

	// Ingestió i clustering V2
	ev, err := s.IngestEvidence(ctx, &rawSig)
	if err == nil && ev != nil && ev.IsDuplicateOf == nil {
		cluster, cErr := s.ProcessEvidenceIntoCluster(ctx, ev)
		if cErr == nil && cluster != nil && cluster.Status == "consolidated" {
			_, _ = s.SynthesizeOpportunityForCluster(ctx, cluster.ID)
		}
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

func (s *Service) AnalyzeSignal(ctx context.Context, title, text, signalType string) (*OpportunityAnalysis, error) {
	return s.analyzeWithGroq(ctx, title, text, signalType)
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
	return &OpportunityAnalysis{
		IneficienciaManual: "Pendent d'anàlisi detallada (resposta automàtica no concloent).",
		PropostaMicroSaas:  "Revisió manual necessària de la publicació.",
		ViabilitatPLGScore: 1,
		DecisorCompra:      "Gerent / Responsable d'Operacions",
		GanxoVenda:         "Contacte directe per conèixer els processos de gestió de l'empresa.",
		RawLLMResponse:     json.RawMessage(`{"fallback": true}`),
		AnalyzedAt:         time.Now(),
	}
}

// ExtractProcessWithGroq implementa l'Etapa 1 (Evidence Extraction) mitjançant el Prompt 5.1
func (s *Service) ExtractProcessWithGroq(ctx context.Context, rawContent, signalType, title string) (*ProcessExtractionLLMResult, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un analista d'operacions i processos de negoci expert en PIMEs de Catalunya.
La teva missió és extreure ÚNICAMENT els fets observats (procés de negoci, eines manuals utilitzades, freqüència i descripció factual de la tasca) a partir del text proporcionat.

PROHIBICIÓ ESTRICTA: No proposis MAI cap solució, ni cap idea de producte, ni cap Micro-SaaS, ni preus. Només extreu els fets del procés tal com s'observen al text.

REGLES PER A LA CITA LITERAL:
- Camp 'source_evidence_quote': Has d'extreure una frase o fragment curt (<200 caràcters) extret LITERALMENT del text original que demostri l'ús d'eines manuals, fulls de càlcul, paper o la freqüència de la tasca.
- No resumeixis ni parafrasegis la cita: ha de coincidir literalment amb el text d'origen.

Retorna ÚNICAMENT un objecte JSON amb aquests camps exactes:
{
  "extracted_process": "Nom curt del procés (ex: Gestió de quadrants de torns, Control d'albarans de repartiment, Re-entrada de comandes, Inventari de magatzem)",
  "task_description": "Descripció concisa i factual de la tasca operativa descrita al text",
  "frequency": "diària" | "setmanal" | "mensual" | "puntual" | "unknown",
  "manuality_score": 0-3 (0 = 100% digital/automàtic, 1 = eina digital bàsica, 2 = manualitat parcial/Excel/WhatsApp/re-entrada, 3 = 100% manual/paper/albarans físics),
  "tools_mentioned": ["Excel", "WhatsApp", "paper", "albarans"],
  "sector": "Sector o indústria del negoci (ex: Logística, Hostaleria, Taller / Indústria, Construcció, Comerç)",
  "evidence_type": "oferta_feina" | "queixa_forum" | "licitacio" | "ressenya" | "altre",
  "source_evidence_quote": "Fragment literal de <200 caràcters present al text"
}`

	userPrompt := fmt.Sprintf("Tipus de senyal: %s\nTítol: %s\n\nText:\n%s", signalType, title, rawContent)
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
			Temperature:    0.1,
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

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
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
		var extraction ProcessExtractionLLMResult
		if err := json.Unmarshal([]byte(rawJSON), &extraction); err != nil {
			lastErr = fmt.Errorf("error parsejant JSON d'extracció de procés: %w", err)
			continue
		}

		if extraction.ManualityScore < 0 {
			extraction.ManualityScore = 0
		} else if extraction.ManualityScore > 3 {
			extraction.ManualityScore = 3
		}

		// Record working model
		groqModelMutex.Lock()
		activeGroqModel = modelName
		groqModelMutex.Unlock()

		return &extraction, nil
	}

	return nil, fmt.Errorf("no s'ha pogut obtenir resposta de cap model Groq per a l'extracció de procés: %w", lastErr)
}

// FallbackProcessExtraction heurístic basat en paraules clau
func (s *Service) FallbackProcessExtraction(rawContent, signalType, title string) *ProcessExtractionLLMResult {
	lower := strings.ToLower(rawContent + " " + title)
	tools := make([]string, 0)
	score := 1
	quote := ""

	toolKeywords := []struct {
		kw   string
		name string
	}{
		{"excel", "Excel"},
		{"sheets", "Google Sheets"},
		{"whatsapp", "WhatsApp"},
		{"albarans", "Albarans en paper"},
		{"albaran", "Albarans en paper"},
		{"paper", "Documents en paper"},
		{"quadrant", "Quadrants manuals"},
		{"cuadrante", "Quadrants manuals"},
	}

	for _, item := range toolKeywords {
		if strings.Contains(lower, item.kw) {
			tools = append(tools, item.name)
			score = 2
		}
	}

	if strings.Contains(lower, "paper") || strings.Contains(lower, "albarans") || strings.Contains(lower, "albaran") {
		score = 3
	}

	// Buscar una frase curta com a cita
	sentences := strings.Split(rawContent, ".")
	for _, sent := range sentences {
		sTrim := strings.TrimSpace(sent)
		if len([]rune(sTrim)) > 15 && len([]rune(sTrim)) <= 180 {
			for _, item := range toolKeywords {
				if strings.Contains(strings.ToLower(sTrim), item.kw) {
					quote = sTrim
					break
				}
			}
			if quote != "" {
				break
			}
		}
	}

	if quote == "" && len([]rune(title)) <= 180 {
		quote = title
	}

	proc := "Gestió operativa manual"
	if len(tools) > 0 {
		proc = fmt.Sprintf("Gestió operativa amb %s", tools[0])
	}

	return &ProcessExtractionLLMResult{
		ExtractedProcess:    proc,
		TaskDescription:     "Tasca de suport administratiu i gestió de processos operatius",
		Frequency:           "diària",
		ManualityScore:      score,
		ToolsMentioned:      tools,
		Sector:              "Serveis / Indústria",
		EvidenceType:        signalType,
		SourceEvidenceQuote: quote,
	}
}

// IngestEvidence normalitza, deduplica i extreu els fets empírics d'una observació
func (s *Service) IngestEvidence(ctx context.Context, rawSig *RawSignal) (*Evidence, error) {
	normURL, err := NormalizeURL(rawSig.SourceURL)
	if err != nil {
		normURL = rawSig.SourceURL
	}

	contentHash := CalculateContentHash(rawSig.RawText)

	// Comprovar deduplicació a BD
	existing, err := s.repo.FindEvidenceByHashOrURL(ctx, normURL, contentHash)
	if err != nil {
		return nil, fmt.Errorf("error comprovant duplicats a BD: %w", err)
	}

	var isDuplicateOf *string
	if existing != nil {
		isDuplicateOf = &existing.ID
		log.Printf("[Pipeline Dedup] Senyal duplicat detectat per URL/Hash (ja existeix ID=%s)", existing.ID)
	}

	// Etapa 1: Extracció de fets amb Groq
	var extraction *ProcessExtractionLLMResult
	if isDuplicateOf == nil {
		extraction, err = s.ExtractProcessWithGroq(ctx, rawSig.RawText, rawSig.SignalType, rawSig.Title)
		if err != nil {
			log.Printf("[Pipeline Stage 1] Avis Groq: %v. Usant extracció heurística de suport.", err)
			extraction = s.FallbackProcessExtraction(rawSig.RawText, rawSig.SignalType, rawSig.Title)
		}
	} else {
		// Per a duplicats, reutilitzar l'extracció existent o heurística sense cridar LLM
		extraction = &ProcessExtractionLLMResult{
			ExtractedProcess:    derefString(existing.ExtractedProcess),
			TaskDescription:     derefString(existing.TaskDescription),
			Frequency:           existing.Frequency,
			ManualityScore:      existing.ManualityScore,
			ToolsMentioned:      existing.ToolsMentioned,
			Sector:              derefString(existing.Sector),
			EvidenceType:        derefString(existing.EvidenceType),
			SourceEvidenceQuote: derefString(existing.SourceEvidenceQuote),
		}
	}

	// Validació estricta de cita literal (<200 caràcters i present al text)
	confidence := "mitja"
	quoteValid := VerifyEvidenceQuote(rawSig.RawText, extraction.SourceEvidenceQuote)
	if !quoteValid {
		confidence = "baixa"
		log.Printf("[Pipeline Stage 1] ⚠️ Cita no verificada literalment al text font: %q (Confidence degradada a 'baixa')", extraction.SourceEvidenceQuote)
	} else if extraction.ManualityScore >= 2 {
		confidence = "alta"
	}

	var compPtr *string
	if rawSig.CompanyName != nil && *rawSig.CompanyName != "" {
		compPtr = rawSig.CompanyName
	}

	ev := &Evidence{
		RawContent:          rawSig.RawText,
		NormalizedURL:       normURL,
		ContentHash:         contentHash,
		Source:              rawSig.Source,
		AuthorOrCompany:     compPtr,
		ExtractedProcess:    &extraction.ExtractedProcess,
		TaskDescription:     &extraction.TaskDescription,
		Frequency:           extraction.Frequency,
		ManualityScore:      extraction.ManualityScore,
		ToolsMentioned:      extraction.ToolsMentioned,
		Sector:              &extraction.Sector,
		EvidenceType:        &extraction.EvidenceType,
		SourceEvidenceQuote: &extraction.SourceEvidenceQuote,
		EvidenceConfidence:  confidence,
		IsDuplicateOf:       isDuplicateOf,
	}

	if err := s.repo.SaveEvidence(ctx, ev); err != nil {
		return nil, fmt.Errorf("error desant evidència a BD: %w", err)
	}

	log.Printf("[Pipeline Stage 1] ✅ Evidència desada [ID=%s]: Procés=%q | Manualitat=%d/3 | Confiança=%s | Eines=%v",
		ev.ID, *ev.ExtractedProcess, ev.ManualityScore, ev.EvidenceConfidence, ev.ToolsMentioned)

	return ev, nil
}

// NormalizeProcessWithGroq implementa l'Etapa 2 (Process Normalization) mitjançant el Prompt 5.2
func (s *Service) NormalizeProcessWithGroq(ctx context.Context, ev *Evidence, existing []ProcessNormalized) (*NormalizationLLMResult, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un arquitecte de processos de negoci especialitzat en la taxonomia operativa de PIMEs catalanes.
La teva missió és normalitzar el procés observat en una evidència cap a un procés de negoci canònic estàndard.

CATEGORIES DISPONIBLES:
- "Logística & Repartiment"
- "Planificació de Torns & RRHH"
- "Facturació & Tresoreria"
- "Magatzem & Inventari"
- "Compres & Proveïdors"
- "Atenció al Client & Vendes"
- "Manteniment & SAT"
- "Gestió Documental & Administració"

INSTRUCCIONS:
1. Revisa la llista de processos canònics existents. Si el procés observat resol el mateix workflow de negoci, utilitza exactament el mateix 'canonical_process_name'.
2. Si és un procés nou, crea un nom canònic clar i concís en català (ex: "Control de fulls de ruta i albarans de lliurament", "Planificació de quadrants de torns rotatius").
3. No creïs variacions lleugeres de sinònims si ja existeix un procés equivalent.

Retorna ÚNICAMENT un objecte JSON amb aquests camps:
{
  "canonical_process_name": "Nom canònic del procés",
  "category": "Una de les categories disponibles",
  "typical_tools": ["Excel", "WhatsApp", "paper"],
  "process_description": "Breu descripció estàndard del workflow del procés",
  "confidence": "alta" | "mitja" | "baixa"
}`

	existingJSON, _ := json.Marshal(existing)
	userPrompt := fmt.Sprintf("Processos canònics existents a la base de dades:\n%s\n\nEvidència a normalitzar:\n- Procés observat: %s\n- Tasca: %s\n- Eines: %v\n- Sector: %s",
		string(existingJSON), derefString(ev.ExtractedProcess), derefString(ev.TaskDescription), ev.ToolsMentioned, derefString(ev.Sector))

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
			Temperature:    0.1,
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

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
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

		var normRes NormalizationLLMResult
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &normRes); err != nil {
			lastErr = fmt.Errorf("error parsejant JSON de normalització: %w", err)
			continue
		}

		if normRes.CanonicalProcessName == "" {
			normRes.CanonicalProcessName = derefString(ev.ExtractedProcess)
		}
		if normRes.Category == "" {
			normRes.Category = "Gestió Documental & Administració"
		}

		return &normRes, nil
	}

	return nil, fmt.Errorf("error obtenint resposta de normalització: %w", lastErr)
}

// FallbackProcessNormalization aplicat quan no hi ha connexió a Groq
func (s *Service) FallbackProcessNormalization(ev *Evidence) *NormalizationLLMResult {
	proc := strings.ToLower(derefString(ev.ExtractedProcess) + " " + derefString(ev.TaskDescription))

	category := "Gestió Documental & Administració"
	canonical := "Gestió de tasques administratives i arxiu"

	if strings.Contains(proc, "torn") || strings.Contains(proc, "quadrant") || strings.Contains(proc, "horari") {
		category = "Planificació de Torns & RRHH"
		canonical = "Planificació de quadrants i torns de treball"
	} else if strings.Contains(proc, "albarà") || strings.Contains(proc, "albaran") || strings.Contains(proc, "ruta") || strings.Contains(proc, "repart") {
		category = "Logística & Repartiment"
		canonical = "Control de fulls de ruta i albarans de lliurament"
	} else if strings.Contains(proc, "estoc") || strings.Contains(proc, "inventari") || strings.Contains(proc, "magatzem") {
		category = "Magatzem & Inventari"
		canonical = "Control d'estocs i inventari de magatzem"
	} else if strings.Contains(proc, "factur") || strings.Contains(proc, "cobrament") || strings.Contains(proc, "comptab") {
		category = "Facturació & Tresoreria"
		canonical = "Conciliació de factures i cobraments pendents"
	} else if strings.Contains(proc, "comanda") || strings.Contains(proc, "pressupost") {
		category = "Atenció al Client & Vendes"
		canonical = "Seguiment de comandes i pressupostos de clients"
	}

	return &NormalizationLLMResult{
		CanonicalProcessName: canonical,
		Category:             category,
		TypicalTools:         ev.ToolsMentioned,
		ProcessDescription:   fmt.Sprintf("Procés operatiu de %s", canonical),
		Confidence:           "mitja",
	}
}

// ClusterEvidenceWithGroq implementa l'Etapa 3 (Pain Clustering Confirmation) mitjançant el Prompt 5.3
func (s *Service) ClusterEvidenceWithGroq(ctx context.Context, ev *Evidence, candidateClusters []PainCluster) (*ClusteringDecisionLLMResult, error) {
	if len(candidateClusters) == 0 {
		return &ClusteringDecisionLLMResult{
			Action:             "create_new",
			NewClusterTitle:    fmt.Sprintf("Dolor operatiu: %s", derefString(ev.ExtractedProcess)),
			NewClusterSummary:  derefString(ev.TaskDescription),
			RelevanceScore:     1.0,
			Reasoning:          "Primer dolor registrat per a aquest procés canònic.",
		}, nil
	}

	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un analista d'agrupació de problemes (Pain Clustering).
La teva missió és decidir si una nova evidència observada comparteix exactament el mateix dolor operatiu que algun dels clústers existents, o si representa un dolor diferent i requereix crear un clúster nou.

CRITERIS DE FUSIÓ ('join_existing'):
- Comparteixen el mateix coll d'ampolla operatiu (ex: ambdós pateixen per actualitzar Excels de torns a mà quan hi ha baixes).
- No importa si són empreses o sectors diferents: si el dolor estructural és el mateix, s'han d'agrupar.

CRITERIS DE NOU CLÚSTER ('create_new'):
- El dolor o flux operatiu és substancialment diferent encara que pertanyi a la mateixa àrea.

Retorna ÚNICAMENT un objecte JSON:
{
  "action": "join_existing" | "create_new",
  "target_cluster_id": "UUID del clúster existent si action és join_existing",
  "new_cluster_title": "Títol concís del dolor si action és create_new",
  "new_cluster_summary": "Resum del dolor comú si action és create_new",
  "relevance_score": 0.0 - 1.0,
  "reasoning": "Explicació breu de la decisió d'agrupació"
}`

	clustersJSON, _ := json.Marshal(candidateClusters)
	userPrompt := fmt.Sprintf("Clústers existents per a aquest procés:\n%s\n\nNova evidència:\n- Procés: %s\n- Tasca: %s\n- Eines: %v\n- Cita literal: %q\n- Sector: %s",
		string(clustersJSON), derefString(ev.ExtractedProcess), derefString(ev.TaskDescription), ev.ToolsMentioned, derefString(ev.SourceEvidenceQuote), derefString(ev.Sector))

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
			Temperature:    0.1,
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

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
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

		var dec ClusteringDecisionLLMResult
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &dec); err != nil {
			lastErr = fmt.Errorf("error parsejant decisió de clustering: %w", err)
			continue
		}

		if dec.RelevanceScore <= 0 {
			dec.RelevanceScore = 1.0
		}

		return &dec, nil
	}

	return nil, fmt.Errorf("error a decisió de clustering amb Groq: %w", lastErr)
}

// FallbackClusteringDecision decisió basada en similitud de text
func (s *Service) FallbackClusteringDecision(ev *Evidence, candidateClusters []PainCluster) *ClusteringDecisionLLMResult {
	if len(candidateClusters) == 0 {
		return &ClusteringDecisionLLMResult{
			Action:             "create_new",
			NewClusterTitle:    fmt.Sprintf("Dolor operatiu: %s", derefString(ev.ExtractedProcess)),
			NewClusterSummary:  derefString(ev.TaskDescription),
			RelevanceScore:     1.0,
			Reasoning:          "Creació inicial de clúster per a aquest procés.",
		}
	}

	evText := derefString(ev.ExtractedProcess) + " " + derefString(ev.TaskDescription)
	bestScore := 0.0
	var bestCluster *PainCluster

	for i := range candidateClusters {
		c := &candidateClusters[i]
		clusterText := c.Title + " " + c.Summary
		sim := CalculateWordOverlapSimilarity(evText, clusterText)
		if sim > bestScore {
			bestScore = sim
			bestCluster = c
		}
	}

	if bestScore >= 0.15 && bestCluster != nil {
		return &ClusteringDecisionLLMResult{
			Action:          "join_existing",
			TargetClusterID: bestCluster.ID,
			RelevanceScore:  bestScore,
			Reasoning:       fmt.Sprintf("Similitud de contingut (%.2f) amb el clúster %s", bestScore, bestCluster.Title),
		}
	}

	return &ClusteringDecisionLLMResult{
		Action:             "create_new",
		NewClusterTitle:    fmt.Sprintf("Dolor operatiu: %s", derefString(ev.ExtractedProcess)),
		NewClusterSummary:  derefString(ev.TaskDescription),
		RelevanceScore:     1.0,
		Reasoning:          "Similitud baixa amb els clústers existents, es crea un nou clúster.",
	}
}

// ProcessEvidenceIntoCluster orquestra les Etapes 2 i 3 (Normalització de procés i Clúster de dolor)
func (s *Service) ProcessEvidenceIntoCluster(ctx context.Context, ev *Evidence) (*PainCluster, error) {
	if ev == nil || s.repo == nil {
		return nil, errors.New("evidència o repositori nul")
	}

	// 1. Etapa 2: Obtenir processos canònics i normalitzar
	canonicalList, err := s.repo.ListCanonicalProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("error obtenint processos canònics: %w", err)
	}

	normRes, err := s.NormalizeProcessWithGroq(ctx, ev, canonicalList)
	if err != nil {
		log.Printf("[Pipeline Stage 2] Avis Groq: %v. Usant normalització heurística.", err)
		normRes = s.FallbackProcessNormalization(ev)
	}

	proc, err := s.repo.FindOrCreateCanonicalProcess(ctx, normRes.CanonicalProcessName, normRes.Category, normRes.ProcessDescription, normRes.TypicalTools)
	if err != nil {
		return nil, fmt.Errorf("error guardant procés canònic: %w", err)
	}

	log.Printf("[Pipeline Stage 2] ✅ Procés normalitzat: %q [%s] (ID=%s)", proc.CanonicalName, proc.Category, proc.ID)

	// 2. Etapa 3: Cercar clústers candidats per a aquest procés
	candidateClusters, err := s.repo.FindClustersByProcess(ctx, proc.ID)
	if err != nil {
		return nil, fmt.Errorf("error cercant clústers per procés: %w", err)
	}

	clusterDec, err := s.ClusterEvidenceWithGroq(ctx, ev, candidateClusters)
	if err != nil {
		log.Printf("[Pipeline Stage 3] Avis Groq: %v. Usant decisió heurística de clustering.", err)
		clusterDec = s.FallbackClusteringDecision(ev, candidateClusters)
	}

	var targetClusterID string

	if clusterDec.Action == "join_existing" && clusterDec.TargetClusterID != "" {
		targetClusterID = clusterDec.TargetClusterID
		log.Printf("[Pipeline Stage 3] 🔗 Afegint evidència a clúster existent [ID=%s] (Score: %.2f)", targetClusterID, clusterDec.RelevanceScore)
	} else {
		// Crear nou clúster
		newCluster := &PainCluster{
			ProcessID:     proc.ID,
			Title:         clusterDec.NewClusterTitle,
			Summary:       clusterDec.NewClusterSummary,
			Status:        "emerging",
			EvidenceCount: 0,
		}
		if newCluster.Title == "" {
			newCluster.Title = fmt.Sprintf("Dolor en %s", proc.CanonicalName)
		}
		if newCluster.Summary == "" {
			newCluster.Summary = derefString(ev.TaskDescription)
		}

		if err := s.repo.SavePainCluster(ctx, newCluster); err != nil {
			return nil, fmt.Errorf("error creant nou clúster: %w", err)
		}
		targetClusterID = newCluster.ID
		log.Printf("[Pipeline Stage 3] 🆕 Creat nou clúster [ID=%s]: %q", targetClusterID, newCluster.Title)
	}

	// 3. Enllaçar evidència al clúster i recalcular mètriques
	if err := s.repo.LinkEvidenceToCluster(ctx, targetClusterID, ev.ID, clusterDec.RelevanceScore); err != nil {
		return nil, fmt.Errorf("error enllaçant evidència a clúster: %w", err)
	}

	if err := s.repo.RecalculateClusterMetrics(ctx, targetClusterID); err != nil {
		return nil, fmt.Errorf("error recalculant mètriques de clúster: %w", err)
	}

	clusterDetails, err := s.repo.GetClusterByID(ctx, targetClusterID)
	if err != nil {
		return nil, err
	}

	log.Printf("[Pipeline Stage 3] 📊 Clúster actualitzat [ID=%s]: %d evidències (%d empreses, %d fonts) | Estat=%s",
		targetClusterID, clusterDetails.EvidenceCount, clusterDetails.CompanyCount, clusterDetails.SourceCount, clusterDetails.Status)

	return &clusterDetails.PainCluster, nil
}

// ListPainClusters retorna clústers de dolor amb paginació
func (s *Service) ListPainClusters(ctx context.Context, status string, minEvidence int, limit, offset int) (*PainClusterListResponse, error) {
	if s.repo == nil {
		return &PainClusterListResponse{Total: 0, Items: []PainCluster{}}, nil
	}
	clusters, total, err := s.repo.ListPainClusters(ctx, status, minEvidence, limit, offset)
	if err != nil {
		return nil, err
	}
	return &PainClusterListResponse{
		Total: total,
		Items: clusters,
	}, nil
}

// GetPainClusterDetails retorna un clúster amb el detall complet d'evidències
func (s *Service) GetPainClusterDetails(ctx context.Context, id string) (*PainClusterWithDetails, error) {
	if s.repo == nil {
		return nil, errors.New("repositori no inicialitzat")
	}
	return s.repo.GetClusterByID(ctx, id)
}

// SynthesizeOpportunityWithGroq implementa l'Etapa 4 (Opportunity Synthesis) mitjançant el Prompt 5.4
func (s *Service) SynthesizeOpportunityWithGroq(ctx context.Context, cluster *PainClusterWithDetails) (*OpportunitySynthesisLLMResult, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un dissenyador de producte expert en Micro-SaaS B2B d'un sol flux per a PIMEs de Catalunya.
La teva missió és sintetitzar una proposta de solució Micro-SaaS altament enfocada a partir d'un clúster de dolor validat.

CRITERIS DE PRODUCTE NO NEGOCIABLES:
1. Un sol problema concret, un sol perfil d'usuari diari, un sol decisor de compra.
2. Aplicació web lleugera, 100% autònoma, self-onboarding en <5 minuts i 0 integracions inicials (no dependre d'APIs d'ERPs antics).
3. Preu orientatiu de 30-50 €/mes per negoci.
4. PROHIBICIÓ ESTRICTA: No proposis ERPs integrals, consultoria, maquinari, OCR d'escaneig de documents físics o IA costosa.

Retorna ÚNICAMENT un objecte JSON amb aquests camps:
{
  "title": "Nom curt del Micro-SaaS (ex: TornsDirect, RutaAlbarà, EstocSimple)",
  "target_user": "Perfil de l'empleat o usuari que utilitzarà l'eina cada dia",
  "buyer_persona": "Càrrec que té autoritat de compra directa per pagar 30-50€/mes",
  "core_workflow": "Descripció concisa del flux pas a pas d'un sol ús",
  "value_prop": "Proposta de valor quantificable (hores estalviades, 0 errors, menys estrès)",
  "pricing_model": "Preu mensual orientatiu (ex: 39€/mes per centre de treball)",
  "outreach_hook": "Frase curta d'obertura comercial per contactar negocis d'aquest sector"
}`

	evidencesSummary := make([]string, 0, len(cluster.Evidences))
	for _, ev := range cluster.Evidences {
		evidencesSummary = append(evidencesSummary, fmt.Sprintf("- Font (%s): %s | Cita: %q", ev.Source, derefString(ev.ExtractedProcess), derefString(ev.SourceEvidenceQuote)))
	}

	userPrompt := fmt.Sprintf("Clúster de Dolor:\nTítol: %s\nProcés Canònic: %s (%s)\nResum: %s\n\nEvidències i Cites Literals:\n%s",
		cluster.Title, cluster.ProcessName, cluster.Category, cluster.Summary, strings.Join(evidencesSummary, "\n"))

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

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
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

		var synth OpportunitySynthesisLLMResult
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &synth); err != nil {
			lastErr = fmt.Errorf("error parsejant síntesi d'oportunitat: %w", err)
			continue
		}

		return &synth, nil
	}

	return nil, fmt.Errorf("error a síntesi d'oportunitat amb Groq: %w", lastErr)
}

// FallbackOpportunitySynthesis genera una proposta Micro-SaaS de contingència
func (s *Service) FallbackOpportunitySynthesis(cluster *PainClusterWithDetails) *OpportunitySynthesisLLMResult {
	title := "Micro-SaaS " + cluster.ProcessName
	target := "Responsable d'equip / Administratiu"
	buyer := "Gerent / Propietari de PIME"
	hook := fmt.Sprintf("Simplifica el teu procés de %s eliminant fulls Excel manuals.", cluster.ProcessName)

	return &OpportunitySynthesisLLMResult{
		Title:        title,
		TargetUser:   target,
		BuyerPersona: buyer,
		CoreWorkflow: fmt.Sprintf("Formulari web àgil per gestionar %s en temps real sense paper ni fulls de càlcul.", cluster.ProcessName),
		ValueProp:    "Estalvia fins a 5 hores setmanals de gestió administrativa i evita errors d'actualització manual.",
		PricingModel: "39€/mes per negoci",
		OutreachHook: hook,
	}
}

// ScoreOpportunityWithGroq implementa l'Etapa 5 (Multidimensional Scoring) mitjançant el Prompt 5.5
func (s *Service) ScoreOpportunityWithGroq(ctx context.Context, opp *OpportunitySynthesisLLMResult, cluster *PainClusterWithDetails) (*ScoringDimensions, error) {
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		return nil, errors.New("GROQ_API_KEY no configurada")
	}

	systemPrompt := `Ets un comitè auditor d'inversió i viabilitat Micro-SaaS B2B.
La teva missió és avaluar de forma crítica i 100% objectiva la proposta de Micro-SaaS sobre 12 dimensions clau.

PER A CADA DIMENSIÓ HAS D'ASSIGNAR:
- "score": Valor sencer entre 0 i 5 (5 = excel·lent / màxim favor; 0 = inviable / desastrós).
- "justification": Breu explicació factual (1-2 frases) que justifiqui la puntuació basant-te en la mida de la PIME, naturalesa de la tasca i tecnologia.

LES 12 DIMENSIONS SÓN:
1. pain_intensity: Severitat de la pèrdua econòmica, hores perdudes o errors operatius.
2. urgency: Necessitat de resoldre-ho avui mateix vs projecte "nice-to-have".
3. budget_discretion: El decisor pot contractar-ho directament (<100€/mes) sense comitès de compra?
4. market_reach: Mida del mercat potencial de PIMEs a Catalunya i Espanya.
5. implementation_simplicity: Es pot programar un MVP funcional complet en 1-2 setmanes?
6. self_onboarding: L'usuari pot començar a utilitzar-lo en <5 minuts sense formació ni suport?
7. zero_integrations: Funciona de forma autònoma sense connectar-se a ERPs antics?
8. no_ocr_no_hardware: 100% interfície digital, 0 dependència d'escàners, paper físic o OCR?
9. retention_stickiness: Genera ús recurrent diari o setmanal com a part del workflow central?
10. niche_competition: Hi ha un buit de solucions hiper-específiques assequibles per a PIMEs?
11. willingness_to_pay: El retorn de la inversió (estalvi de temps) supera clarament els 30-50€/mes?
12. scalability_reach: Replicable a altres sectors o regions amb mínims canvis?

Retorna ÚNICAMENT un objecte JSON amb l'objecte "scores" que contingui aquests 12 camps exactes.`

	oppJSON, _ := json.Marshal(opp)
	userPrompt := fmt.Sprintf("Proposta Micro-SaaS a auditar:\n%s\n\nClúster de Dolor d'origen:\n- Títol: %s\n- Categoria: %s\n- Recompte evidències: %d (%d empreses, %d fonts)\n- Resum: %s",
		string(oppJSON), cluster.Title, cluster.Category, cluster.EvidenceCount, cluster.CompanyCount, cluster.SourceCount, cluster.Summary)

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
			Temperature:    0.1,
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

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("groq status %d (%s): %s", resp.StatusCode, modelName, string(respBody))
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

		var scoringRes MultidimensionalScoringLLMResult
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &scoringRes); err != nil {
			lastErr = fmt.Errorf("error parsejant scoring multidimensional: %w", err)
			continue
		}

		return &scoringRes.Scores, nil
	}

	return nil, fmt.Errorf("error auditant scoring amb Groq: %w", lastErr)
}

// FallbackOpportunityScoring calcula una puntuació de suport realista
func (s *Service) FallbackOpportunityScoring(opp *OpportunitySynthesisLLMResult, cluster *PainClusterWithDetails) *ScoringDimensions {
	return &ScoringDimensions{
		PainIntensity:            FactorScore{Score: 4, Justification: "Procés manual repetitiu que consumeix temps d'equip setmanalment."},
		Urgency:                  FactorScore{Score: 3, Justification: "Incomoditat operativa constant resoluble a curt termini."},
		BudgetDiscretion:         FactorScore{Score: 5, Justification: "Preu de 39€/mes a l'abast directe del gerent o responsable de departament."},
		MarketReach:              FactorScore{Score: 4, Justification: "Milers de PIMEs catalanes operen en aquest sector."},
		ImplementationSimplicity: FactorScore{Score: 5, Justification: "Arquitectura web lleugera construïble ràpidament com a formulari interactiu."},
		SelfOnboarding:           FactorScore{Score: 4, Justification: "Configuració senzilla sense requerir integracions complexes."},
		ZeroIntegrations:         FactorScore{Score: 5, Justification: "Eina 100% autònoma sense dependència d'altres sistemes ERP."},
		NoOCRNoHardware:          FactorScore{Score: 5, Justification: "Flux digitalitzat pur sense escàner físic ni OCR fràgil."},
		RetentionStickiness:      FactorScore{Score: 4, Justification: "Ús freqüent associat al cicle operatiu de l'empresa."},
		NicheCompetition:         FactorScore{Score: 4, Justification: "Pocs competidors enfocats estrictament en aquest micro-flux."},
		WillingnessToPay:         FactorScore{Score: 4, Justification: "El cost mensual és molt inferior al cost hora del personal dedicat a la tasca manual."},
		ScalabilityReach:         FactorScore{Score: 4, Justification: "Facilitat per expandir a altres sectors amb necessitats similars."},
	}
}

// SynthesizeOpportunityForCluster orquestra les Etapes 4 i 5 per a un clúster consolidat
func (s *Service) SynthesizeOpportunityForCluster(ctx context.Context, clusterID string) (*Opportunity, error) {
	if s.repo == nil {
		return nil, errors.New("repositori no inicialitzat")
	}

	clusterDetails, err := s.repo.GetClusterByID(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("error obtenint clúster: %w", err)
	}
	if clusterDetails == nil {
		return nil, fmt.Errorf("clúster amb ID %s no trobat", clusterID)
	}

	// 1. Etapa 4: Síntesi de la proposta Micro-SaaS
	synth, err := s.SynthesizeOpportunityWithGroq(ctx, clusterDetails)
	if err != nil {
		log.Printf("[Pipeline Stage 4] Avis Groq: %v. Usant síntesi de suport.", err)
		synth = s.FallbackOpportunitySynthesis(clusterDetails)
	}

	// 2. Etapa 5: Avaluació multidimensional dels 12 factors
	scores, err := s.ScoreOpportunityWithGroq(ctx, synth, clusterDetails)
	if err != nil {
		log.Printf("[Pipeline Stage 5] Avis Groq: %v. Usant scoring de suport.", err)
		scores = s.FallbackOpportunityScoring(synth, clusterDetails)
	}

	// 3. Càlcul determinista del GlobalScore i ViabilityTier
	globalScore, tier := CalculateGlobalScore(scores)

	opp := &Opportunity{
		ClusterID:     clusterID,
		Title:         synth.Title,
		TargetUser:    synth.TargetUser,
		BuyerPersona:  synth.BuyerPersona,
		CoreWorkflow:  synth.CoreWorkflow,
		ValueProp:     synth.ValueProp,
		PricingModel:  synth.PricingModel,
		OutreachHook:  synth.OutreachHook,
		Scores:        *scores,
		GlobalScore:   globalScore,
		ViabilityTier: tier,
	}

	if err := s.repo.SaveOpportunity(ctx, opp); err != nil {
		return nil, fmt.Errorf("error guardant oportunitat a BD: %w", err)
	}

	log.Printf("[Pipeline Stage 4&5] 🚀 Oportunitat generada [ID=%s]: %q | GlobalScore=%.2f/5.0 (%s)",
		opp.ID, opp.Title, opp.GlobalScore, opp.ViabilityTier)

	return s.repo.GetOpportunityByID(ctx, opp.ID)
}

// ListOpportunities retorna la llista d'oportunitats sintetitzades
func (s *Service) ListOpportunities(ctx context.Context, minScore float64, tier string, limit, offset int) (*OpportunityListResponse, error) {
	if s.repo == nil {
		return &OpportunityListResponse{Total: 0, Items: []Opportunity{}}, nil
	}
	opps, total, err := s.repo.ListOpportunities(ctx, minScore, tier, limit, offset)
	if err != nil {
		return nil, err
	}
	return &OpportunityListResponse{
		Total: total,
		Items: opps,
	}, nil
}

// GetOpportunityByID retorna una oportunitat pel seu ID
func (s *Service) GetOpportunityByID(ctx context.Context, id string) (*Opportunity, error) {
	if s.repo == nil {
		return nil, errors.New("repositori no inicialitzat")
	}
	return s.repo.GetOpportunityByID(ctx, id)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}




